package provisioner

import (
	"fmt"
	"os"
	"path"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/mattermost/mattermost-cloud/internal/model"
	"github.com/mattermost/mattermost-cloud/internal/tools/k8s"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/internal/tools/terraform"
)

// KopsProvisioner provisions clusters using kops+terraform.
type KopsProvisioner struct {
	clusterRootDir string
	s3StateStore   string
	logger         log.FieldLogger
}

// NewKopsProvisioner creates a new KopsProvisioner.
func NewKopsProvisioner(clusterRootDir string, s3StateStore string, logger log.FieldLogger) *KopsProvisioner {
	return &KopsProvisioner{
		clusterRootDir: clusterRootDir,
		s3StateStore:   s3StateStore,
		logger:         logger,
	}
}

// CreateCluster creates a cluster using kops and terraform.
func (provisioner *KopsProvisioner) CreateCluster(cluster *model.Cluster) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	awsMetadata := model.NewAWSMetadata(cluster.ProviderMetadata)

	clusterSize, err := kops.GetSize(cluster.Size)
	if err != nil {
		return err
	}

	// Generate the kops name using the cluster id.
	kopsMetadata := model.NewKopsMetadata(cluster.ProvisionerMetadata)
	kopsMetadata.Name = fmt.Sprintf("%s-kops.k8s.local", cluster.ID)
	cluster.SetProvisionerMetadata(kopsMetadata)

	// Temporarily locate the kops output directory to a local folder based on the
	// cluster name. This won't be necessary once we persist the output to S3 instead.
	_, err = os.Stat(provisioner.clusterRootDir)
	if err != nil && os.IsNotExist(err) {
		err = os.Mkdir(provisioner.clusterRootDir, 0755)
		if err != nil {
			return errors.Wrap(err, "unable to create cluster root dir")
		}
	} else if err != nil {
		return errors.Wrapf(err, "failed to stat cluster root directory %q", provisioner.clusterRootDir)
	}

	outputDir := path.Join(provisioner.clusterRootDir, cluster.ID)
	_, err = os.Stat(outputDir)
	if err == nil {
		return fmt.Errorf("encountered cluster ID collision: directory %q already exists", outputDir)
	} else if err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to stat cluster directory %q", outputDir)
	}

	logger.WithField("name", kopsMetadata.Name).Info("creating cluster")
	kops, err := kops.New(provisioner.s3StateStore, logger)
	if err != nil {
		return err
	}
	defer kops.Close()
	err = kops.CreateCluster(kopsMetadata.Name, cluster.Provider, clusterSize, awsMetadata.Zones)
	if err != nil {
		return err
	}

	err = os.Rename(kops.GetOutputDirectory(), outputDir)
	if err != nil {
		return fmt.Errorf("failed to rename kops output directory to %q", outputDir)
	}

	terraformClient := terraform.New(outputDir, logger)
	defer terraformClient.Close()

	err = terraformClient.Init()
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

	err = kops.UpdateCluster(kopsMetadata.Name)
	if err != nil {
		return err
	}

	err = terraformClient.Apply()
	if err != nil {
		return err
	}

	// TODO: Rework this as we make the API calls asynchronous.
	wait := 600
	logger.Infof("waiting up to %d seconds for k8s cluster to become ready...", wait)
	err = kops.WaitForKubernetesReadiness(kopsMetadata.Name, wait)
	if err != nil {
		// Run non-silent validate one more time to log final cluster state
		// and return original timeout error.
		kops.ValidateCluster(kopsMetadata.Name, false)
		return err
	}

	logger.WithField("name", kopsMetadata.Name).Info("successfully deployed kubernetes")

	// Begin deploying the mattermost operator.
	k8sClient, err := k8s.New(kops.GetKubeConfigPath(), logger)
	if err != nil {
		return err
	}

	mysqlOperatorNamespace := "mysql-operator"
	mattermostOperatorNamespace := "mattermost-operator"
	namespaces := []string{
		mysqlOperatorNamespace,
		mattermostOperatorNamespace,
	}

	_, err = k8sClient.CreateNamespaces(namespaces)
	if err != nil {
		return err
	}

	// TODO: determine if we want to hard-code the k8s resource objects in code.
	// For now, we will ingest manifest files to deploy the mattermost operator.
	files := []k8s.ManifestFile{
		k8s.ManifestFile{
			Path:            "operator-manifests/mysql-operator/crds/mysql_crd.yaml",
			DeployNamespace: mysqlOperatorNamespace,
		}, k8s.ManifestFile{
			Path:            "operator-manifests/mysql-operator/service_account.yaml",
			DeployNamespace: mysqlOperatorNamespace,
		}, k8s.ManifestFile{
			Path:            "operator-manifests/mysql-operator/role.yaml",
			DeployNamespace: mysqlOperatorNamespace,
		}, k8s.ManifestFile{
			Path:            "operator-manifests/mysql-operator/role_binding.yaml",
			DeployNamespace: mysqlOperatorNamespace,
		}, k8s.ManifestFile{
			Path:            "operator-manifests/mysql-operator/operator.yaml",
			DeployNamespace: mysqlOperatorNamespace,
		}, k8s.ManifestFile{
			Path:            "operator-manifests/mattermost-operator/crds/mm_clusterinstallation_crd.yaml",
			DeployNamespace: mattermostOperatorNamespace,
		}, k8s.ManifestFile{
			Path:            "operator-manifests/mattermost-operator/service_account.yaml",
			DeployNamespace: mattermostOperatorNamespace,
		}, k8s.ManifestFile{
			Path:            "operator-manifests/mattermost-operator/role.yaml",
			DeployNamespace: mattermostOperatorNamespace,
		}, k8s.ManifestFile{
			Path:            "operator-manifests/mattermost-operator/role_binding.yaml",
			DeployNamespace: mattermostOperatorNamespace,
		}, k8s.ManifestFile{
			Path:            "operator-manifests/mattermost-operator/operator.yaml",
			DeployNamespace: mattermostOperatorNamespace,
		},
	}
	err = k8sClient.CreateFromFiles(files)
	if err != nil {
		return err
	}

	// TODO: Rework this as we make the API calls asynchronous.
	wait = 60
	logger.Infof("waiting up to %d seconds for mattermost operator to start...", wait)
	pod, err := k8sClient.WaitForPodRunning("mattermost-operator", "mattermost-operator", wait)
	if err != nil {
		return err
	}
	logger.Infof("successfully deployed operator %q", pod.Name)

	logger.WithField("name", kopsMetadata.Name).Info("successfully created cluster")

	return nil
}

// UpgradeCluster upgrades a cluster to the latest recommended production ready k8s version.
func (provisioner *KopsProvisioner) UpgradeCluster(cluster *model.Cluster) error {
	kopsMetadata := model.NewKopsMetadata(cluster.ProvisionerMetadata)

	logger := provisioner.logger.WithField("cluster", cluster.ID)

	// Temporarily look for the kops output directory as a local folder named after
	// the cluster ID. See above.
	outputDir := path.Join(provisioner.clusterRootDir, cluster.ID)

	// Validate the provided cluster ID before we alter state in any way.
	_, err := os.Stat(outputDir)
	if err != nil {
		return errors.Wrapf(err, "failed to find cluster directory %q", outputDir)
	}

	terraformClient := terraform.New(outputDir, logger)
	defer terraformClient.Close()

	err = terraformClient.Init()
	if err != nil {
		return err
	}
	out, err := terraformClient.Output("cluster_name")
	if err != nil {
		return err
	}
	if out != kopsMetadata.Name {
		return fmt.Errorf("terraform cluster_name (%s) does not match kops name from provided ID (%s)", out, kopsMetadata.Name)
	}

	kops, err := kops.New(provisioner.s3StateStore, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()
	_, err = kops.GetCluster(kopsMetadata.Name)
	if err != nil {
		return err
	}

	logger.Info("upgrading cluster")

	err = kops.UpgradeCluster(kopsMetadata.Name)
	if err != nil {
		return err
	}
	err = kops.UpdateCluster(kopsMetadata.Name)
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
	wait := 600
	if wait > 0 {
		logger.Infof("waiting up to %d seconds for k8s cluster to become ready...", wait)
		err = kops.WaitForKubernetesReadiness(kopsMetadata.Name, wait)
		if err != nil {
			// Run non-silent validate one more time to log final cluster state
			// and return original timeout error.
			kops.ValidateCluster(kopsMetadata.Name, false)
			return err
		}
	}

	logger.Info("successfully upgraded cluster")

	return nil
}

// DeleteCluster deletes a previously created cluster using kops and terraform.
func (provisioner *KopsProvisioner) DeleteCluster(cluster *model.Cluster) error {
	kopsMetadata := model.NewKopsMetadata(cluster.ProvisionerMetadata)

	logger := provisioner.logger.WithField("cluster", cluster.ID)

	// Temporarily look for the kops output directory as a local folder named after
	// the cluster ID. See above.
	outputDir := path.Join(provisioner.clusterRootDir, cluster.ID)

	// Validate the provided cluster ID before we alter state in any way.
	_, err := os.Stat(outputDir)
	if err != nil {
		return errors.Wrapf(err, "failed to find cluster directory %q", outputDir)
	}

	terraformClient := terraform.New(outputDir, logger)
	defer terraformClient.Close()

	err = terraformClient.Init()
	if err != nil {
		return err
	}
	out, err := terraformClient.Output("cluster_name")
	if err != nil {
		return err
	}
	if out != kopsMetadata.Name {
		return fmt.Errorf("terraform cluster_name (%s) does not match kops_name from provided ID (%s)", out, kopsMetadata.Name)
	}

	kops, err := kops.New(provisioner.s3StateStore, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()
	_, err = kops.GetCluster(kopsMetadata.Name)
	if err != nil {
		return err
	}

	logger.Info("deleting cluster")
	err = terraformClient.Destroy()
	if err != nil {
		return err
	}

	err = kops.DeleteCluster(kopsMetadata.Name)
	if err != nil {
		return errors.Wrap(err, "failed to delete cluster")
	}

	err = os.RemoveAll(outputDir)
	if err != nil {
		return errors.Wrap(err, "failed to clean up output directory")
	}

	logger.Info("successfully deleted cluster")

	return nil
}
