package provisioner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/k8s"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/internal/tools/terraform"
	"github.com/mattermost/mattermost-cloud/model"
)

// DefaultKubernetesVersion is the default value for a kubernetes cluster
// version value.
const DefaultKubernetesVersion = "0.0.0"

// KopsProvisioner provisions clusters using kops+terraform.
type KopsProvisioner struct {
	s3StateStore            string
	privateSubnetIds        string
	publicSubnetIds         string
	privateDNS              string
	useExistingAWSResources bool
	logger                  log.FieldLogger
}

// NewKopsProvisioner creates a new KopsProvisioner.
func NewKopsProvisioner(s3StateStore, privateDNS string, useExistingAWSResources bool, logger log.FieldLogger) *KopsProvisioner {
	return &KopsProvisioner{
		s3StateStore:            s3StateStore,
		privateDNS:              privateDNS,
		useExistingAWSResources: useExistingAWSResources,
		logger:                  logger,
	}
}

// PrepareCluster ensures a cluster object is ready for provisioning.
func (provisioner *KopsProvisioner) PrepareCluster(cluster *model.Cluster) (bool, error) {
	kopsMetadata, err := model.NewKopsMetadata(cluster.ProvisionerMetadata)
	if err != nil {
		return false, errors.Wrap(err, "failed to parse existing provisioner metadata")
	}

	// Don't regenerate the name if already set.
	if kopsMetadata.Name != "" {
		return false, nil
	}

	// Generate the kops name using the cluster id.
	kopsMetadata.Name = fmt.Sprintf("%s-kops.k8s.local", cluster.ID)
	cluster.SetProvisionerMetadata(kopsMetadata)

	return true, nil
}

// CreateCluster creates a cluster using kops and terraform.
func (provisioner *KopsProvisioner) CreateCluster(cluster *model.Cluster, awsClient aws.AWS) error {
	kopsMetadata, err := model.NewKopsMetadata(cluster.ProvisionerMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to parse provisioner metadata")
	}

	logger := provisioner.logger.WithField("cluster", cluster.ID)

	awsMetadata, err := model.NewAWSMetadata(cluster.ProviderMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to parse provider metadata")
	}

	isAMIValid, err := awsClient.IsValidAMI(kopsMetadata.AMI)
	if err != nil {
		return errors.Wrapf(err, "Error checking the AWS AMI Image %s", kopsMetadata.AMI)
	}
	if !isAMIValid {
		return errors.Wrapf(err, "invalid AWS AMI Image %s", kopsMetadata.AMI)
	}

	clusterSize, err := kops.GetSize(cluster.Size)
	if err != nil {
		return err
	}

	logger.WithField("name", kopsMetadata.Name).Info("Creating cluster")
	kops, err := kops.New(provisioner.s3StateStore, logger)
	if err != nil {
		return err
	}
	defer kops.Close()

	var clusterResources aws.ClusterResources
	if provisioner.useExistingAWSResources {
		clusterResources, err = awsClient.GetAndClaimVpcResources(cluster.ID, logger)
		if err != nil {
			return err
		}
	}

	err = kops.CreateCluster(
		kopsMetadata.Name,
		kopsMetadata.Version,
		kopsMetadata.AMI,
		cluster.Provider,
		clusterSize,
		awsMetadata.Zones,
		clusterResources.PrivateSubnetIDs,
		clusterResources.PublicSubnetsIDs,
		clusterResources.MasterSecurityGroupIDs,
		clusterResources.WorkerSecurityGroupIDs,
	)
	if err != nil {
		releaseErr := awsClient.ReleaseVpc(cluster.ID, logger)
		if releaseErr != nil {
			logger.WithError(releaseErr).Error("Unable to release VPC")
		}

		return errors.Wrap(err, "unable to create kops cluster")
	}

	terraformClient, err := terraform.New(kops.GetOutputDirectory(), provisioner.s3StateStore, logger)
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

	ugh, err := newUtilityGroupHandle(kops, provisioner, cluster, awsClient, logger)
	if err != nil {
		return err
	}

	return ugh.CreateUtilityGroup()
}

// ProvisionCluster installs all the baseline kubernetes resources needed for
// managing installations. This can be called on an already-provisioned cluster
// to reprovision with the newest version of the resources.
func (provisioner *KopsProvisioner) ProvisionCluster(cluster *model.Cluster, awsClient aws.AWS) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	kops, err := kops.New(provisioner.s3StateStore, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()

	kopsMetadata, err := model.NewKopsMetadata(cluster.ProvisionerMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to parse provisioner metadata")
	}

	err = kops.ExportKubecfg(kopsMetadata.Name)
	if err != nil {
		return errors.Wrap(err, "failed to export kubecfg")
	}

	logger.Info("Provisioning cluster")

	// Begin deploying the mattermost operator.
	k8sClient, err := k8s.New(kops.GetKubeConfigPath(), logger)
	if err != nil {
		return err
	}

	mysqlOperatorNamespace := "mysql-operator"
	minioOperatorNamespace := "minio-operator"
	mattermostOperatorNamespace := "mattermost-operator"
	namespaces := []string{
		mysqlOperatorNamespace,
		minioOperatorNamespace,
		mattermostOperatorNamespace,
	}

	// Remove all previously-installed operator namespaces and resources.
	for _, namespace := range namespaces {
		logger.Infof("Cleaning up namespace %s", namespace)
		err = k8sClient.Clientset.CoreV1().Namespaces().Delete(namespace, &metav1.DeleteOptions{})
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

	_, err = k8sClient.CreateNamespacesIfDoesNotExist(namespaces)
	if err != nil {
		return err
	}

	// TODO: determine if we want to hard-code the k8s resource objects in code.
	// For now, we will ingest manifest files to deploy the mattermost operator.
	files := []k8s.ManifestFile{
		{
			Path:            "operator-manifests/mysql/mysql-operator.yaml",
			DeployNamespace: mysqlOperatorNamespace,
		}, {
			Path:            "operator-manifests/minio/minio-operator.yaml",
			DeployNamespace: minioOperatorNamespace,
		}, {
			Path:            "operator-manifests/mattermost/crds/mm_clusterinstallation_crd.yaml",
			DeployNamespace: mattermostOperatorNamespace,
		}, {
			Path:            "operator-manifests/mattermost/crds/mm_mattermostrestoredb_crd.yaml",
			DeployNamespace: mattermostOperatorNamespace,
		}, {
			Path:            "operator-manifests/mattermost/service_account.yaml",
			DeployNamespace: mattermostOperatorNamespace,
		}, {
			Path:            "operator-manifests/mattermost/role.yaml",
			DeployNamespace: mattermostOperatorNamespace,
		}, {
			Path:            "operator-manifests/mattermost/role_binding.yaml",
			DeployNamespace: mattermostOperatorNamespace,
		}, {
			Path:            "operator-manifests/mattermost/operator.yaml",
			DeployNamespace: mattermostOperatorNamespace,
		},
	}
	err = k8sClient.CreateFromFiles(files)
	if err != nil {
		return err
	}

	// change the waiting time because creation can take more time
	// due container download / init / container creation / volume allocation
	wait = 240
	operatorsWithDeployment := []string{"minio-operator", "mattermost-operator"}
	for _, operator := range operatorsWithDeployment {
		pods, err := k8sClient.GetPodsFromDeployment(operator, operator)
		if err != nil {
			return err
		}
		if len(pods.Items) == 0 {
			return fmt.Errorf("no pods found from %q deployment", operator)
		}

		for _, pod := range pods.Items {
			logger.Infof("Waiting up to %d seconds for %q pod %q to start...", wait, operator, pod.GetName())
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(wait)*time.Second)
			defer cancel()
			pod, err := k8sClient.WaitForPodRunning(ctx, operator, pod.GetName())
			if err != nil {
				return err
			}
			logger.Infof("Successfully deployed operator pod %q", pod.Name)
		}
	}

	operatorsWithStatefulSet := []string{"mysql-operator"}
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
			logger.Infof("Successfully deployed operator pod %q", pod.Name)
		}
	}

	ugh, err := newUtilityGroupHandle(kops, provisioner, cluster, awsClient, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create new cluster utility group handle")
	}

	err = ugh.UpgradeUtilityGroup()
	if err != nil {
		return errors.Wrap(err, "failed to upgrade all services in utility group")
	}

	logger.WithField("name", kopsMetadata.Name).Info("Successfully provisioned cluster")

	return nil
}

// UpgradeCluster upgrades a cluster to the latest recommended production ready k8s version.
func (provisioner *KopsProvisioner) UpgradeCluster(cluster *model.Cluster) error {
	kopsMetadata, err := model.NewKopsMetadata(cluster.ProvisionerMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to parse provisioner metadata")
	}

	logger := provisioner.logger.WithField("cluster", cluster.ID)

	kops, err := kops.New(provisioner.s3StateStore, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()

	err = kops.UpdateCluster(kopsMetadata.Name, kops.GetOutputDirectory())
	if err != nil {
		return err
	}

	terraformClient, err := terraform.New(kops.GetOutputDirectory(), provisioner.s3StateStore, logger)
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

	switch kopsMetadata.Version {
	case "latest", "":
		err = kops.UpgradeCluster(kopsMetadata.Name)
		if err != nil {
			return err
		}
	default:
		setValue := fmt.Sprintf("spec.kubernetesVersion=%s", kopsMetadata.Version)
		err = kops.SetCluster(kopsMetadata.Name, setValue)
		if err != nil {
			return err
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

	logger.Info("Successfully upgraded cluster")

	return nil
}

// DeleteCluster deletes a previously created cluster using kops and terraform.
func (provisioner *KopsProvisioner) DeleteCluster(cluster *model.Cluster, awsClient aws.AWS) error {
	kopsMetadata, err := model.NewKopsMetadata(cluster.ProvisionerMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to parse provisioner metadata")
	}

	logger := provisioner.logger.WithField("cluster", cluster.ID)

	kops, err := kops.New(provisioner.s3StateStore, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()

	err = kops.UpdateCluster(kopsMetadata.Name, kops.GetOutputDirectory())
	if err != nil {
		return err
	}

	terraformClient, err := terraform.New(kops.GetOutputDirectory(), provisioner.s3StateStore, logger)
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
		// There are a few reasons why this may fail including an failed cluster
		// creation. For now, let's log an error if there is one and proceed.
		// TODO: make this better.
		logger.WithError(err).Error("Proceeding with cluster deletion despite failing terraform output match check")
	}

	logger.Info("Deleting cluster")

	err = terraformClient.Destroy()
	if err != nil {
		return err
	}

	err = kops.DeleteCluster(kopsMetadata.Name)
	if err != nil {
		return errors.Wrap(err, "failed to delete cluster")
	}

	err = awsClient.ReleaseVpc(cluster.ID, logger)
	if err != nil {
		return errors.Wrap(err, "unable to release VPC")
	}

	ugh, err := newUtilityGroupHandle(kops, provisioner, cluster, awsClient, logger)
	if err != nil {
		return errors.Wrap(err, "couldn't greate new utility group handle while deleting the cluster")
	}

	err = ugh.DestroyUtilityGroup()
	if err != nil {
		return errors.Wrap(err, "failed to destroy all services in the utility group")
	}

	logger.Info("Successfully deleted cluster")

	return nil
}

// GetClusterResources returns a snapshot of resources of a given cluster.
func (provisioner *KopsProvisioner) GetClusterResources(cluster *model.Cluster, onlySchedulable bool) (*k8s.ClusterResources, error) {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	kops, err := kops.New(provisioner.s3StateStore, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()

	kopsMetadata, err := model.NewKopsMetadata(cluster.ProvisionerMetadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse provisioner metadata")
	}

	err = kops.ExportKubecfg(kopsMetadata.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to export kubecfg")
	}

	k8sClient, err := k8s.New(kops.GetKubeConfigPath(), logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct k8s client")
	}

	allPods, err := k8sClient.Clientset.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	usedCPU, usedMemory := k8s.CalculateTotalPodMilliResourceRequests(allPods)

	var totalCPU, totalMemory int64
	nodes, err := k8sClient.Clientset.CoreV1().Nodes().List(metav1.ListOptions{})
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
				if taint.Effect == "NoSchedule" {
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

// GetClusterVersion returns the version of kubernetes running on the cluster.
func (provisioner *KopsProvisioner) GetClusterVersion(cluster *model.Cluster) (string, error) {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	logger.Info("Getting cluster kubernetes version")

	kops, err := kops.New(provisioner.s3StateStore, logger)
	if err != nil {
		return DefaultKubernetesVersion, errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()

	kopsMetadata, err := model.NewKopsMetadata(cluster.ProvisionerMetadata)
	if err != nil {
		return DefaultKubernetesVersion, errors.Wrap(err, "failed to parse provisioner metadata")
	}

	err = kops.ExportKubecfg(kopsMetadata.Name)
	if err != nil {
		return DefaultKubernetesVersion, errors.Wrap(err, "failed to export kubecfg")
	}

	k8sClient, err := k8s.New(kops.GetKubeConfigPath(), logger)
	if err != nil {
		return DefaultKubernetesVersion, errors.Wrap(err, "failed to construct k8s client")
	}

	versionInfo, err := k8sClient.Clientset.Discovery().ServerVersion()
	if err != nil {
		return DefaultKubernetesVersion, errors.Wrap(err, "failed to get kubernetes version")
	}

	// The GitVersion string usually looks like v1.14.2 so we trim the "v" off
	// to match the version syntax used in kops.
	version := strings.TrimLeft(versionInfo.GitVersion, "v")

	return version, nil
}
