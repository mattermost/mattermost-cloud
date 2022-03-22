// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	mmv1beta1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1beta1"

	"github.com/mattermost/mattermost-operator/pkg/resources"
	"github.com/pborman/uuid"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	bifrostEndpoint           = "bifrost.bifrost:80"
	ciExecJobTTLSeconds int32 = 180
)

// ClusterInstallationProvisioner is an interface for provisioning and managing ClusterInstallations.
type ClusterInstallationProvisioner interface {
	CreateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error
	EnsureCRMigrated(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation) (bool, error)
	HibernateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error
	UpdateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error
	VerifyClusterInstallationMatchesConfig(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) (bool, error)
	DeleteOldClusterInstallationLicenseSecrets(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error
	DeleteClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error
	IsResourceReady(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation) (bool, error)
	RefreshSecrets(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error
	PrepareClusterUtilities(cluster *model.Cluster, installation *model.Installation, store model.ClusterUtilityDatabaseStoreInterface, awsClient aws.AWS) error
}

// ClusterInstallationProvisioner function returns an implementation of ClusterInstallationProvisioner interface
// based on specified Custom Resource version.
func (provisioner *KopsProvisioner) ClusterInstallationProvisioner(crVersion string) ClusterInstallationProvisioner {
	if crVersion != model.V1betaCRVersion {
		provisioner.logger.Errorf("Unexpected resource version: %s", crVersion)
	}

	return &crProvisionerWrapper{KopsProvisioner: provisioner}
}

type crProvisionerWrapper struct {
	*KopsProvisioner
}

// CreateClusterInstallation creates a Mattermost installation within the given cluster.
func (provisioner *crProvisionerWrapper) CreateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
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
			Size:          installation.Size,
			Version:       translateMattermostVersion(installation.Version),
			Image:         installation.Image,
			MattermostEnv: mattermostEnv.ToEnvList(),
			Ingress:       makeIngressSpec(installation.DNS),
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

	err = provisioner.createInstallationSLI(clusterInstallation, k8sClient, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create installation SLI")
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
func (provisioner *crProvisionerWrapper) HibernateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
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
	err = provisioner.deleteInstallationSLI(clusterInstallation, k8sClient, logger)
	if err != nil {
		return errors.Wrap(err, "failed to delete installation SLI")
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
func (provisioner *crProvisionerWrapper) UpdateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
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

	// Just to be sure, for the update we reset deprecated fields.
	mattermost.Spec.IngressName = ""
	mattermost.Spec.IngressAnnotations = nil
	mattermost.Spec.Ingress = makeIngressSpec(installation.DNS)

	_, err = k8sClient.MattermostClientsetV1Beta.MattermostV1beta1().Mattermosts(clusterInstallation.Namespace).Update(ctx, mattermost, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to update cluster installation %s", clusterInstallation.ID)
	}

	err = provisioner.createOrUpdateInstallationSLI(clusterInstallation, k8sClient, logger)
	if err != nil {
		return errors.Wrapf(err, "failed to create cluster installation SLI %s", clusterInstallation.ID)
	}

	logger.Info("Updated cluster installation")

	return nil
}

// RefreshSecrets deletes old secrets for database and file store and replaces them with new ones.
func (provisioner *crProvisionerWrapper) RefreshSecrets(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
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

func (provisioner *crProvisionerWrapper) deleteMMSecrets(ns string, mattermost *mmv1beta1.Mattermost, kubeClient *k8s.KubeClient, logger log.FieldLogger) error {
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

func (provisioner *crProvisionerWrapper) ensureFilestoreAndDatabase(
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

func (provisioner *crProvisionerWrapper) EnsureCRMigrated(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation) (bool, error) {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})
	logger.Debug("All cluster installation are expected to be v1beta version")

	return true, nil
}

// VerifyClusterInstallationMatchesConfig attempts to verify that a cluster
// installation custom resource matches the configuration that is defined in the
// provisioner
// NOTE: this does NOT ensure that other resources such as network policies for
// that namespace are correct. Also, the values checked are ONLY values that are
// defined by both the installation and group configuration.
func (provisioner *crProvisionerWrapper) VerifyClusterInstallationMatchesConfig(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) (bool, error) {
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
func (provisioner *crProvisionerWrapper) DeleteClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
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

	err = provisioner.deleteInstallationSLI(clusterInstallation, k8sClient, logger)
	if err != nil {
		return errors.Wrap(err, "failed to delete installation SLI")
	}

	logger.Info("Successfully deleted cluster installation")

	return nil
}

// IsResourceReady checks if the ClusterInstallation Custom Resource is ready on the cluster.
func (provisioner *crProvisionerWrapper) IsResourceReady(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation) (bool, error) {
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

func makeIngressSpec(installationDNS string) *mmv1beta1.Ingress {
	ingressClass := "nginx-controller"
	return &mmv1beta1.Ingress{
		Enabled:      true,
		Host:         installationDNS,
		Annotations:  getIngressAnnotations(),
		IngressClass: &ingressClass,
	}
}

// generateAffinityConfig generates pods Affinity configuration aiming to spread pods of single cluster installation
// across different availability zones and nodes.
func generateAffinityConfig(installation *model.Installation, clusterInstallation *model.ClusterInstallation) *corev1.Affinity {
	return &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: 100,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: clusterInstallationBaseLabels(installation, clusterInstallation),
						},
						Namespaces:  []string{clusterInstallation.Namespace},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
				{
					Weight: 100,
					PodAffinityTerm: corev1.PodAffinityTerm{
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
func (provisioner *crProvisionerWrapper) getMattermostCustomResource(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, logger log.FieldLogger) (*mmv1beta1.Mattermost, error) {
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

// ExecMattermostCLI invokes the Mattermost CLI for the given cluster installation with the given args.
func (provisioner *KopsProvisioner) ExecMattermostCLI(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) ([]byte, error) {
	return provisioner.ExecClusterInstallationCLI(cluster, clusterInstallation, append([]string{"./bin/mattermost"}, args...)...)
}

// ExecClusterInstallationCLI execs the provided command on the defined cluster installation.
func (provisioner *KopsProvisioner) ExecClusterInstallationCLI(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) ([]byte, error) {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	configLocation, err := provisioner.getCachedKopsClusterKubecfg(cluster.ProvisionerMetadataKops.Name, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kops config from cache")
	}
	defer provisioner.invalidateCachedKopsClientOnError(err, cluster.ProvisionerMetadataKops.Name, logger)

	k8sClient, err := k8s.NewFromFile(configLocation, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create k8s client from file")
	}

	ctx := context.TODO()
	podList, err := k8sClient.Clientset.CoreV1().Pods(clusterInstallation.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=mattermost",
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to query mattermost pods")
	}

	// In the future, we'd ideally just spin our own container on demand, allowing
	// configuration changes even if the pods are failing to start the server. For now,
	// we find the first pod running Mattermost, and pick the first container therein.

	if len(podList.Items) == 0 {
		return nil, errors.New("failed to find mattermost pods on which to exec")
	}

	pod := podList.Items[0]
	if len(pod.Spec.Containers) == 0 {
		return nil, errors.Errorf("failed to find containers in pod %s", pod.Name)
	}

	container := pod.Spec.Containers[0]
	logger.Debugf("Executing `%s` on pod %s, container %s, running image %s", strings.Join(args, " "), pod.Name, container.Name, container.Image)

	execRequest := k8sClient.Clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(clusterInstallation.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container.Name,
			Command:   args,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	now := time.Now()
	output, err := k8sClient.RemoteCommand("POST", execRequest.URL())

	logger.Debugf("Command `%s` on pod %s finished in %.0f seconds", strings.Join(args, " "), pod.Name, time.Since(now).Seconds())

	return output, err
}

// ExecClusterInstallationJob creates job executing command on cluster installation.
func (provisioner *KopsProvisioner) ExecClusterInstallationJob(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})
	logger.Info("Executing job with CLI command on cluster installation")

	k8sClient, invalidateCache, err := provisioner.k8sClient(cluster.ProvisionerMetadataKops.Name, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client")
	}
	defer invalidateCache(err)

	ctx := context.TODO()
	deploymentList, err := k8sClient.Clientset.AppsV1().Deployments(clusterInstallation.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=mattermost",
	})
	if err != nil {
		return errors.Wrap(err, "failed to get installation deployments")
	}

	if len(deploymentList.Items) == 0 {
		return errors.New("no mattermost deployments found")
	}

	jobName := fmt.Sprintf("command-%s", uuid.New()[:6])
	job := resources.PrepareMattermostJobTemplate(jobName, clusterInstallation.Namespace, &deploymentList.Items[0])
	// TODO: refactor above method in Mattermost Operator to take command and handle this logic inside.
	for i := range job.Spec.Template.Spec.Containers {
		job.Spec.Template.Spec.Containers[i].Command = args
		// We want to match bifrost network policy so that server can come up quicker.
		job.Spec.Template.Labels["app"] = "mattermost"
	}
	jobTTL := ciExecJobTTLSeconds
	job.Spec.TTLSecondsAfterFinished = &jobTTL

	jobsClient := k8sClient.Clientset.BatchV1().Jobs(clusterInstallation.Namespace)

	defer func() {
		err := jobsClient.Delete(ctx, jobName, metav1.DeleteOptions{})
		if err != nil && !k8sErrors.IsNotFound(err) {
			logger.Errorf("Failed to cleanup exec job: %q", jobName)
		}
	}()

	job, err = jobsClient.Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create CLI command job")
	}

	err = wait.Poll(time.Second, 2*time.Minute, func() (bool, error) {
		job, err = jobsClient.Get(ctx, jobName, metav1.GetOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "failed to get %q job", jobName)
		}
		if job.Status.Succeeded < 1 {
			logger.Infof("job %q not yet finished, waiting up to 1 minute", jobName)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return errors.Wrapf(err, "job %q did not finish in expected time", jobName)
	}

	return nil
}

// DeleteOldClusterInstallationLicenseSecrets removes k8s secrets found matching
// the license naming scheme that are not the current license used by the
// installation.
func (provisioner *KopsProvisioner) DeleteOldClusterInstallationLicenseSecrets(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
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

	var currentSecretName string
	if installation.License != "" {
		currentSecretName = generateCILicenseName(installation, clusterInstallation)
	}

	err = cleanupOldLicenseSecrets(currentSecretName, clusterInstallation, k8sClient, logger)
	if err != nil {
		return errors.Wrap(err, "failed to cleanup old license secrets")
	}

	return nil
}

// PrepareClusterUtilities performs any updates to cluster utilities that may
// be needed for clusterinstallations to function correctly.
func (provisioner *KopsProvisioner) PrepareClusterUtilities(cluster *model.Cluster, installation *model.Installation, store model.ClusterUtilityDatabaseStoreInterface, awsClient aws.AWS) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)
	logger.Info("Preparing cluster utilities")

	// TODO: move this logic to a database interface method.
	if installation.Database != model.InstallationDatabaseMultiTenantRDSPostgresPGBouncer {
		return nil
	}

	configLocation, err := provisioner.getCachedKopsClusterKubecfg(cluster.ProvisionerMetadataKops.Name, logger)
	if err != nil {
		return errors.Wrap(err, "failed to get kops config from cache")
	}
	defer provisioner.invalidateCachedKopsClientOnError(err, cluster.ProvisionerMetadataKops.Name, logger)

	k8sClient, err := k8s.NewFromFile(configLocation, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client from file")
	}

	// TODO: Yeah, so this is definitely a bit of a race condition. We would
	// need to lock a bunch of stuff to do this completely properly, but that
	// isn't really feasible right now.
	ini, err := generatePGBouncerIni(cluster.ProvisionerMetadataKops.VPC, store)
	if err != nil {
		return errors.Wrap(err, "failed to generate updated pgbouncer ini contents")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
	defer cancel()

	configMap, err := k8sClient.Clientset.CoreV1().ConfigMaps("pgbouncer").Get(ctx, "pgbouncer-configmap", metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get configmap for pgbouncer-configmap")
	}
	if configMap.Data["pgbouncer.ini"] != ini {
		logger.Debug("Updating pgbouncer.ini with new database configuration")

		configMap.Data["pgbouncer.ini"] = ini
		_, err = k8sClient.Clientset.CoreV1().ConfigMaps("pgbouncer").Update(ctx, configMap, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to update configmap pgbouncer-configmap")
		}
	}

	userlistSecret, err := k8sClient.Clientset.CoreV1().Secrets("pgbouncer").Get(ctx, "pgbouncer-userlist-secret", metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get configmap for pgbouncer-configmap")
	}

	if !strings.Contains(string(userlistSecret.Data["userlist.txt"]), aws.DefaultPGBouncerAuthUsername) {
		logger.Debug("Updating pgbouncer userlist.txt with auth_user credentials")

		userlist, err := generatePGBouncerUserlist(cluster.ProvisionerMetadataKops.VPC, awsClient)
		if err != nil {
			return errors.Wrap(err, "failed to generate pgbouncer userlist")
		}

		userlistSecret.Data["userlist.txt"] = []byte(userlist)
		_, err = k8sClient.Clientset.CoreV1().Secrets("pgbouncer").Update(ctx, userlistSecret, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to update secret pgbouncer-userlist-secret")
		}
	}

	return nil
}

func (provisioner *KopsProvisioner) prepareClusterInstallationEnv(clusterInstallation *model.ClusterInstallation, k8sClient *k8s.KubeClient) (string, error) {
	_, err := k8sClient.CreateOrUpdateNamespace(clusterInstallation.Namespace)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create namespace %s", clusterInstallation.Namespace)
	}

	installationName := makeClusterInstallationName(clusterInstallation)

	file := k8s.ManifestFile{
		Path:            "manifests/network-policies/mm-installation-netpol.yaml",
		DeployNamespace: clusterInstallation.Namespace,
	}
	err = k8sClient.CreateFromFile(file, installationName)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create network policy %s", clusterInstallation.Namespace)
	}

	return installationName, nil
}

func prepareCILicenseSecret(installation *model.Installation, clusterInstallation *model.ClusterInstallation, k8sClient *k8s.KubeClient) (string, error) {
	licenseSecretName := generateCILicenseName(installation, clusterInstallation)
	licenseSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      licenseSecretName,
			Namespace: clusterInstallation.Namespace,
		},
		StringData: map[string]string{"license": installation.License},
	}

	_, err := k8sClient.CreateOrUpdateSecret(clusterInstallation.Namespace, licenseSecret)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create the license secret %s/%s", clusterInstallation.Namespace, licenseSecretName)
	}

	return licenseSecretName, nil
}

// generateCILicenseName generates a unique license secret name by using a short
// sha256 hash.
func generateCILicenseName(installation *model.Installation, clusterInstallation *model.ClusterInstallation) string {
	return fmt.Sprintf("%s-%s-license",
		makeClusterInstallationName(clusterInstallation),
		fmt.Sprintf("%x", sha256.Sum256([]byte(installation.License)))[0:6],
	)
}

// cleanupOldLicenseSecrets removes an secrets matching the license naming
// convention except the current license secret name. Pass in a blank name value
// to cleanup all license secrets.
func cleanupOldLicenseSecrets(currentSecretName string, clusterInstallation *model.ClusterInstallation, k8sClient *k8s.KubeClient, logger log.FieldLogger) error {
	secrets, err := k8sClient.Clientset.CoreV1().Secrets(clusterInstallation.Namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to list secrets")
	}
	for _, secret := range secrets.Items {
		if !strings.HasPrefix(secret.Name, makeClusterInstallationName(clusterInstallation)) || !strings.HasSuffix(secret.Name, "-license") {
			continue
		}
		if secret.Name == currentSecretName {
			continue
		}

		logger.Infof("Deleting old license secret %s", secret.Name)

		err := k8sClient.Clientset.CoreV1().Secrets(clusterInstallation.Namespace).Delete(context.Background(), secret.Name, metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to delete secret %s/%s", clusterInstallation.Namespace, secret.Name)
		}
	}

	return nil
}

// generateClusterInstallationResourceLabels generates standard resource labels
// for ClusterInstallation resources.
func generateClusterInstallationResourceLabels(installation *model.Installation, clusterInstallation *model.ClusterInstallation) map[string]string {
	labels := clusterInstallationBaseLabels(installation, clusterInstallation)
	if installation.GroupID != nil {
		labels["group-id"] = *installation.GroupID
	}
	if installation.GroupSequence != nil {
		labels["group-sequence"] = fmt.Sprintf("%d", *installation.GroupSequence)
	}

	return labels
}

func clusterInstallationBaseLabels(installation *model.Installation, clusterInstallation *model.ClusterInstallation) map[string]string {
	return map[string]string{
		"installation-id":         installation.ID,
		"cluster-installation-id": clusterInstallation.ID,
	}
}

// Set env overrides that are required from installations for function correctly
// in the cloud environment.
// NOTE: this should be called whenever the Mattermost custom resource is created
// or updated.
func getMattermostEnvWithOverrides(installation *model.Installation) model.EnvVarMap {
	mattermostEnv := installation.GetEnvVars()
	if mattermostEnv == nil {
		mattermostEnv = map[string]model.EnvVar{}
	}

	// General overrides.
	mattermostEnv["MM_CLOUD_INSTALLATION_ID"] = model.EnvVar{Value: installation.ID}
	groupID := installation.GroupID
	if groupID != nil {
		mattermostEnv["MM_CLOUD_GROUP_ID"] = model.EnvVar{Value: *groupID}
	}
	mattermostEnv["MM_SERVICESETTINGS_ENABLELOCALMODE"] = model.EnvVar{Value: "true"}

	// Filestore overrides.
	if !installation.InternalFilestore() {
		mattermostEnv["MM_FILESETTINGS_AMAZONS3SSE"] = model.EnvVar{Value: "true"}
	}
	if installation.Filestore == model.InstallationFilestoreMultiTenantAwsS3 ||
		installation.Filestore == model.InstallationFilestoreBifrost {
		mattermostEnv["MM_FILESETTINGS_AMAZONS3PATHPREFIX"] = model.EnvVar{Value: installation.ID}
	}
	if installation.Filestore == model.InstallationFilestoreBifrost {
		mattermostEnv["MM_CLOUD_FILESTORE_BIFROST"] = model.EnvVar{Value: "true"}
		mattermostEnv["MM_FILESETTINGS_AMAZONS3ENDPOINT"] = model.EnvVar{Value: bifrostEndpoint}
		mattermostEnv["MM_FILESETTINGS_AMAZONS3SIGNV2"] = model.EnvVar{Value: "false"}
		mattermostEnv["MM_FILESETTINGS_AMAZONS3SSE"] = model.EnvVar{Value: "false"}
		mattermostEnv["MM_FILESETTINGS_AMAZONS3SSL"] = model.EnvVar{Value: "false"}
	}

	return mattermostEnv
}

// getIngressAnnotations returns ingress annotations used by Mattermost installations.
func getIngressAnnotations() map[string]string {
	return map[string]string{
		"kubernetes.io/tls-acme":                               "true",
		"nginx.ingress.kubernetes.io/proxy-buffering":          "on",
		"nginx.ingress.kubernetes.io/proxy-body-size":          "100m",
		"nginx.ingress.kubernetes.io/proxy-send-timeout":       "600",
		"nginx.ingress.kubernetes.io/proxy-read-timeout":       "600",
		"nginx.ingress.kubernetes.io/proxy-max-temp-file-size": "0",
		"nginx.ingress.kubernetes.io/ssl-redirect":             "true",
		"nginx.ingress.kubernetes.io/configuration-snippet": `
				  proxy_force_ranges on;
				  add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
				  proxy_cache mattermost_cache;
				  proxy_cache_revalidate on;
				  proxy_cache_min_uses 2;
				  proxy_cache_use_stale timeout;
				  proxy_cache_lock on;
				  proxy_cache_key "$host$request_uri$cookie_user";`,
		"nginx.org/server-snippets": "gzip on;",
	}
}

// getHibernatingIngressAnnotations returns ingress annotations used by
// hibernating Mattermost installations.
func getHibernatingIngressAnnotations() map[string]string {
	annotations := getIngressAnnotations()
	annotations["nginx.ingress.kubernetes.io/configuration-snippet"] = "return 410;"

	return annotations
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

func ensureEnvMatch(wanted corev1.EnvVar, all []corev1.EnvVar) bool {
	for _, env := range all {
		if env == wanted {
			return true
		}
	}

	return false
}
