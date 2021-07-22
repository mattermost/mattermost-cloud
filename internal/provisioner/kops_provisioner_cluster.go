// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/internal/tools/terraform"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	rotatorModel "github.com/mattermost/rotator/model"
	"github.com/mattermost/rotator/rotator"
)

// DefaultKubernetesVersion is the default value for a kubernetes cluster
// version value.
const (
	DefaultKubernetesVersion = "0.0.0"
	igFilename               = "ig-nodes.yaml"
)

// PrepareCluster ensures a cluster object is ready for provisioning.
func (provisioner *KopsProvisioner) PrepareCluster(cluster *model.Cluster) bool {
	// Don't regenerate the name if already set.
	if cluster.ProvisionerMetadataKops.Name != "" {
		return false
	}

	// Generate the kops name using the cluster id.
	cluster.ProvisionerMetadataKops.Name = fmt.Sprintf("%s-kops.k8s.local", cluster.ID)

	return true
}

// CreateCluster creates a cluster using kops and terraform.
func (provisioner *KopsProvisioner) CreateCluster(cluster *model.Cluster, awsClient aws.AWS) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	kopsMetadata := cluster.ProvisionerMetadataKops

	err := kopsMetadata.ValidateChangeRequest()
	if err != nil {
		return errors.Wrap(err, "KopsMetadata ChangeRequest failed validation")
	}

	if kopsMetadata.ChangeRequest.AMI != "" && kopsMetadata.ChangeRequest.AMI != "latest" {
		isAMIValid, err := awsClient.IsValidAMI(kopsMetadata.ChangeRequest.AMI, logger)
		if err != nil {
			return errors.Wrapf(err, "error checking the AWS AMI image %s", kopsMetadata.ChangeRequest.AMI)
		}
		if !isAMIValid {
			return errors.Errorf("invalid AWS AMI image %s", kopsMetadata.ChangeRequest.AMI)
		}
	}

	cncVPCName := fmt.Sprintf("mattermost-cloud-%s-command-control", awsClient.GetCloudEnvironmentName())
	cncVPCCIDR, err := awsClient.GetCIDRByVPCTag(cncVPCName, logger)
	if err != nil {
		return errors.Wrapf(err, "failed to get the CIDR for the VPC Name %s", cncVPCName)
	}
	allowSSHCIDRS := []string{cncVPCCIDR}
	allowSSHCIDRS = append(allowSSHCIDRS, provisioner.params.VpnCIDRList...)

	logger.WithField("name", kopsMetadata.Name).Info("Creating cluster")
	kops, err := kops.New(provisioner.params.S3StateStore, logger)
	if err != nil {
		return err
	}
	defer kops.Close()

	var clusterResources aws.ClusterResources
	if kopsMetadata.ChangeRequest.VPC != "" && provisioner.params.UseExistingAWSResources {
		clusterResources, err = awsClient.GetVpcResourcesByVpcID(kopsMetadata.ChangeRequest.VPC, logger)
		if err != nil {
			return err
		}
	} else if provisioner.params.UseExistingAWSResources {
		clusterResources, err = awsClient.GetAndClaimVpcResources(cluster.ID, provisioner.params.Owner, logger)
		if err != nil {
			return err
		}
	}

	err = kops.CreateCluster(
		kopsMetadata.Name,
		cluster.Provider,
		kopsMetadata.ChangeRequest,
		cluster.ProviderMetadataAWS.Zones,
		clusterResources.PrivateSubnetIDs,
		clusterResources.PublicSubnetsIDs,
		clusterResources.MasterSecurityGroupIDs,
		clusterResources.WorkerSecurityGroupIDs,
		allowSSHCIDRS,
	)
	// release VPC resources
	if err != nil {
		releaseErr := awsClient.ReleaseVpc(cluster.ID, logger)
		if releaseErr != nil {
			logger.WithError(releaseErr).Error("Unable to release VPC")
		}

		return errors.Wrap(err, "unable to create kops cluster")
	}
	// Tag Public subnets & respective VPC for the secondary cluster if there is no error.
	if kopsMetadata.ChangeRequest.VPC != "" {
		err = awsClient.TagResourcesByCluster(clusterResources, cluster.ID, provisioner.params.Owner, logger)
		if err != nil {
			return err
		}
	}
	terraformClient, err := terraform.New(kops.GetOutputDirectory(), provisioner.params.S3StateStore, logger)
	if err != nil {
		return err
	}
	defer terraformClient.Close()

	err = terraformClient.Init(kopsMetadata.Name)
	if err != nil {
		return err
	}

	err = terraformClient.ApplyTarget(fmt.Sprintf("aws_internet_gateway.%s-kops-k8s-local", cluster.ID))
	if err != nil {
		return err
	}

	err = terraformClient.ApplyTarget(fmt.Sprintf("aws_elb.api-%s-kops-k8s-local", cluster.ID))
	if err != nil {
		return err
	}

	// TODO: read from config file
	logger.Info("Updating kubelet options")
	setValue := "spec.kubelet.authenticationTokenWebhook=true"
	err = kops.SetCluster(kopsMetadata.Name, setValue)
	if err != nil {
		return err
	}
	setValue = "spec.kubelet.authorizationMode=Webhook"
	err = kops.SetCluster(kopsMetadata.Name, setValue)
	if err != nil {
		return err
	}

	err = kops.UpdateCluster(kopsMetadata.Name, kops.GetOutputDirectory())
	if err != nil {
		return err
	}

	err = terraformClient.Apply()
	if err != nil {
		return err
	}

	// TODO: Rework this as we make the API calls asynchronous.
	wait := 1000
	logger.Infof("Waiting up to %d seconds for k8s cluster to become ready...", wait)
	err = kops.WaitForKubernetesReadiness(kopsMetadata.Name, wait)
	if err != nil {
		// Run non-silent validate one more time to log final cluster state
		// and return original timeout error.
		kops.ValidateCluster(kopsMetadata.Name, false)
		return err
	}

	logger.WithField("name", kopsMetadata.Name).Info("Successfully deployed kubernetes")

	logger.WithField("name", kopsMetadata.Name).Info("Updating VolumeBindingMode in default storage class")
	k8sClient, err := k8s.NewFromFile(kops.GetKubeConfigPath(), logger)
	if err != nil {
		return err
	}

	_, err = k8sClient.UpdateStorageClassVolumeBindingMode("gp2")
	if err != nil {
		return err
	}
	logger.WithField("name", kopsMetadata.Name).Info("Successfully updated storage class")

	iamRole := fmt.Sprintf("nodes.%s", kopsMetadata.Name)
	err = awsClient.AttachPolicyToRole(iamRole, aws.CustomNodePolicyName, logger)
	if err != nil {
		return errors.Wrap(err, "unable to attach custom node policy")
	}

	ugh, err := newUtilityGroupHandle(kops, provisioner, cluster, awsClient, logger)
	if err != nil {
		return err
	}

	return ugh.CreateUtilityGroup()
}

// ProvisionCluster installs all the baseline kubernetes resources needed for
// managing installations. This can be called on an already-provisioned cluster
// to re-provision with the newest version of the resources.
func (provisioner *KopsProvisioner) ProvisionCluster(cluster *model.Cluster, awsClient aws.AWS) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	logger.Info("Provisioning cluster")
	kopsClient, err := provisioner.getCachedKopsClient(cluster.ProvisionerMetadataKops.Name, logger)
	if err != nil {
		return errors.Wrap(err, "failed to get kops client from cache")
	}
	defer provisioner.invalidateCachedKopsClientOnError(err, cluster.ProvisionerMetadataKops.Name, logger)

	// Start by gathering resources that will be needed later. If any of this
	// fails then no cluster changes have been made which reduces risk.
	bifrostSecret, err := awsClient.GenerateBifrostUtilitySecret(cluster.ID, logger)
	if err != nil {
		return errors.Wrap(err, "failed to generate bifrost secret")
	}

	// Begin deploying the mattermost operator.
	k8sClient, err := k8s.NewFromFile(kopsClient.GetKubeConfigPath(), logger)
	if err != nil {
		return err
	}

	mysqlOperatorNamespace := "mysql-operator"
	minioOperatorNamespace := "minio-operator"
	mattermostOperatorNamespace := "mattermost-operator"

	namespaces := []string{
		mattermostOperatorNamespace,
	}

	if provisioner.params.DeployMysqlOperator {
		namespaces = append(namespaces, mysqlOperatorNamespace)
	}

	if provisioner.params.DeployMinioOperator {
		namespaces = append(namespaces, minioOperatorNamespace)
	}

	// Remove all previously-installed operator namespaces and resources.
	ctx := context.TODO()
	for _, namespace := range namespaces {
		logger.Infof("Cleaning up namespace %s", namespace)
		err = k8sClient.Clientset.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
		if k8sErrors.IsNotFound(err) {
			logger.Infof("Namespace %s not found; skipping...", namespace)
		} else if err != nil {
			return errors.Wrapf(err, "failed to delete namespace %s", namespace)
		}
	}

	wait := 60
	logger.Infof("Waiting up to %d seconds for namespaces to be terminated...", wait)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(wait)*time.Second)
	defer cancel()
	err = waitForNamespacesDeleted(ctx, namespaces, k8sClient)
	if err != nil {
		return err
	}

	// The bifrost utility cannot have downtime so it is not part of the namespace
	// cleanup and recreation flow. We always only update bifrost.
	bifrostNamespace := "bifrost"
	namespaces = append(namespaces, bifrostNamespace)
	logger.Info("Creating utility namespaces")
	_, err = k8sClient.CreateOrUpdateNamespaces(namespaces)
	if err != nil {
		return errors.Wrap(err, "failed to create bifrost namespace")
	}

	logger.Info("Creating or updating bifrost secret")
	_, err = k8sClient.CreateOrUpdateSecret(bifrostNamespace, bifrostSecret)
	if err != nil {
		return errors.Wrap(err, "failed to create bifrost secret")
	}

	// Need to remove two items from the calico because the fields after the creation are immutable so the
	// create/update does not work. We might want to refactor this in the future to avoid this
	logger.Info("Cleaning up some calico resources to reapply")
	err = k8sClient.Clientset.CoreV1().Services("kube-system").Delete(ctx, "calico-typha", metav1.DeleteOptions{})
	if k8sErrors.IsNotFound(err) {
		logger.Info("Service calico-typha not found; skipping...")
	} else if err != nil {
		return errors.Wrap(err, "failed to delete service calico-typha")
	}

	err = k8sClient.Clientset.PolicyV1beta1().PodDisruptionBudgets("kube-system").Delete(ctx, "calico-typha", metav1.DeleteOptions{})
	if k8sErrors.IsNotFound(err) {
		logger.Info("PodDisruptionBudget calico-typha not found; skipping...")
	} else if err != nil {
		return errors.Wrap(err, "failed to delete PodDisruptionBudget calico-typha")
	}

	// Need to remove two items from the metrics-server because the fields after the creation are immutable so the
	// create/update does not work. We might want to refactor this in the future to avoid this
	logger.Info("Cleaning up some metrics-server resources to reapply")
	err = k8sClient.Clientset.CoreV1().Services("kube-system").Delete(ctx, "metrics-server", metav1.DeleteOptions{})
	if k8sErrors.IsNotFound(err) {
		logger.Info("Service metrics-server not found; skipping...")
	} else if err != nil {
		return errors.Wrap(err, "failed to delete service metrics-server")
	}

	err = k8sClient.KubeagClientSet.ApiregistrationV1beta1().APIServices().Delete(ctx, "v1beta1.metrics.k8s.io", metav1.DeleteOptions{})
	if k8sErrors.IsNotFound(err) {
		logger.Info("APIService v1beta1.metrics.k8s.io not found; skipping...")
	} else if err != nil {
		return errors.Wrap(err, "failed to delete APIService v1beta1.metrics.k8s.io")
	}

	// TODO: determine if we want to hard-code the k8s resource objects in code.
	// For now, we will ingest manifest files to deploy the mattermost operator.
	files := []k8s.ManifestFile{
		{
			Path:            "manifests/operator-manifests/mattermost/crds/mm_clusterinstallation_crd.yaml",
			DeployNamespace: mattermostOperatorNamespace,
		}, {
			Path:            "manifests/operator-manifests/mattermost/crds/mm_mattermostrestoredb_crd.yaml",
			DeployNamespace: mattermostOperatorNamespace,
		}, {
			Path:            "manifests/operator-manifests/mattermost/crds/mm_mattermost_crd.yaml",
			DeployNamespace: mattermostOperatorNamespace,
		}, {
			Path:            "manifests/operator-manifests/mattermost/service_account.yaml",
			DeployNamespace: mattermostOperatorNamespace,
		}, {
			Path:            "manifests/operator-manifests/mattermost/role.yaml",
			DeployNamespace: mattermostOperatorNamespace,
		}, {
			Path:            "manifests/operator-manifests/mattermost/role_binding.yaml",
			DeployNamespace: mattermostOperatorNamespace,
		}, {
			Path:            "manifests/operator-manifests/mattermost/operator.yaml",
			DeployNamespace: mattermostOperatorNamespace,
		}, {
			Path:            "manifests/bifrost/bifrost.yaml",
			DeployNamespace: bifrostNamespace,
		}, {
			Path:            "manifests/calico-policy-only.yaml",
			DeployNamespace: "kube-system",
		}, {
			Path:            "manifests/metric-server/metric-server.yaml",
			DeployNamespace: "kube-system",
		}, {
			Path:            "manifests/k8s-spot-termination-handler/k8s-spot-termination-handler.yaml",
			DeployNamespace: "kube-system",
		},
	}

	if provisioner.params.DeployMysqlOperator {
		files = append(files, k8s.ManifestFile{
			Path:            "manifests/operator-manifests/mysql/mysql-operator.yaml",
			DeployNamespace: mysqlOperatorNamespace,
		})
	}

	if provisioner.params.DeployMinioOperator {
		files = append(files, k8s.ManifestFile{
			Path:            "manifests/operator-manifests/minio/minio-operator.yaml",
			DeployNamespace: minioOperatorNamespace,
		})
	}

	err = k8sClient.CreateFromFiles(files)
	if err != nil {
		return err
	}

	// change the waiting time because creation can take more time
	// due container download / init / container creation / volume allocation
	wait = 240
	appsWithDeployment := map[string]string{
		"mattermost-operator":                mattermostOperatorNamespace,
		"bifrost":                            bifrostNamespace,
		"calico-typha-horizontal-autoscaler": "kube-system",
		"calico-typha":                       "kube-system",
	}

	if provisioner.params.DeployMinioOperator {
		appsWithDeployment["minio-operator"] = minioOperatorNamespace
	}

	for deployment, namespace := range appsWithDeployment {
		pods, err := k8sClient.GetPodsFromDeployment(namespace, deployment)
		if err != nil {
			return err
		}
		if len(pods.Items) == 0 {
			return fmt.Errorf("no pods found from %q deployment", deployment)
		}

		for _, pod := range pods.Items {
			logger.Infof("Waiting up to %d seconds for %q pod %q to start...", wait, deployment, pod.GetName())
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(wait)*time.Second)
			defer cancel()
			_, err := k8sClient.WaitForPodRunning(ctx, namespace, pod.GetName())
			if err != nil {
				return err
			}
			logger.Infof("Successfully deployed service pod %q", pod.GetName())
		}
	}

	var operatorsWithStatefulSet []string
	if provisioner.params.DeployMysqlOperator {
		operatorsWithStatefulSet = append(operatorsWithStatefulSet, "mysql-operator")
	}

	for _, operator := range operatorsWithStatefulSet {
		pods, err := k8sClient.GetPodsFromStatefulset(operator, operator)
		if err != nil {
			return err
		}
		if len(pods.Items) == 0 {
			return fmt.Errorf("no pods found from %q statefulSet", operator)
		}

		for _, pod := range pods.Items {

			logger.Infof("Waiting up to %d seconds for %q pod %q to start...", wait, operator, pod.GetName())
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(wait)*time.Second)
			defer cancel()
			pod, err := k8sClient.WaitForPodRunning(ctx, operator, pod.GetName())
			if err != nil {
				return err
			}
			logger.Infof("Successfully deployed service pod %q", pod.GetName())
		}
	}

	supportAppsWithDaemonSets := map[string]string{
		"calico-node": "kube-system",
		"k8s-spot-termination-handler": "kube-system",
	}
	for daemonSet, namespace := range supportAppsWithDaemonSets {
		if daemonSet == "k8s-spot-termination-handler" && (len(os.Getenv(model.MattermostChannel)) > 0 || len(os.Getenv(model.MattermostWebhook)) > 0){
			daemonSetObj, err := k8sClient.Clientset.AppsV1().DaemonSets(namespace).Get(ctx, daemonSet, metav1.GetOptions{})
			if err != nil {
				return errors.Wrapf(err, "failed to get daemonSet %s", daemonSet)
			}

			var payload []k8s.PatchStringValue
			for i, envVar := range daemonSetObj.Spec.Template.Spec.Containers[0].Env {
				if envVar.Name == "CLUSTER" {
					payload = []k8s.PatchStringValue{{
						Op:    "replace",
						Path:  "/spec/template/spec/containers/0/env/" + strconv.Itoa(i) + "/value",
						Value: cluster.ID,
					}}
				}
				if envVar.Name == "MATTERMOST_CHANNEL" && len(os.Getenv(model.MattermostChannel)) > 0 {
					payload = append(payload,
						k8s.PatchStringValue{
							Op:    "replace",
							Path:  "/spec/template/spec/containers/0/env/" + strconv.Itoa(i) + "/value",
							Value: os.Getenv(model.MattermostChannel),
						})
				}
				if envVar.Name == "MATTERMOST_WEBHOOK" && len(os.Getenv(model.MattermostWebhook)) > 0 {
					payload = append(payload,
						k8s.PatchStringValue{
							Op:    "replace",
							Path:  "/spec/template/spec/containers/0/env/" + strconv.Itoa(i) + "/value",
							Value: os.Getenv(model.MattermostWebhook),
						})
				}
			}

			err = k8sClient.PatchPodsDaemonSet("kube-system", "k8s-spot-termination-handler", payload)
			if err != nil {
				return err
			}
		}
		pods, err := k8sClient.GetPodsFromDaemonSet(namespace, daemonSet)
		if err != nil {
			return err
		}
		// Pods for k8s-spot-termination-handler do not ment to be schedule in every cluster so doesn't need to fail provision in this case/
		if len(pods.Items) == 0 && daemonSet != "k8s-spot-termination-handler" {
			return fmt.Errorf("no pods found from %s/%s daemonSet", namespace, daemonSet)
		}

		for _, pod := range pods.Items {
			logger.Infof("Waiting up to %d seconds for %q/%q pod %q to start...", wait, namespace, daemonSet, pod.GetName())
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(wait)*time.Second)
			defer cancel()
			pod, err := k8sClient.WaitForPodRunning(ctx, namespace, pod.GetName())
			if err != nil {
				return err
			}
			logger.Infof("Successfully deployed support apps pod %q", pod.GetName())
		}
	}
	iamRole := fmt.Sprintf("nodes.%s", cluster.ProvisionerMetadataKops.Name)
	err = awsClient.AttachPolicyToRole(iamRole, aws.CustomNodePolicyName, logger)
	if err != nil {
		return errors.Wrap(err, "unable to attach custom node policy")
	}

	ugh, err := newUtilityGroupHandle(kopsClient, provisioner, cluster, awsClient, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create new cluster utility group handle")
	}

	err = ugh.ProvisionUtilityGroup()
	if err != nil {
		return errors.Wrap(err, "failed to upgrade all services in utility group")
	}

	logger.WithField("name", cluster.ProvisionerMetadataKops.Name).Info("Successfully provisioned cluster")

	return nil
}

// UpgradeCluster upgrades a cluster to the latest recommended production ready k8s version.
func (provisioner *KopsProvisioner) UpgradeCluster(cluster *model.Cluster, awsClient aws.AWS) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	kopsMetadata := cluster.ProvisionerMetadataKops

	err := kopsMetadata.ValidateChangeRequest()
	if err != nil {
		return errors.Wrap(err, "KopsMetadata ChangeRequest failed validation")
	}

	if kopsMetadata.ChangeRequest.AMI != "" && kopsMetadata.ChangeRequest.AMI != "latest" {
		isAMIValid, err := awsClient.IsValidAMI(kopsMetadata.ChangeRequest.AMI, logger)
		if err != nil {
			return errors.Wrapf(err, "error checking the AWS AMI image %s", kopsMetadata.ChangeRequest.AMI)
		}
		if !isAMIValid {
			return errors.Errorf("invalid AWS AMI image %s", kopsMetadata.ChangeRequest.AMI)
		}
	}

	kops, err := kops.New(provisioner.params.S3StateStore, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()

	switch kopsMetadata.ChangeRequest.Version {
	case "":
		logger.Info("Skipping kubernetes cluster version update")
	case "latest":
		logger.Info("Updating kubernetes to latest stable version")
		err = kops.UpgradeCluster(kopsMetadata.Name)
		if err != nil {
			return err
		}
	default:
		logger.Infof("Updating kubernetes to version %s", kopsMetadata.ChangeRequest.Version)
		setValue := fmt.Sprintf("spec.kubernetesVersion=%s", kopsMetadata.ChangeRequest.Version)
		err = kops.SetCluster(kopsMetadata.Name, setValue)
		if err != nil {
			return err
		}
	}

	err = updateKopsInstanceGroupAMIs(kops, kopsMetadata, logger)
	if err != nil {
		return errors.Wrap(err, "failed to update kops instance group AMIs")
	}

	// TODO: read from config file
	// TODO: check if those configs are already or remove this when we update all clusters
	logger.Info("Updating kubelet options")
	setValue := "spec.kubelet.authenticationTokenWebhook=true"
	err = kops.SetCluster(kopsMetadata.Name, setValue)
	if err != nil {
		return err
	}
	setValue = "spec.kubelet.authorizationMode=Webhook"
	err = kops.SetCluster(kopsMetadata.Name, setValue)
	if err != nil {
		return err
	}

	err = kops.UpdateCluster(kopsMetadata.Name, kops.GetOutputDirectory())
	if err != nil {
		return err
	}

	terraformClient, err := terraform.New(kops.GetOutputDirectory(), provisioner.params.S3StateStore, logger)
	if err != nil {
		return err
	}
	defer terraformClient.Close()

	err = terraformClient.Init(kopsMetadata.Name)
	if err != nil {
		return err
	}

	err = verifyTerraformAndKopsMatch(kopsMetadata.Name, terraformClient, logger)
	if err != nil {
		return err
	}

	logger.Info("Upgrading cluster")

	err = terraformClient.Plan()
	if err != nil {
		return err
	}
	err = terraformClient.Apply()
	if err != nil {
		return err
	}

	if cluster.ProvisionerMetadataKops.RotatorRequest.Config != nil {
		if *cluster.ProvisionerMetadataKops.RotatorRequest.Config.UseRotator {
			logger.Info("Using node rotator for node upgrade")
			err = provisioner.RotateClusterNodes(cluster)
			if err != nil {
				return err
			}
		}
	}

	err = kops.RollingUpdateCluster(kopsMetadata.Name)
	if err != nil {
		return err
	}

	// TODO: Rework this as we make the API calls asynchronous.
	wait := 1000
	if wait > 0 {
		logger.Infof("Waiting up to %d seconds for k8s cluster to become ready...", wait)
		err = kops.WaitForKubernetesReadiness(kopsMetadata.Name, wait)
		if err != nil {
			// Run non-silent validate one more time to log final cluster state
			// and return original timeout error.
			kops.ValidateCluster(kopsMetadata.Name, false)
			return err
		}
	}

	iamRole := fmt.Sprintf("nodes.%s", kopsMetadata.Name)
	err = awsClient.AttachPolicyToRole(iamRole, aws.CustomNodePolicyName, logger)
	if err != nil {
		return errors.Wrap(err, "unable to attach custom node policy")
	}

	logger.Info("Successfully upgraded cluster")

	return nil
}

// RotateClusterNodes rotates k8s cluster nodes using the Mattermost node rotator
func (provisioner *KopsProvisioner) RotateClusterNodes(cluster *model.Cluster) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	kopsClient, err := provisioner.getCachedKopsClient(cluster.ProvisionerMetadataKops.Name, logger)
	if err != nil {
		return errors.Wrap(err, "failed to get kops client from cache")
	}
	defer provisioner.invalidateCachedKopsClientOnError(err, cluster.ProvisionerMetadataKops.Name, logger)

	k8sClient, err := k8s.NewFromFile(kopsClient.GetKubeConfigPath(), logger)
	if err != nil {
		return err
	}
	clientset, err := kubernetes.NewForConfig(k8sClient.GetConfig())

	clusterRotator := rotatorModel.Cluster{
		ClusterID:            cluster.ID,
		MaxScaling:           *cluster.ProvisionerMetadataKops.RotatorRequest.Config.MaxScaling,
		RotateMasters:        true,
		RotateWorkers:        true,
		MaxDrainRetries:      *cluster.ProvisionerMetadataKops.RotatorRequest.Config.MaxDrainRetries,
		EvictGracePeriod:     *cluster.ProvisionerMetadataKops.RotatorRequest.Config.EvictGracePeriod,
		WaitBetweenRotations: *cluster.ProvisionerMetadataKops.RotatorRequest.Config.WaitBetweenRotations,
		WaitBetweenDrains:    *cluster.ProvisionerMetadataKops.RotatorRequest.Config.WaitBetweenDrains,
		ClientSet:            clientset,
	}

	rotatorMetadata := cluster.ProvisionerMetadataKops.RotatorRequest.Status
	if rotatorMetadata == nil {
		rotatorMetadata = &rotator.RotatorMetadata{}
	}
	rotatorMetadata, err = rotator.InitRotateCluster(&clusterRotator, rotatorMetadata, logger)
	if err != nil {
		cluster.ProvisionerMetadataKops.RotatorRequest.Status = rotatorMetadata
		return err
	}

	return nil
}

// ResizeCluster resizes a cluster.
func (provisioner *KopsProvisioner) ResizeCluster(cluster *model.Cluster, awsClient aws.AWS) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	kopsMetadata := cluster.ProvisionerMetadataKops

	err := kopsMetadata.ValidateChangeRequest()
	if err != nil {
		return errors.Wrap(err, "KopsMetadata ChangeRequest failed validation")
	}

	kops, err := kops.New(provisioner.params.S3StateStore, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()

	err = kops.UpdateCluster(kopsMetadata.Name, kops.GetOutputDirectory())
	if err != nil {
		return err
	}

	terraformClient, err := terraform.New(kops.GetOutputDirectory(), provisioner.params.S3StateStore, logger)
	if err != nil {
		return err
	}
	defer terraformClient.Close()

	err = terraformClient.Init(kopsMetadata.Name)
	if err != nil {
		return err
	}

	err = verifyTerraformAndKopsMatch(kopsMetadata.Name, terraformClient, logger)
	if err != nil {
		return err
	}

	logger.Info("Resizing cluster")

	for igName, changeMetadata := range kopsMetadata.GetWorkerNodesResizeChanges() {
		logger.Infof("Resizing instance group %s to %d nodes", igName, changeMetadata.NodeMinCount)

		igManifest, err := kops.GetInstanceGroupYAML(kopsMetadata.Name, igName)
		if err != nil {
			return err
		}

		igManifest, err = grossKopsReplaceSize(
			igManifest,
			kopsMetadata.ChangeRequest.NodeInstanceType,
			fmt.Sprintf("%d", changeMetadata.NodeMinCount),
			fmt.Sprintf("%d", changeMetadata.NodeMaxCount),
		)
		if err != nil {
			return errors.Wrap(err, "failed to update instance group yaml file")
		}

		err = ioutil.WriteFile(path.Join(kops.GetTempDir(), igFilename), []byte(igManifest), 0600)
		if err != nil {
			return errors.Wrap(err, "failed to write instance group yaml file")
		}
		_, err = kops.Replace(igFilename)
		if err != nil {
			return errors.Wrap(err, "failed to replace instance group resources")
		}
	}

	err = kops.UpdateCluster(kopsMetadata.Name, kops.GetOutputDirectory())
	if err != nil {
		return err
	}

	err = terraformClient.Plan()
	if err != nil {
		return err
	}
	err = terraformClient.Apply()
	if err != nil {
		return err
	}

	err = kops.RollingUpdateCluster(kopsMetadata.Name)
	if err != nil {
		return err
	}

	// TODO: Rework this as we make the API calls asynchronous.
	wait := 1000
	if wait > 0 {
		logger.Infof("Waiting up to %d seconds for k8s cluster to become ready...", wait)
		err = kops.WaitForKubernetesReadiness(kopsMetadata.Name, wait)
		if err != nil {
			// Run non-silent validate one more time to log final cluster state
			// and return original timeout error.
			kops.ValidateCluster(kopsMetadata.Name, false)
			return err
		}
	}

	iamRole := fmt.Sprintf("nodes.%s", kopsMetadata.Name)
	err = awsClient.AttachPolicyToRole(iamRole, aws.CustomNodePolicyName, logger)
	if err != nil {
		return errors.Wrap(err, "unable to attach custom node policy")
	}

	logger.Info("Successfully resized cluster")

	return nil
}

// DeleteCluster deletes a previously created cluster using kops and terraform.
func (provisioner *KopsProvisioner) DeleteCluster(cluster *model.Cluster, awsClient aws.AWS) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	kopsMetadata := cluster.ProvisionerMetadataKops

	// Use these vars to keep track of which resources need to be deleted.
	var skipDeleteKops, skipDeleteTerraform bool

	logger.Info("Deleting cluster")

	kopsClient, err := provisioner.getCachedKopsClient(kopsMetadata.Name, logger)
	if err != nil {
		return errors.Wrap(err, "failed to get kops client from cache")
	}
	defer provisioner.invalidateCachedKopsClientOnError(err, kopsMetadata.Name, logger)

	ugh, err := newUtilityGroupHandle(kopsClient, provisioner, cluster, awsClient, logger)
	if err != nil {
		return errors.Wrap(err, "couldn't create new utility group handle while deleting the cluster")
	}

	err = ugh.DestroyUtilityGroup()
	if err != nil {
		return errors.Wrap(err, "failed to destroy all services in the utility group")
	}

	iamRole := fmt.Sprintf("nodes.%s", kopsMetadata.Name)
	err = awsClient.DetachPolicyFromRole(iamRole, aws.CustomNodePolicyName, logger)
	if err != nil {
		return errors.Wrap(err, "unable to detach custom node policy")
	}

	_, err = kopsClient.GetCluster(kopsMetadata.Name)
	if err != nil {
		logger.WithError(err).Error("Failed kops get cluster check: proceeding assuming kops and terraform resources were never created")
		skipDeleteKops = true
		skipDeleteTerraform = true
	}

	if !skipDeleteKops {
		err = kopsClient.UpdateCluster(kopsMetadata.Name, kopsClient.GetOutputDirectory())
		if err != nil {
			return errors.Wrap(err, "failed to run kops update")
		}
	}

	if !skipDeleteTerraform {
		terraformClient, err := terraform.New(kopsClient.GetOutputDirectory(), provisioner.params.S3StateStore, logger)
		if err != nil {
			return errors.Wrap(err, "failed to create terraform wrapper")
		}
		defer terraformClient.Close()

		err = terraformClient.Init(kopsMetadata.Name)
		if err != nil {
			return errors.Wrap(err, "failed to init terraform")
		}

		err = verifyTerraformAndKopsMatch(kopsMetadata.Name, terraformClient, logger)
		if err != nil {
			skipDeleteTerraform = true
			logger.WithError(err).Error("Proceeding with cluster deletion despite failing terraform output match check")
		}

		err = terraformClient.Destroy()
		if err != nil {
			return errors.Wrap(err, "failed to run terraform destroy")
		}
	}

	if !skipDeleteKops {
		err = kopsClient.DeleteCluster(kopsMetadata.Name)
		if err != nil {
			return errors.Wrap(err, "failed to run kops delete")
		}
	}

	err = awsClient.ReleaseVpc(cluster.ID, logger)
	if err != nil {
		return errors.Wrap(err, "failed to release cluster VPC")
	}

	provisioner.invalidateCachedKopsClient(kopsMetadata.Name, logger)

	logger.Info("Successfully deleted cluster")

	return nil
}

// GetClusterResources returns a snapshot of resources of a given cluster.
func (provisioner *KopsProvisioner) GetClusterResources(cluster *model.Cluster, onlySchedulable bool) (*k8s.ClusterResources, error) {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

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
	allPods, err := k8sClient.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	usedCPU, usedMemory := k8s.CalculateTotalPodMilliResourceRequests(allPods)

	var totalCPU, totalMemory int64
	nodes, err := k8sClient.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, node := range nodes.Items {
		var skipNode bool

		if onlySchedulable {
			if node.Spec.Unschedulable {
				logger.Debugf("Ignoring unschedulable node %s", node.GetName())
				skipNode = true
			}

			// TODO: handle scheduling taints in a more robust way.
			// This is a quick and dirty check for scheduling issues that could
			// lead to false positives. In the future, we should use a scheduling
			// library to perform the check instead.
			for _, taint := range node.Spec.Taints {
				if taint.Effect == v1.TaintEffectNoSchedule || taint.Effect == v1.TaintEffectPreferNoSchedule {
					logger.Debugf("Ignoring node %s with taint '%s'", node.GetName(), taint.ToString())
					skipNode = true
					break
				}
			}
		}

		if !skipNode {
			totalCPU += node.Status.Allocatable.Cpu().MilliValue()
			totalMemory += node.Status.Allocatable.Memory().MilliValue()
		}
	}

	return &k8s.ClusterResources{
		MilliTotalCPU:    totalCPU,
		MilliUsedCPU:     usedCPU,
		MilliTotalMemory: totalMemory,
		MilliUsedMemory:  usedMemory,
	}, nil
}

// RefreshKopsMetadata updates the kops metadata of a cluster with the current
// values of the running cluster.
func (provisioner *KopsProvisioner) RefreshKopsMetadata(cluster *model.Cluster) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	logger.Info("Refreshing kops metadata")

	kopsClient, err := provisioner.getCachedKopsClient(cluster.ProvisionerMetadataKops.Name, logger)
	if err != nil {
		return errors.Wrap(err, "failed to get kops client from cache")
	}
	defer provisioner.invalidateCachedKopsClientOnError(err, cluster.ProvisionerMetadataKops.Name, logger)

	k8sClient, err := k8s.NewFromFile(kopsClient.GetKubeConfigPath(), logger)
	if err != nil {
		return errors.Wrap(err, "failed to construct k8s client")
	}

	versionInfo, err := k8sClient.Clientset.Discovery().ServerVersion()
	if err != nil {
		return errors.Wrap(err, "failed to get kubernetes version")
	}

	// The GitVersion string usually looks like v1.14.2 so we trim the "v" off
	// to match the version syntax used in kops.
	cluster.ProvisionerMetadataKops.Version = strings.TrimLeft(versionInfo.GitVersion, "v")

	err = kopsClient.UpdateMetadata(cluster.ProvisionerMetadataKops)
	if err != nil {
		return errors.Wrap(err, "failed to update metadata from kops state")
	}

	return nil
}
