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

	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-operator/apis/mattermost/v1alpha1"
	mmv1beta1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1beta1"
	"github.com/mattermost/mattermost-operator/pkg/resources"
	"github.com/mattermost/mattermost-operator/pkg/utils"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	bifrostEndpoint           = "bifrost.bifrost:80"
	ciExecJobTTLSeconds int32 = 180
)

// ClusterInstallationProvisioner function returns an implementation of ClusterInstallationProvisioner interface
// based on specified Custom Resource version.
func (provisioner Provisioner) ClusterInstallationProvisioner(version string) supervisor.ClusterInstallationProvisioner {
	if version != model.V1betaCRVersion {
		provisioner.logger.Errorf("Unexpected resource version: %s", version)
	}

	return provisioner
}

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

func (provisioner Provisioner) EnsureCRMigrated(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation) (bool, error) {
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
func (provisioner Provisioner) VerifyClusterInstallationMatchesConfig(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) (bool, error) {
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

// getMattermostCustomResource gets the cluster installation resource from
// the kubernetes API.
func (provisioner Provisioner) getMattermostCustomResource(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, logger log.FieldLogger) (*mmv1beta1.Mattermost, error) {
	configLocation, err := provisioner.getClusterKubecfg(cluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kube config path")
	}

	return getMattermostCustomResource(clusterInstallation, configLocation, logger)
}

// ExecMattermostCLI invokes the Mattermost CLI for the given cluster installation
// with the given args. Setup and exec errors both result in a single return error.
func (provisioner Provisioner) ExecMattermostCLI(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) ([]byte, error) {
	output, execErr, err := provisioner.ExecClusterInstallationCLI(cluster, clusterInstallation, append([]string{"./bin/mattermost"}, args...)...)
	if err != nil {
		return output, errors.Wrap(err, "failed to run mattermost command")
	}
	if execErr != nil {
		return output, errors.Wrap(execErr, "mattermost command encountered an error")
	}

	return output, nil
}

// ExecMMCTL runs the given MMCTL command against the given cluster installation.
// Setup and exec errors both result in a single return error.
func (provisioner Provisioner) ExecMMCTL(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) ([]byte, error) {
	output, execErr, err := provisioner.ExecClusterInstallationCLI(cluster, clusterInstallation, append([]string{"./bin/mmctl"}, args...)...)
	if err != nil {
		return output, errors.Wrap(err, "failed to run mmctl command")
	}
	if execErr != nil {
		return output, errors.Wrap(execErr, "mmctl command encountered an error")
	}

	return output, nil
}

func execClusterInstallationCLI(k8sClient *k8s.KubeClient, clusterInstallation *model.ClusterInstallation, logger log.FieldLogger, args ...string) ([]byte, error, error) {
	ctx := context.TODO()
	podList, err := k8sClient.Clientset.CoreV1().Pods(clusterInstallation.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=mattermost",
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to query mattermost pods")
	}

	// In the future, we'd ideally just spin our own container on demand, allowing
	// configuration changes even if the pods are failing to start the server. For now,
	// we find the first pod running Mattermost, and pick the first container therein.

	if len(podList.Items) == 0 {
		return nil, nil, errors.New("failed to find mattermost pods on which to exec")
	}

	pod := podList.Items[0]
	if len(pod.Spec.Containers) == 0 {
		return nil, nil, errors.Errorf("failed to find containers in pod %s", pod.Name)
	}

	container := pod.Spec.Containers[0]
	logger.Debugf("Executing `%s` on pod %s: container=%s, image=%s, phase=%s", strings.Join(args, " "), pod.Name, container.Name, container.Image, pod.Status.Phase)

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
	output, execErr := k8sClient.RemoteCommand("POST", execRequest.URL())

	if execErr != nil {
		logger.WithError(execErr).Warnf("Command `%s` on pod %s finished in %.0f seconds, but encountered an error", strings.Join(args, " "), pod.Name, time.Since(now).Seconds())
	} else {
		logger.Debugf("Command `%s` on pod %s finished in %.0f seconds", strings.Join(args, " "), pod.Name, time.Since(now).Seconds())
	}

	return output, execErr, nil
}

// ExecClusterInstallationCLI execs the provided command on the defined cluster
// installation and returns both exec preparation errors as well as errors from
// the exec command itself.
func (provisioner Provisioner) ExecClusterInstallationCLI(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) ([]byte, error, error) {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	k8sClient, err := provisioner.k8sClient(cluster)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get kube client")
	}

	return execClusterInstallationCLI(k8sClient, clusterInstallation, logger, args...)
}

// ExecClusterInstallationJob creates job executing command on cluster installation.
func (provisioner Provisioner) ExecClusterInstallationJob(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})
	logger.Info("Executing job with CLI command on cluster installation")

	k8sClient, err := provisioner.k8sClient(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to get kube client")
	}

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
	job := resources.PrepareMattermostJobTemplate(jobName, clusterInstallation.Namespace, &deploymentList.Items[0], nil)
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
		errDefer := jobsClient.Delete(ctx, jobName, metav1.DeleteOptions{})
		if errDefer != nil && !k8sErrors.IsNotFound(errDefer) {
			logger.Errorf("Failed to cleanup exec job: %q", jobName)
		}
	}()

	job, err = jobsClient.Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create CLI command job")
	}

	err = wait.Poll(time.Second, 10*time.Minute, func() (bool, error) {
		job, err = jobsClient.Get(ctx, jobName, metav1.GetOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "failed to get %q job", jobName)
		}
		if job.Status.Succeeded < 1 {
			logger.Infof("job %q not yet finished, waiting up to 10 minute", jobName)
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

// PrepareClusterUtilities performs any updates to cluster utilities that may
// be needed for clusterinstallations to function correctly.
func (provisioner Provisioner) PrepareClusterUtilities(cluster *model.Cluster, installation *model.Installation, store model.ClusterUtilityDatabaseStoreInterface, awsClient aws.AWS) error {
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

	return prepareClusterUtilities(cluster, configLocation, store, awsClient, provisioner.params.PGBouncerConfig, logger)
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

func ensureEnvMatch(wanted corev1.EnvVar, all []corev1.EnvVar) bool {
	for _, env := range all {
		if env == wanted {
			return true
		}
	}

	return false
}

func setNdots(ndotsValue string) *corev1.PodDNSConfig {
	return &corev1.PodDNSConfig{Options: []corev1.PodDNSConfigOption{{Name: "ndots", Value: &ndotsValue}}}
}
