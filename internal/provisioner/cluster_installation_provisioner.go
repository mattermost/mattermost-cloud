// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/utils"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	mmv1beta1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1beta1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CommonProvisioner groups functions common to different provisioners.
type CommonProvisioner struct {
	resourceUtil *utils.ResourceUtil
	store        model.InstallationDatabaseStoreInterface
	params       ProvisioningParams
	logger       log.FieldLogger
}

func (provisioner *CommonProvisioner) createClusterInstallation(clusterInstallation *model.ClusterInstallation, installation *model.Installation, installationDNS []*model.InstallationDNS, kubeconfigPath string, logger log.FieldLogger) error {
	k8sClient, err := k8s.NewFromFile(kubeconfigPath, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client from file")
	}

	installationName, err := prepareClusterInstallationEnv(clusterInstallation, k8sClient)
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
			Version:       translateMattermostVersion(installation.Version),
			Image:         installation.Image,
			MattermostEnv: mattermostEnv.ToEnvList(),
			Ingress:       makeIngressSpec(installationDNS),
			// Set `installation-id` and `cluster-installation-id` labels for all related resources.
			ResourceLabels: clusterInstallationStableLabels(installation, clusterInstallation),
			Scheduling: mmv1beta1.Scheduling{
				Affinity: generateAffinityConfig(installation, clusterInstallation),
			},
			DNSConfig: setNdots(provisioner.params.NdotsValue),
		},
	}

	err = setMMInstanceSize(installation, mattermost)
	if err != nil {
		return errors.Wrap(err, "failed to set Mattermost instance size")
	}

	if installation.State == model.InstallationStateHibernating {
		logger.Info("creating hibernated cluster installation")
		configureInstallationForHibernation(mattermost, installation, clusterInstallation)
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

	err = createInstallationSLI(clusterInstallation, k8sClient, logger)
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

func (provisioner *CommonProvisioner) ensureFilestoreAndDatabase(
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

func hibernateInstallation(configLocation string, logger *log.Entry, clusterInstallation *model.ClusterInstallation, installation *model.Installation) error {
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

	configureInstallationForHibernation(cr, installation, clusterInstallation)

	_, err = k8sClient.MattermostClientsetV1Beta.MattermostV1beta1().Mattermosts(clusterInstallation.Namespace).Update(ctx, cr, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to update cluster installation %s", clusterInstallation.ID)
	}
	err = deleteInstallationSLI(clusterInstallation, k8sClient, logger)
	if err != nil {
		return errors.Wrap(err, "failed to delete installation SLI")
	}
	logger.Info("Updated cluster installation")

	return nil
}

// refreshSecrets deletes old secrets for database and file store and replaces them with new ones.
func (provisioner *CommonProvisioner) refreshSecrets(installation *model.Installation, clusterInstallation *model.ClusterInstallation, configLocation string) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})
	logger.Info("Refreshing secrets for cluster installation")

	k8sClient, err := k8s.NewFromFile(configLocation, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client from file")
	}

	installationName := makeClusterInstallationName(clusterInstallation)

	ctx := context.TODO()
	mmClient := k8sClient.MattermostClientsetV1Beta.MattermostV1beta1().Mattermosts(clusterInstallation.Namespace)

	mattermost, err := mmClient.Get(ctx, installationName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to get mattermost installation %s", clusterInstallation.ID)
	}

	err = deleteMMSecrets(clusterInstallation.Namespace, mattermost, k8sClient, logger)
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

func (provisioner *CommonProvisioner) updateClusterInstallation(
	configLocation string,
	installation *model.Installation,
	installationDNS []*model.InstallationDNS,
	clusterInstallation *model.ClusterInstallation,
	logger log.FieldLogger) error {
	k8sClient, err := k8s.NewFromFile(configLocation, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client from file")
	}

	installationName, err := prepareClusterInstallationEnv(clusterInstallation, k8sClient)
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
	mattermost.Spec.ResourceLabels = clusterInstallationStableLabels(installation, clusterInstallation)

	mattermost.Spec.Scheduling.Affinity = generateAffinityConfig(installation, clusterInstallation)

	mattermost.Spec.DNSConfig = setNdots(provisioner.params.NdotsValue)

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
	err = setMMInstanceSize(installation, mattermost)
	if err != nil {
		return errors.Wrap(err, "failed to set Mattermost instance size")
	}

	mattermost.Spec.LicenseSecret = ""
	var secretName string
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
	mattermost.Spec.Ingress = makeIngressSpec(installationDNS)

	_, err = k8sClient.MattermostClientsetV1Beta.MattermostV1beta1().Mattermosts(clusterInstallation.Namespace).Update(ctx, mattermost, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to update cluster installation %s", clusterInstallation.ID)
	}

	err = createOrUpdateInstallationSLI(clusterInstallation, k8sClient, logger)
	if err != nil {
		return errors.Wrapf(err, "failed to create cluster installation SLI %s", clusterInstallation.ID)
	}

	logger.Info("Updated cluster installation")

	return nil
}

func deleteMMSecrets(ns string, mattermost *mmv1beta1.Mattermost, kubeClient *k8s.KubeClient, logger log.FieldLogger) error {
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

func deleteClusterInstallation(
	installation *model.Installation,
	clusterInstallation *model.ClusterInstallation,
	configLocation string,
	logger log.FieldLogger) error {

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

	err = deleteInstallationSLI(clusterInstallation, k8sClient, logger)
	if err != nil {
		return errors.Wrap(err, "failed to delete installation SLI")
	}

	logger.Info("Successfully deleted cluster installation")

	return nil
}

// deleteOldClusterInstallationLicenseSecrets removes k8s secrets found matching
// the license naming scheme that are not the current license used by the
// installation.
func deleteOldClusterInstallationLicenseSecrets(kubeconfigPath string, installation *model.Installation, clusterInstallation *model.ClusterInstallation, logger log.FieldLogger) error {
	k8sClient, err := k8s.NewFromFile(kubeconfigPath, logger)
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

func prepareClusterUtilities(
	cluster *model.Cluster,
	configLocation string,
	store model.ClusterUtilityDatabaseStoreInterface,
	awsClient aws.AWS,
	pgbouncerConfig *PGBouncerConfig,
	logger log.FieldLogger) error {
	k8sClient, err := k8s.NewFromFile(configLocation, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client from file")
	}

	var vpc string
	if cluster.ProvisionerMetadataKops != nil {
		vpc = cluster.ProvisionerMetadataKops.VPC
	} else if cluster.ProvisionerMetadataEKS != nil {
		vpc = cluster.ProvisionerMetadataEKS.VPC
	} else {
		return errors.New("cluster metadata is nil cannot determine VPC")
	}

	// TODO: Yeah, so this is definitely a bit of a race condition. We would
	// need to lock a bunch of stuff to do this completely properly, but that
	// isn't really feasible right now.
	ini, err := generatePGBouncerIni(vpc, store, pgbouncerConfig)
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

		userlist, err := generatePGBouncerUserlist(vpc, awsClient)
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

func prepareClusterInstallationEnv(clusterInstallation *model.ClusterInstallation, k8sClient *k8s.KubeClient) (string, error) {
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

func isMattermostReady(cr *mmv1beta1.Mattermost) (bool, bool, error) {
	if cr.Generation != cr.Status.ObservedGeneration {
		return false, false, nil
	}
	if cr.Status.State == mmv1beta1.Stable {
		return true, true, nil
	}
	if cr.Status.State == mmv1beta1.Ready {
		return true, false, nil
	}

	return false, false, nil
}

// getMattermostCustomResource gets the cluster installation resource from
// the kubernetes API.
func getMattermostCustomResource(clusterInstallation *model.ClusterInstallation, kubeconfigFile string, logger log.FieldLogger) (*mmv1beta1.Mattermost, error) {
	k8sClient, err := k8s.NewFromFile(kubeconfigFile, logger)
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
