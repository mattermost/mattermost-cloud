// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/mattermost/mattermost-cloud/internal/provisioner/pgbouncer"
	"github.com/mattermost/mattermost-cloud/internal/provisioner/prometheus"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-operator/apis/mattermost/v1alpha1"
	mmv1beta1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1beta1"
	"github.com/mattermost/mattermost-operator/pkg/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	bifrostEndpoint = "bifrost.bifrost:80"
)

// CreateClusterInstallation creates a Mattermost installation within the given cluster.
func (provisioner Provisioner) CreateClusterInstallation(cluster *model.Cluster, installation *model.Installation, installationDNS []*model.InstallationDNS, clusterInstallation *model.ClusterInstallation) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
		"version":      "v1beta1",
	})
	logger.Info("Creating cluster installation")

	configLocation, err := provisioner.getClusterKubecfg(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to get kube config path")
	}

	return provisioner.createClusterInstallation(clusterInstallation, installation, installationDNS, configLocation, logger)
}

func (provisioner Provisioner) EnsureCRMigrated(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation) (bool, error) {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})
	logger.Debug("All cluster installation are expected to be v1beta version")

	return true, nil
}

// HibernateClusterInstallation updates a cluster installation to consume fewer
// resources.
func (provisioner Provisioner) HibernateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	configLocation, err := provisioner.getClusterKubecfg(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to get kube config path")
	}

	return hibernateInstallation(configLocation, logger, clusterInstallation, installation)
}

// UpdateClusterInstallation updates the cluster installation spec to match the
// installation specification.
func (provisioner Provisioner) UpdateClusterInstallation(cluster *model.Cluster, installation *model.Installation, installationDNS []*model.InstallationDNS, clusterInstallation *model.ClusterInstallation) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	configLocation, err := provisioner.getClusterKubecfg(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to get kube config path")
	}

	return provisioner.updateClusterInstallation(configLocation, installation, installationDNS, clusterInstallation, logger)
}

// DeleteOldClusterInstallationLicenseSecrets removes k8s secrets found matching
// the license naming scheme that are not the current license used by the
// installation.
func (provisioner Provisioner) DeleteOldClusterInstallationLicenseSecrets(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	configLocation, err := provisioner.getClusterKubecfg(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to get kube config path")
	}

	return deleteOldClusterInstallationLicenseSecrets(configLocation, installation, clusterInstallation, logger)
}

// DeleteClusterInstallation deletes a Mattermost installation within the given cluster.
func (provisioner Provisioner) DeleteClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	configLocation, err := provisioner.getClusterKubecfg(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to get kube config path")
	}

	return deleteClusterInstallation(installation, clusterInstallation, configLocation, logger)
}

// IsResourceReadyAndStable checks if the ClusterInstallation Custom Resource is
// both ready and stable on the cluster.
func (provisioner Provisioner) IsResourceReadyAndStable(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation) (bool, bool, error) {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":              clusterInstallation.ClusterID,
		"installation":         clusterInstallation.InstallationID,
		"cluster_installation": clusterInstallation.ID,
	})

	cr, err := provisioner.getMattermostCustomResource(cluster, clusterInstallation, logger)
	if err != nil {
		return false, false, errors.Wrap(err, "failed to get ClusterInstallation Custom Resource")
	}
	return isMattermostReady(cr)
}

// RefreshSecrets deletes old secrets for database and file store and replaces them with new ones.
func (provisioner Provisioner) RefreshSecrets(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})
	logger.Info("Refreshing secrets for cluster installation")

	configLocation, err := provisioner.getClusterKubecfg(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to get kube config path")
	}

	return provisioner.refreshSecrets(installation, clusterInstallation, configLocation)
}

// PrepareClusterUtilities performs any updates to cluster utilities that may
// be needed for clusterinstallations to function correctly.
func (provisioner Provisioner) PrepareClusterUtilities(cluster *model.Cluster, installation *model.Installation, store model.ClusterUtilityDatabaseStoreInterface) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)
	logger.Info("Preparing cluster utilities")

	// TODO: move this logic to a database interface method.
	if installation.Database != model.InstallationDatabaseMultiTenantRDSPostgresPGBouncer {
		return nil
	}

	configLocation, err := provisioner.getClusterKubecfg(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to get kube config path")
	}

	return prepareClusterUtilities(cluster, configLocation, store, provisioner.awsClient, provisioner.params.PGBouncerConfig, logger)
}

func (provisioner Provisioner) createClusterInstallation(clusterInstallation *model.ClusterInstallation, installation *model.Installation, installationDNS []*model.InstallationDNS, kubeconfigPath string, logger log.FieldLogger) error {
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
			DeploymentTemplate: &mmv1beta1.DeploymentTemplate{
				RevisionHistoryLimit: ptr.Int32(1),
			},
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
		var licenseSecretName string
		licenseSecretName, err = prepareCILicenseSecret(installation, clusterInstallation, k8sClient)
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

	if installation.GroupID != nil && *installation.GroupID != "" {
		if containsInstallationGroup(*installation.GroupID, provisioner.params.SLOInstallationGroups) {
			logger.Debug("Installation belongs in the approved SLO installation group list. Adding SLI")
			installationName := makeClusterInstallationName(clusterInstallation)
			err = prometheus.CreateInstallationSLI(clusterInstallation, k8sClient, installationName, logger)
			if err != nil {
				return errors.Wrap(err, "failed to create installation SLI")
			}
		}
		if containsInstallationGroup(*installation.GroupID, provisioner.params.SLOEnterpriseGroups) {
			logger.Debug("Installation belongs in the approved enterprise installation group list. Adding Nginx SLI")
			serviceName := makeClusterInstallationName(clusterInstallation)
			err = prometheus.CreateOrUpdateNginxSLI(clusterInstallation, k8sClient, serviceName, logger)
			if err != nil {
				return errors.Wrap(err, "failed to create enterprise nginx SLI")
			}
		}
	}

	ctx := context.TODO()
	_, err = k8sClient.MattermostClientsetV1Beta.MattermostV1beta1().Mattermosts(clusterInstallation.Namespace).Create(ctx, mattermost, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create cluster installation")
	}

	logger.Info("Successfully created cluster installation")

	return nil
}

func (provisioner Provisioner) ensureFilestoreAndDatabase(
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
	err = prometheus.DeleteInstallationSLI(clusterInstallation, k8sClient, logger)
	if err != nil {
		return errors.Wrap(err, "failed to delete installation SLI")
	}

	if err = prometheus.EnsureNginxSLIDeleted(clusterInstallation, k8sClient, logger); err != nil {
		return errors.Wrap(err, "failed to delete enterprise nginx SLI")
	}

	logger.Info("Updated cluster installation")

	return nil
}

// refreshSecrets deletes old secrets for database and file store and replaces them with new ones.
func (provisioner Provisioner) refreshSecrets(installation *model.Installation, clusterInstallation *model.ClusterInstallation, configLocation string) error {
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

func (provisioner Provisioner) updateClusterInstallation(
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

	if *installation.GroupID != "" && containsInstallationGroup(*installation.GroupID, provisioner.params.SLOInstallationGroups) {
		logger.Debug("Creating or updating Mattermost installation SLI")
		err = prometheus.CreateOrUpdateInstallationSLI(clusterInstallation, k8sClient, installationName, logger)
		if err != nil {
			return errors.Wrapf(err, "failed to create cluster installation SLI %s", clusterInstallation.ID)
		}
	} else {
		logger.Debug("Removing Mattermost installation SLI as installation group not in the approved list")
		err = prometheus.DeleteInstallationSLI(clusterInstallation, k8sClient, logger)
		if err != nil {
			return errors.Wrapf(err, "failed to delete cluster installation SLI %s", clusterInstallation.ID)
		}
	}

	if installation.GroupID != nil && *installation.GroupID != "" && containsInstallationGroup(*installation.GroupID, provisioner.params.SLOEnterpriseGroups) {
		logger.Debug("Creating or updating Mattermost Enterprise Nginx SLI")
		serviceName := makeClusterInstallationName(clusterInstallation)
		if err = prometheus.CreateOrUpdateNginxSLI(clusterInstallation, k8sClient, serviceName, logger); err != nil {
			return errors.Wrapf(err, "failed to create enterprise nginx SLI %s", prometheus.GetNginxSlothObjectName(clusterInstallation))
		}
	} else {
		logger.Debug("Removing Mattermost Enterprise Nginx SLI")
		if err := prometheus.EnsureNginxSLIDeleted(clusterInstallation, k8sClient, logger); err != nil {
			return errors.Wrapf(err, "failed to delete enterprise nginx SLI %s", prometheus.GetNginxSlothObjectName(clusterInstallation))
		}
	}

	logger.Info("Updated cluster installation")

	return nil
}

// getMattermostCustomResource gets the cluster installation resource from
// the kubernetes API.
func (provisioner Provisioner) getMattermostCustomResource(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, logger log.FieldLogger) (*mmv1beta1.Mattermost, error) {
	configLocation, err := provisioner.getClusterKubecfg(cluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kube config path")
	}

	return getMattermostCustomResource(clusterInstallation, configLocation, logger)
}

func (provisioner Provisioner) GetClusterInstallationStatus(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation) (*model.ClusterInstallationStatus, error) {
	k8sClient, err := provisioner.k8sClient(cluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kube client")
	}

	var status model.ClusterInstallationStatus

	deployment, err := k8sClient.Clientset.AppsV1().Deployments(clusterInstallation.Namespace).Get(context.TODO(), makeClusterInstallationName(clusterInstallation), metav1.GetOptions{})
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			return nil, errors.Wrap(err, "failed to query mattermost deployment")
		}

		return &status, nil
	}

	status.InstallationFound = true
	status.Replicas = deployment.Spec.Replicas

	if status.Replicas == nil || *status.Replicas == 0 {
		return &status, nil
	}

	podList, err := k8sClient.Clientset.CoreV1().Pods(clusterInstallation.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app=mattermost",
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to query mattermost pods")
	}

	status.TotalPod = ptr.Int32(int32(len(podList.Items)))

	args := []string{"./bin/mmctl", "--local", "system", "status", "--json"}

	var podRunningCount int32
	var podReadyCount int32
	var podStartedCount int32
	var mmctlSuccessCount = new(int32)
	var wg sync.WaitGroup

	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodRunning {
			podRunningCount++
		}

		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.Name == "mattermost" {
				if containerStatus.Ready {
					podReadyCount++
				}
				if containerStatus.Started != nil && *containerStatus.Started {
					podStartedCount++

					wg.Add(1)
					go func(podName string) {
						defer wg.Done()
						_, execErr := execCLI(k8sClient, clusterInstallation.Namespace, podName, "mattermost", args...)
						if execErr == nil {
							atomic.AddInt32(mmctlSuccessCount, 1)
						}
					}(pod.Name)

				}
				break
			}
		}
	}

	wg.Wait()

	status.RunningPod = &podRunningCount
	status.ReadyPod = &podReadyCount
	status.StartedPod = &podStartedCount
	status.ReadyLocalServer = mmctlSuccessCount

	return &status, nil
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

	err = prometheus.DeleteInstallationSLI(clusterInstallation, k8sClient, logger)
	if err != nil {
		return errors.Wrap(err, "failed to delete installation SLI")
	}

	if err = prometheus.EnsureNginxSLIDeleted(clusterInstallation, k8sClient, logger); err != nil {
		return errors.Wrap(err, "failed to delete enterprise nginx SLI")
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
	pgbouncerConfig *model.PGBouncerConfig,
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
	defer cancel()
	err = pgbouncer.UpdatePGBouncerConfigMap(ctx, vpc, store, pgbouncerConfig, k8sClient, logger)
	if err != nil {
		return errors.Wrap(err, "failed to update configmap for pgbouncer-configmap")
	}

	userlistSecret, err := k8sClient.Clientset.CoreV1().Secrets("pgbouncer").Get(ctx, "pgbouncer-userlist-secret", metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get configmap for pgbouncer-configmap")
	}

	if !strings.Contains(string(userlistSecret.Data["userlist.txt"]), aws.DefaultPGBouncerAuthUsername) {
		logger.Debug("Updating pgbouncer userlist.txt with auth_user credentials")

		userlist, err := pgbouncer.GeneratePGBouncerUserlist(vpc, awsClient)
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

func containsInstallationGroup(installationGroup string, installationGroups []string) bool {
	for _, ig := range installationGroups {
		if ig == installationGroup {
			return true
		}
	}
	return false
}

func configureInstallationForHibernation(mattermost *mmv1beta1.Mattermost, installation *model.Installation, clusterInstallation *model.ClusterInstallation) {
	// Hibernation is currently considered changing the Mattermost app deployment
	// to 0 replicas in the pod. i.e. Scale down to no Mattermost apps running.
	// The current way to do this is to set a negative replica count in the
	// k8s custom resource. Custom ingress annotations are also used.
	// TODO: enhance hibernation to include database and/or filestore.
	mattermost.Spec.Replicas = int32Ptr(0)
	mattermost.Spec.Size = ""
	if mattermost.Spec.Ingress != nil { // In case Installation was not yet updated and still uses old Ingress spec.
		mattermost.Spec.Ingress.Annotations = getHibernatingIngressAnnotations()
	} else {
		mattermost.Spec.IngressAnnotations = getHibernatingIngressAnnotations()
	}

	mattermost.Spec.ResourceLabels = clusterInstallationHibernatedLabels(installation, clusterInstallation)
}

func makeIngressSpec(installationDNS []*model.InstallationDNS) *mmv1beta1.Ingress {
	primaryRecord := installationDNS[0]
	for _, rec := range installationDNS {
		if rec.IsPrimary {
			primaryRecord = rec
			break
		}
	}

	ingressClass := "nginx-controller"
	return &mmv1beta1.Ingress{
		Enabled:      true,
		Host:         primaryRecord.DomainName,
		Hosts:        mapDomains(installationDNS),
		Annotations:  getIngressAnnotations(),
		IngressClass: &ingressClass,
	}
}

func mapDomains(installationDNS []*model.InstallationDNS) []mmv1beta1.IngressHost {
	hosts := make([]mmv1beta1.IngressHost, 0, len(installationDNS))

	for _, dns := range installationDNS {
		hosts = append(hosts, mmv1beta1.IngressHost{
			HostName: dns.DomainName,
		})
	}

	return hosts
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

func setMMInstanceSize(installation *model.Installation, mattermost *mmv1beta1.Mattermost) error {
	if strings.HasPrefix(installation.Size, model.ProvisionerSizePrefix) {
		resSize, err := model.ParseProvisionerSize(installation.Size)
		if err != nil {
			return errors.Wrap(err, "failed to parse custom installation size")
		}
		overrideReplicasAndResourcesFromSize(resSize, mattermost)
		return nil
	}
	mattermost.Spec.Size = installation.Size
	return nil
}

// This function is adapted from Mattermost Operator, we can make it public
// there to avoid copying.
func overrideReplicasAndResourcesFromSize(size v1alpha1.ClusterInstallationSize, mm *mmv1beta1.Mattermost) {
	mm.Spec.Size = ""

	mm.Spec.Replicas = utils.NewInt32(size.App.Replicas)
	mm.Spec.Scheduling.Resources = size.App.Resources
	mm.Spec.FileStore.OverrideReplicasAndResourcesFromSize(size)
	mm.Spec.Database.OverrideReplicasAndResourcesFromSize(size)
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

func clusterInstallationStableLabels(installation *model.Installation, clusterInstallation *model.ClusterInstallation) map[string]string {
	labels := clusterInstallationBaseLabels(installation, clusterInstallation)
	labels["state"] = "running"
	return labels
}

func clusterInstallationHibernatedLabels(installation *model.Installation, clusterInstallation *model.ClusterInstallation) map[string]string {
	labels := clusterInstallationBaseLabels(installation, clusterInstallation)
	labels["state"] = "hibernated"
	return labels
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
				  add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;`,
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

func setNdots(ndotsValue string) *corev1.PodDNSConfig {
	return &corev1.PodDNSConfig{Options: []corev1.PodDNSConfigOption{{Name: "ndots", Value: &ndotsValue}}}
}

// Override the version to make match the nil value in the custom resource.
// TODO: this could probably be better. We may want the operator to understand
// default values instead of needing to pass in empty values.
func translateMattermostVersion(version string) string {
	if version == "stable" {
		return ""
	}

	return version
}

func makeClusterInstallationName(clusterInstallation *model.ClusterInstallation) string {
	// TODO: Once https://mattermost.atlassian.net/browse/MM-15467 is fixed, we can use the
	// full namespace as part of the name. For now, truncate to keep within the existing limit
	// of 60 characters.
	return fmt.Sprintf("mm-%s", clusterInstallation.Namespace[0:4])
}
