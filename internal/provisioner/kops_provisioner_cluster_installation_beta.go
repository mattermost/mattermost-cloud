// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"

	"github.com/mattermost/mattermost-operator/pkg/client/clientset/versioned/typed/mattermost/v1alpha1"
	"github.com/mattermost/mattermost-operator/pkg/client/v1beta1/clientset/versioned/typed/mattermost/v1beta1"

	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	mmv1beta1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1beta1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type kopsCIBeta struct {
	*KopsProvisioner
}

// CreateClusterInstallation creates a Mattermost installation within the given cluster.
func (provisioner *kopsCIBeta) CreateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
		"version":      "v1beta1",
	})
	logger.Info("Creating cluster installation")

	configLocation, err := provisioner.getCachedKopsClusterKubecfg(cluster.ProvisionerMetadataKops.Name, logger)
	if err != nil {
		return errors.Wrap(err, "failed to get kops config from cache")
	}
	defer provisioner.invalidateCachedKopsClientOnError(err, cluster.ProvisionerMetadataKops.Name, logger)

	k8sClient, err := k8s.NewFromFile(configLocation, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client from file")
	}

	installationName, err := provisioner.prepareClusterInstallationEnv(clusterInstallation, k8sClient)
	if err != nil {
		return errors.Wrap(err, "failed to prepare cluster installation env")
	}

	mattermostEnv := getMattermostEnvWithOverrides(installation)

	mattermost := &mmv1beta1.Mattermost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      installationName,
			Namespace: clusterInstallation.Namespace,
			Labels:    generateClusterInstallationResourceLabels(installation, clusterInstallation),
		},
		Spec: mmv1beta1.MattermostSpec{
			Size:               installation.Size,
			Version:            translateMattermostVersion(installation.Version),
			Image:              installation.Image,
			IngressName:        installation.DNS,
			MattermostEnv:      mattermostEnv.ToEnvList(),
			UseIngressTLS:      false,
			IngressAnnotations: getIngressAnnotations(),
			// Set `installation-id` and `cluster-installation-id` labels for all related resources.
			ResourceLabels: clusterInstallationBaseLabels(installation, clusterInstallation),
			Scheduling: mmv1beta1.Scheduling{
				Affinity: generateAffinityConfig(installation, clusterInstallation),
			},
		},
	}

	if installation.State == model.InstallationStateHibernating {
		logger.Info("creating hibernated cluster installation")
		configureInstallationForHibernation(mattermost)
	}

	if installation.License != "" {
		licenseSecretName, err := prepareCILicenseSecret(installation, clusterInstallation, k8sClient)
		if err != nil {
			return errors.Wrap(err, "failed to prepare license secret")
		}

		mattermost.Spec.LicenseSecret = licenseSecretName
		logger.Debug("Cluster installation configured with a Mattermost license")
	}

	err = provisioner.ensureFilestoreAndDatabase(mattermost, installation, clusterInstallation, k8sClient, logger)
	if err != nil {
		return errors.Wrap(err, "failed to ensure database and filestore")
	}

	ctx := context.TODO()
	_, err = k8sClient.MattermostClientsetV1Beta.MattermostV1beta1().Mattermosts(clusterInstallation.Namespace).Create(ctx, mattermost, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create cluster installation")
	}

	logger.Info("Successfully created cluster installation")

	return nil
}

// HibernateClusterInstallation updates a cluster installation to consume fewer
// resources.
func (provisioner *kopsCIBeta) HibernateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	configLocation, err := provisioner.getCachedKopsClusterKubecfg(cluster.ProvisionerMetadataKops.Name, logger)
	if err != nil {
		return errors.Wrap(err, "failed to get kops config from cache")
	}
	defer provisioner.invalidateCachedKopsClientOnError(err, cluster.ProvisionerMetadataKops.Name, logger)

	k8sClient, err := k8s.NewFromFile(configLocation, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client from file")
	}

	ctx := context.TODO()
	name := makeClusterInstallationName(clusterInstallation)

	cr, err := k8sClient.MattermostClientsetV1Beta.MattermostV1beta1().Mattermosts(clusterInstallation.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to get cluster installation %s", clusterInstallation.ID)
	}

	configureInstallationForHibernation(cr)

	_, err = k8sClient.MattermostClientsetV1Beta.MattermostV1beta1().Mattermosts(clusterInstallation.Namespace).Update(ctx, cr, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to update cluster installation %s", clusterInstallation.ID)
	}

	logger.Info("Updated cluster installation")

	return nil
}

func configureInstallationForHibernation(mattermost *mmv1beta1.Mattermost) {
	// Hibernation is currently considered changing the Mattermost app deployment
	// to 0 replicas in the pod. i.e. Scale down to no Mattermost apps running.
	// The current way to do this is to set a negative replica count in the
	// k8s custom resource. Custom ingress annotations are also used.
	// TODO: enhance hibernation to include database and/or filestore.
	mattermost.Spec.Replicas = int32Ptr(0)
	mattermost.Spec.IngressAnnotations = getHibernatingIngressAnnotations()
	mattermost.Spec.Size = ""
}

// UpdateClusterInstallation updates the cluster installation spec to match the
// installation specification.
func (provisioner *kopsCIBeta) UpdateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	configLocation, err := provisioner.getCachedKopsClusterKubecfg(cluster.ProvisionerMetadataKops.Name, logger)
	if err != nil {
		return errors.Wrap(err, "failed to get kops config from cache")
	}
	defer provisioner.invalidateCachedKopsClientOnError(err, cluster.ProvisionerMetadataKops.Name, logger)

	k8sClient, err := k8s.NewFromFile(configLocation, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client from file")
	}

	installationName, err := provisioner.prepareClusterInstallationEnv(clusterInstallation, k8sClient)
	if err != nil {
		return errors.Wrap(err, "failed to prepare cluster installation env")
	}

	ctx := context.TODO()

	mattermost, err := k8sClient.MattermostClientsetV1Beta.MattermostV1beta1().Mattermosts(clusterInstallation.Namespace).Get(ctx, installationName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to get mattermost installation %s", clusterInstallation.ID)
	}

	logger.WithField("status", fmt.Sprintf("%+v", mattermost.Status)).Debug("Got mattermost installation")

	mattermost.ObjectMeta.Labels = generateClusterInstallationResourceLabels(installation, clusterInstallation)
	mattermost.Spec.ResourceLabels = clusterInstallationBaseLabels(installation, clusterInstallation)

	mattermost.Spec.Scheduling.Affinity = generateAffinityConfig(installation, clusterInstallation)

	version := translateMattermostVersion(installation.Version)
	if mattermost.Spec.Version == version {
		logger.Debugf("Mattermost installation already on version %s", version)
	} else {
		logger.Debugf("Mattermost installation version updated from %s to %s", mattermost.Spec.Version, installation.Version)
		mattermost.Spec.Version = version
	}

	if mattermost.Spec.Image == installation.Image {
		logger.Debugf("Mattermost installation already on image %s", installation.Image)
	} else {
		logger.Debugf("Mattermost installation image updated from %s to %s", mattermost.Spec.Image, installation.Image)
		mattermost.Spec.Image = installation.Image
	}

	// A few notes on installation sizing changes:
	//  - Resizing currently ignores the installation scheduling algorithm.
	//    There is no good interface to determine if the new installation
	//    size will safely fit on the cluster. This could, in theory, be done
	//    when the size request change comes in on the API, but would require
	//    new scheduling logic. For now, take care when resizing.
	//    TODO: address these issue.
	mattermost.Spec.Size = installation.Size // Appropriate replicas and resources will be set by Operator.

	mattermost.Spec.LicenseSecret = ""
	secretName := fmt.Sprintf("%s-license", installationName)
	if installation.License != "" {
		secretName, err = prepareCILicenseSecret(installation, clusterInstallation, k8sClient)
		if err != nil {
			return errors.Wrap(err, "failed to prepare license secret")
		}

		mattermost.Spec.LicenseSecret = secretName
	} else {
		err = cleanupOldLicenseSecrets("", clusterInstallation, k8sClient, logger)
		if err != nil {
			return errors.Wrap(err, "failed to cleanup old license secrets")
		}
	}

	err = provisioner.ensureFilestoreAndDatabase(mattermost, installation, clusterInstallation, k8sClient, logger)
	if err != nil {
		return errors.Wrap(err, "failed to ensure database and filestore")
	}

	mattermostEnv := getMattermostEnvWithOverrides(installation)
	mattermost.Spec.MattermostEnv = mattermostEnv.ToEnvList()

	mattermost.Spec.IngressAnnotations = getIngressAnnotations()

	_, err = k8sClient.MattermostClientsetV1Beta.MattermostV1beta1().Mattermosts(clusterInstallation.Namespace).Update(ctx, mattermost, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to update cluster installation %s", clusterInstallation.ID)
	}

	logger.Info("Updated cluster installation")

	return nil
}

// RefreshSecrets deletes old secrets for database and file store and replaces them with new ones.
func (provisioner *kopsCIBeta) RefreshSecrets(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})
	logger.Info("Refreshing secrets for cluster installation")

	k8sClient, invalidateCache, err := provisioner.k8sClient(cluster.ProvisionerMetadataKops.Name, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client")
	}
	defer invalidateCache(err)

	installationName := makeClusterInstallationName(clusterInstallation)

	ctx := context.TODO()
	mmClient := k8sClient.MattermostClientsetV1Beta.MattermostV1beta1().Mattermosts(clusterInstallation.Namespace)

	mattermost, err := mmClient.Get(ctx, installationName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to get mattermost installation %s", clusterInstallation.ID)
	}

	err = provisioner.deleteMMSecrets(clusterInstallation.Namespace, mattermost, k8sClient, logger)
	if err != nil {
		return errors.Wrap(err, "failed to delete old secrets")
	}

	err = provisioner.ensureFilestoreAndDatabase(mattermost, installation, clusterInstallation, k8sClient, logger)
	if err != nil {
		return errors.Wrap(err, "failed to ensure database and filestore")
	}

	_, err = mmClient.Update(ctx, mattermost, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to update mattermost CR %s", mattermost.Name)
	}

	logger.Info("Refreshed database and file store secrets")

	return nil
}

func (provisioner *kopsCIBeta) deleteMMSecrets(ns string, mattermost *mmv1beta1.Mattermost, kubeClient *k8s.KubeClient, logger log.FieldLogger) error {
	secretsClient := kubeClient.Clientset.CoreV1().Secrets(ns)

	if mattermost.Spec.Database.External != nil {
		err := secretsClient.Delete(context.Background(), mattermost.Spec.Database.External.Secret, metav1.DeleteOptions{})
		if err != nil {
			if !k8sErrors.IsNotFound(err) {
				return errors.Wrap(err, "failed to delete old database secret")
			}
			logger.Debug("Database secret does not exist, assuming already deleted")
		}
	}

	if mattermost.Spec.FileStore.External != nil {
		err := secretsClient.Delete(context.Background(), mattermost.Spec.FileStore.External.Secret, metav1.DeleteOptions{})
		if err != nil {
			if !k8sErrors.IsNotFound(err) {
				return errors.Wrap(err, "failed to delete old file store secret")
			}
			logger.Debug("File store secret does not exist, assuming already deleted")
		}
	}

	return nil
}

func (provisioner *kopsCIBeta) ensureFilestoreAndDatabase(
	mattermost *mmv1beta1.Mattermost,
	installation *model.Installation,
	clusterInstallation *model.ClusterInstallation,
	k8sClient *k8s.KubeClient,
	logger log.FieldLogger) error {

	databaseSecret, err := provisioner.resourceUtil.GetDatabaseForInstallation(installation).GenerateDatabaseSecret(provisioner.store, logger)
	if err != nil {
		return errors.Wrap(err, "failed to generate database configuration")
	}
	// If Secret is nil - the default will be used
	if databaseSecret != nil {
		_, err = k8sClient.CreateOrUpdateSecret(clusterInstallation.Namespace, databaseSecret)
		if err != nil {
			return errors.Wrapf(err, "failed to create the database secret %s/%s", clusterInstallation.Namespace, databaseSecret.Name)
		}
		mattermost.Spec.Database = mmv1beta1.Database{
			External: &mmv1beta1.ExternalDatabase{Secret: databaseSecret.Name},
		}
	}

	filestoreConfig, filestoreSecret, err := provisioner.resourceUtil.GetFilestore(installation).GenerateFilestoreSpecAndSecret(provisioner.store, logger)
	if err != nil {
		return errors.Wrap(err, "failed to generate filestore configuration")
	}
	if filestoreSecret != nil {
		_, err = k8sClient.CreateOrUpdateSecret(clusterInstallation.Namespace, filestoreSecret)
		if err != nil {
			return errors.Wrapf(err, "failed to create the filestore secret %s/%s", clusterInstallation.Namespace, filestoreSecret.Name)
		}
	}
	// If FilestoreConfig is nil - the default will be used
	if filestoreConfig != nil {
		mattermost.Spec.FileStore = mmv1beta1.FileStore{External: &mmv1beta1.ExternalFileStore{
			URL:    filestoreConfig.URL,
			Bucket: filestoreConfig.Bucket,
			Secret: filestoreConfig.Secret,
		}}
	}

	return nil
}

func (provisioner *kopsCIBeta) EnsureCRMigrated(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation) (bool, error) {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})
	logger.Info("Ensuring cluster installation migrated to v1beta")

	configLocation, err := provisioner.getCachedKopsClusterKubecfg(cluster.ProvisionerMetadataKops.Name, logger)
	if err != nil {
		return false, errors.Wrap(err, "failed to get kops config from cache")
	}
	defer provisioner.invalidateCachedKopsClientOnError(err, cluster.ProvisionerMetadataKops.Name, logger)

	k8sClient, err := k8s.NewFromFile(configLocation, logger)
	if err != nil {
		return false, errors.Wrap(err, "failed to create k8s client from file")
	}

	isMigrated, err := provisioner.migrateFromClusterInstallation(clusterInstallation, k8sClient, logger)
	if err != nil {
		return false, errors.Wrap(err, "failed to migrate ClusterInstallation to Mattermost")
	}

	return isMigrated, nil
}

// VerifyClusterInstallationMatchesConfig attempts to verify that a cluster
// installation custom resource matches the configuration that is defined in the
// provisioner
// NOTE: this does NOT ensure that other resources such as network policies for
// that namespace are correct. Also, the values checked are ONLY values that are
// defined by both the installation and group configuration.
func (provisioner *kopsCIBeta) VerifyClusterInstallationMatchesConfig(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) (bool, error) {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	logger.Info("Verifying cluster installation resource configuration")

	cr, err := provisioner.getMattermostCustomResource(cluster, clusterInstallation, logger)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get cluster installation %s", clusterInstallation.ID)
	}

	version := translateMattermostVersion(installation.Version)
	if cr.Spec.Version != version {
		logger.Debugf("Mattermost installation resource on version %s when expecting %s", cr.Spec.Version, version)
		return false, nil
	}

	if cr.Spec.Image != installation.Image {
		logger.Debugf("Mattermost installation resource on image %s when expecting %s", cr.Spec.Image, installation.Image)
		return false, nil
	}

	mattermostEnv := getMattermostEnvWithOverrides(installation)
	for _, wanted := range mattermostEnv.ToEnvList() {
		if !ensureEnvMatch(wanted, cr.Spec.MattermostEnv) {
			logger.Debugf("Mattermost installation resource couldn't find env match for %s", wanted.Name)
			return false, nil
		}
	}

	logger.Debug("Verified cluster installation config matches")

	return true, nil
}

// DeleteClusterInstallation deletes a Mattermost installation within the given cluster.
func (provisioner *kopsCIBeta) DeleteClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	configLocation, err := provisioner.getCachedKopsClusterKubecfg(cluster.ProvisionerMetadataKops.Name, logger)
	if err != nil {
		return errors.Wrap(err, "failed to get kops config from cache")
	}
	defer provisioner.invalidateCachedKopsClientOnError(err, cluster.ProvisionerMetadataKops.Name, logger)

	k8sClient, err := k8s.NewFromFile(configLocation, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client from file")
	}

	name := makeClusterInstallationName(clusterInstallation)

	ctx := context.TODO()

	err = k8sClient.MattermostClientsetV1Beta.MattermostV1beta1().Mattermosts(clusterInstallation.Namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if k8sErrors.IsNotFound(err) {
		logger.Warnf("Cluster installation %s not found, assuming already deleted", name)
	} else if err != nil {
		return errors.Wrapf(err, "failed to delete cluster installation %s", clusterInstallation.ID)
	}

	if installation.License != "" {
		err = cleanupOldLicenseSecrets("", clusterInstallation, k8sClient, logger)
		if err != nil {
			return errors.Wrap(err, "failed to delete license secret")
		}
	}

	err = k8sClient.Clientset.CoreV1().Namespaces().Delete(ctx, clusterInstallation.Namespace, metav1.DeleteOptions{})
	if k8sErrors.IsNotFound(err) {
		logger.Warnf("Namespace %s not found, assuming already deleted", clusterInstallation.Namespace)
	} else if err != nil {
		return errors.Wrapf(err, "failed to delete namespace %s", clusterInstallation.Namespace)
	}

	logger.Info("Successfully deleted cluster installation")

	return nil
}

// IsResourceReady checks if the ClusterInstallation Custom Resource is ready on the cluster.
func (provisioner *kopsCIBeta) IsResourceReady(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation) (bool, error) {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	cr, err := provisioner.getMattermostCustomResource(cluster, clusterInstallation, logger)
	if err != nil {
		return false, errors.Wrap(err, "failed to get ClusterInstallation Custom Resource")
	}

	if cr.Status.State != mmv1beta1.Stable {
		return false, nil
	}
	if cr.Status.ObservedGeneration != 0 {
		if cr.Generation != cr.Status.ObservedGeneration {
			return false, nil
		}
	} else {
		// The new ObservedGeneration check is not supported because the operator
		// has not yet been updated. Log this and fall back to original check.
		// TODO: remove once all clusters have been reprovisioned.
		logger.Warn("ObservedGeneration status value missing during reconciliation check; update mattermost operator on this cluster")
		if unwrapInt32(cr.Spec.Replicas) != cr.Status.Replicas ||
			cr.Spec.Version != cr.Status.Version {
			return false, nil
		}
	}

	return true, nil
}

// generateAffinityConfig generates pods Affinity configuration aiming to spread pods of single cluster installation
// across different availability zones and nodes.
func generateAffinityConfig(installation *model.Installation, clusterInstallation *model.ClusterInstallation) *v1.Affinity {
	return &v1.Affinity{
		PodAntiAffinity: &v1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
				{
					Weight: 100,
					PodAffinityTerm: v1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: clusterInstallationBaseLabels(installation, clusterInstallation),
						},
						Namespaces:  []string{clusterInstallation.Namespace},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
				{
					Weight: 100,
					PodAffinityTerm: v1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: clusterInstallationBaseLabels(installation, clusterInstallation),
						},
						Namespaces:  []string{clusterInstallation.Namespace},
						TopologyKey: "topology.kubernetes.io/zone",
					},
				},
			},
		},
	}
}

// getMattermostCustomResource gets the cluster installation resource from
// the kubernetes API.
func (provisioner *kopsCIBeta) getMattermostCustomResource(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, logger log.FieldLogger) (*mmv1beta1.Mattermost, error) {
	configLocation, err := provisioner.getCachedKopsClusterKubecfg(cluster.ProvisionerMetadataKops.Name, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kops config from cache")
	}
	defer provisioner.invalidateCachedKopsClientOnError(err, cluster.ProvisionerMetadataKops.Name, logger)

	k8sClient, err := k8s.NewFromFile(configLocation, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create k8s client from file")
	}

	name := makeClusterInstallationName(clusterInstallation)

	ctx := context.TODO()
	cr, err := k8sClient.MattermostClientsetV1Beta.MattermostV1beta1().Mattermosts(clusterInstallation.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return cr, errors.Wrapf(err, "failed to get cluster installation %s", clusterInstallation.ID)
	}

	logger.WithField("status", fmt.Sprintf("%+v", cr.Status)).Debug("Got cluster installation")

	return cr, nil
}

func (provisioner *kopsCIBeta) migrateFromClusterInstallation(clusterInstallation *model.ClusterInstallation, k8sClient *k8s.KubeClient, logger log.FieldLogger) (bool, error) {
	name := makeClusterInstallationName(clusterInstallation)
	ciClient := k8sClient.MattermostClientsetV1Alpha.MattermostV1alpha1().ClusterInstallations(clusterInstallation.Namespace)
	mmClient := k8sClient.MattermostClientsetV1Beta.MattermostV1beta1().Mattermosts(clusterInstallation.Namespace)

	// If old resource does not exist we assume the migration is done,
	// otherwise we cannot migrate anyway.
	ctx := context.TODO()
	cr, err := ciClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return true, nil
		}

		return false, errors.Wrapf(err, "failed to get cluster installation %s", clusterInstallation.ID)
	}

	if cr.Spec.Migrate {
		if cr.Status.Migration == nil {
			logger.Warn("Installation migration waiting to start")
			return false, nil
		}
		if cr.Status.Migration.Error != "" {
			return false, errors.Errorf("error while migrating ClusterInstallation: %s", cr.Status.Migration.Error)
		}

		logger.Infof("Migration already in progress, status: %s", cr.Status.Migration.Status)
		return provisioner.isCRMigrated(name, ciClient, mmClient, logger)
	}

	logger.Info("Starting CR migration to Mattermost")

	cr.Spec.Migrate = true
	cr, err = ciClient.Update(ctx, cr, metav1.UpdateOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to start migration of the ClusterInstallation")
	}

	return provisioner.isCRMigrated(name, ciClient, mmClient, logger)
}

func (provisioner *kopsCIBeta) isCRMigrated(name string, ciClient v1alpha1.ClusterInstallationInterface, mmClient v1beta1.MattermostInterface, logger log.FieldLogger) (bool, error) {
	ctx := context.Background()
	_, err := ciClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return false, errors.Wrapf(err, "failed to get cluster installation %s", name)
	}
	if err == nil {
		logger.Info("Cluster Installation not migrated, old still CR exists")
		return false, nil
	}

	mm, err := mmClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			logger.Info("Cluster Installation not migrated, new CR does not exist")
			return false, nil
		}
		return false, errors.Wrapf(err, "failed to get Mattermost CR: %s", name)
	}

	if mm.Status.State != mmv1beta1.Stable {
		logger.Infof("Cluster Installation not migrated, new CR not stable, state: %s", mm.Status.State)
		return false, nil
	}

	return true, nil
}

func int32Ptr(i int) *int32 {
	i32 := int32(i)
	return &i32
}

func unwrapInt32(i *int32) int32 {
	if i != nil {
		return *i
	}
	return 0
}
