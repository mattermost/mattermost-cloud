package provisioner

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/mattermost/mattermost-cloud/internal/api"
	"github.com/mattermost/mattermost-cloud/internal/model"
	"github.com/mattermost/mattermost-cloud/internal/tools/k8s"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/internal/tools/terraform"
)

// KopsProvisioner provisions clusters using kops+terraform.
type KopsProvisioner struct {
	clusterRootDir   string
	s3StateStore     string
	cs               clusterStore
	kopsFactory      kopsFactoryFunc
	terraformFactory terraformFactoryFunc
	k8sFactory       k8sFactoryFunc
	logger           log.FieldLogger
}

// NewKopsProvisioner creates a new KopsProvisioner.
//
// kopsFactory and terraformFactory exist purely to allow for unit testing. Passing nil defaults
// to using the kops and terraform binary wrappers.
func NewKopsProvisioner(clusterRootDir string, s3StateStore string, cs clusterStore, kopsFactory kopsFactoryFunc, terraformFactory terraformFactoryFunc, k8sFactory k8sFactoryFunc, logger log.FieldLogger) *KopsProvisioner {
	if kopsFactory == nil {
		kopsFactory = func(logger log.FieldLogger) (KopsCmd, error) {
			return kops.New(s3StateStore, logger)
		}
	}

	if terraformFactory == nil {
		terraformFactory = func(outputDir string, logger log.FieldLogger) TerraformCmd {
			return terraform.New(outputDir, logger)
		}
	}

	if k8sFactory == nil {
		k8sFactory = func(configLocation string, logger log.FieldLogger) (K8sClient, error) {
			return k8s.New(configLocation, logger)
		}
	}

	return &KopsProvisioner{
		clusterRootDir:   clusterRootDir,
		s3StateStore:     s3StateStore,
		cs:               cs,
		kopsFactory:      kopsFactory,
		terraformFactory: terraformFactory,
		k8sFactory:       k8sFactory,
		logger:           logger,
	}
}

// CreateCluster creates a cluster using kops and terraform.
func (provisioner *KopsProvisioner) CreateCluster(request *api.CreateClusterRequest) (*model.Cluster, error) {
	provider, err := checkProvider(request.Provider)
	if err != nil {
		return nil, err
	}

	clusterSize, err := kops.GetSize(request.Size)
	if err != nil {
		return nil, err
	}

	cluster := model.Cluster{
		Provider:    request.Provider,
		Provisioner: "kops",
	}
	err = provisioner.cs.CreateCluster(&cluster)
	if err != nil {
		return nil, err
	}

	// Once the cluster has been recorded, generate the kops name using the cluster id.
	kopsMetadata := model.KopsMetadata{
		Name: fmt.Sprintf("%s-kops.k8s.local", cluster.ID),
	}
	cluster.SetProvisionerMetadata(kopsMetadata)

	err = provisioner.cs.UpdateCluster(&cluster)
	if err != nil {
		return nil, err
	}

	// Temporarily locate the kops output directory to a local folder based on the
	// cluster name. This won't be necessary once we persist the output to S3 instead.
	_, err = os.Stat(provisioner.clusterRootDir)
	if err != nil && os.IsNotExist(err) {
		err = os.Mkdir(provisioner.clusterRootDir, 0755)
		if err != nil {
			return nil, errors.Wrap(err, "unable to create cluster root dir")
		}
	} else if err != nil {
		return nil, errors.Wrapf(err, "failed to stat cluster root directory %q", provisioner.clusterRootDir)
	}

	outputDir := path.Join(provisioner.clusterRootDir, cluster.ID)
	_, err = os.Stat(outputDir)
	if err == nil {
		return nil, fmt.Errorf("encountered cluster ID collision: directory %q already exists", outputDir)
	} else if err != nil && !os.IsNotExist(err) {
		return nil, errors.Wrapf(err, "failed to stat cluster directory %q", outputDir)
	}
	// --------------------

	logger := provisioner.logger.WithField("cluster", cluster.ID)

	logger.WithField("name", kopsMetadata.Name).Info("creating cluster")

	kops, err := provisioner.kopsFactory(logger)
	if err != nil {
		return nil, err
	}
	defer kops.Close()
	err = kops.CreateCluster(kopsMetadata.Name, provider, clusterSize, request.Zones)
	if err != nil {
		return nil, err
	}

	err = os.Rename(kops.GetOutputDirectory(), outputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to rename kops output directory to %q", outputDir)
	}

	terraformClient := provisioner.terraformFactory(outputDir, logger)
	defer terraformClient.Close()
	err = terraformClient.Init()
	if err != nil {
		return nil, err
	}

	err = terraformClient.ApplyTarget(fmt.Sprintf("aws_internet_gateway.%s-kops-k8s-local", cluster.ID))
	if err != nil {
		return nil, err
	}

	err = terraformClient.ApplyTarget(fmt.Sprintf("aws_elb.api-%s-kops-k8s-local", cluster.ID))
	if err != nil {
		return nil, err
	}

	err = kops.UpdateCluster(kopsMetadata.Name)
	if err != nil {
		return nil, err
	}

	err = terraformClient.Apply()
	if err != nil {
		return nil, err
	}

	// TODO: Rework this as we make the API calls asynchronous.
	wait := 600
	logger.Infof("waiting up to %d seconds for k8s cluster to become ready...", wait)
	err = kops.WaitForKubernetesReadiness(kopsMetadata.Name, wait)
	if err != nil {
		// Run non-silent validate one more time to log final cluster state
		// and return original timeout error.
		kops.ValidateCluster(kopsMetadata.Name, false)
		return nil, err
	}

	logger.WithField("name", kopsMetadata.Name).Info("successfully deployed kubernetes")

	// Begin deploying the mattermost operator.
	// TODO: remove reliance on kube config being in the default location.
	k8sClient, err := provisioner.k8sFactory(filepath.Join(os.Getenv("HOME"), ".kube", "config"), logger)
	if err != nil {
		return &cluster, err
	}

	mysqlOperatorNamespace := "mysql-operator"
	mattermostOperatorNamespace := "mattermost-operator"
	namespaces := []string{
		mysqlOperatorNamespace,
		mattermostOperatorNamespace,
	}

	for _, ns := range namespaces {
		_, err = k8sClient.CreateNamespace(ns)
		if err != nil {
			return &cluster, err
		}
	}

	// TODO: determine if we want to hard-code the k8s resource objects in code.
	// For now, we will ingest manifest files to deploy the mattermost operator.
	files := []k8s.ManifestFile{
		k8s.ManifestFile{
			Name:            "mysql_crd.yaml",
			Directory:       "operator-manifests/mysql-operator/crds",
			DeployNamespace: mysqlOperatorNamespace,
		}, k8s.ManifestFile{
			Name:            "service_account.yaml",
			Directory:       "operator-manifests/mysql-operator",
			DeployNamespace: mysqlOperatorNamespace,
		}, k8s.ManifestFile{
			Name:            "role.yaml",
			Directory:       "operator-manifests/mysql-operator",
			DeployNamespace: mysqlOperatorNamespace,
		}, k8s.ManifestFile{
			Name:            "role_binding.yaml",
			Directory:       "operator-manifests/mysql-operator",
			DeployNamespace: mysqlOperatorNamespace,
		}, k8s.ManifestFile{
			Name:            "operator.yaml",
			Directory:       "operator-manifests/mysql-operator",
			DeployNamespace: mysqlOperatorNamespace,
		}, k8s.ManifestFile{
			Name:            "mm_clusterinstallation_crd.yaml",
			Directory:       "operator-manifests/mattermost-operator/crds",
			DeployNamespace: mattermostOperatorNamespace,
		}, k8s.ManifestFile{
			Name:            "service_account.yaml",
			Directory:       "operator-manifests/mattermost-operator",
			DeployNamespace: mattermostOperatorNamespace,
		}, k8s.ManifestFile{
			Name:            "role.yaml",
			Directory:       "operator-manifests/mattermost-operator",
			DeployNamespace: mattermostOperatorNamespace,
		}, k8s.ManifestFile{
			Name:            "role_binding.yaml",
			Directory:       "operator-manifests/mattermost-operator",
			DeployNamespace: mattermostOperatorNamespace,
		}, k8s.ManifestFile{
			Name:            "operator.yaml",
			Directory:       "operator-manifests/mattermost-operator",
			DeployNamespace: mattermostOperatorNamespace,
		},
	}
	for _, f := range files {
		err = k8sClient.CreateFromFile(f)
		if err != nil {
			return &cluster, err
		}
	}

	// TODO: Rework this as we make the API calls asynchronous.
	wait = 60
	logger.Infof("waiting up to %d seconds for mattermost operator to start...", wait)
	pod, err := k8sClient.WaitForPodRunning("mattermost-operator", "mattermost-operator", wait)
	if err != nil {
		return &cluster, err
	}
	logger.Infof("successfully deployed operator %q", pod.Name)

	logger.WithField("name", kopsMetadata.Name).Info("successfully created cluster")

	return &cluster, nil
}

// UpgradeCluster upgrades a cluster to the latest recommended production ready k8s version.
func (provisioner *KopsProvisioner) UpgradeCluster(clusterID string, version string) error {
	// TODO: Support something other than "latest".
	if version != "latest" {
		return errors.Errorf(`unsupported kubernetes version %s, pass "latest"`, version)
	}

	cluster, err := provisioner.cs.GetCluster(clusterID)
	if err != nil {
		return err
	}
	if cluster == nil {
		return errors.Errorf("unknown cluster %s", clusterID)
	}

	kopsMetadata := model.NewKopsMetadata(cluster.ProvisionerMetadata)

	logger := provisioner.logger.WithField("cluster", cluster.ID)

	// Temporarily look for the kops output directory as a local folder named after
	// the cluster ID. See above.
	outputDir := path.Join(provisioner.clusterRootDir, cluster.ID)

	// Validate the provided cluster ID before we alter state in any way.
	_, err = os.Stat(outputDir)
	if err != nil {
		return errors.Wrapf(err, "failed to find cluster directory %q", outputDir)
	}

	terraformClient := provisioner.terraformFactory(outputDir, logger)
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

	kops, err := provisioner.kopsFactory(logger)
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
func (provisioner *KopsProvisioner) DeleteCluster(clusterID string) error {
	cluster, err := provisioner.cs.GetCluster(clusterID)
	if err != nil {
		return err
	}
	if cluster == nil {
		return errors.Errorf("unknown cluster %s", clusterID)
	}

	kopsMetadata := model.NewKopsMetadata(cluster.ProvisionerMetadata)

	logger := provisioner.logger.WithField("cluster", cluster.ID)

	// Temporarily look for the kops output directory as a local folder named after
	// the cluster ID. See above.
	outputDir := path.Join(provisioner.clusterRootDir, cluster.ID)

	// Validate the provided cluster ID before we alter state in any way.
	_, err = os.Stat(outputDir)
	if err != nil {
		return errors.Wrapf(err, "failed to find cluster directory %q", outputDir)
	}

	terraformClient := provisioner.terraformFactory(outputDir, logger)
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

	kops, err := provisioner.kopsFactory(logger)
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

	err = provisioner.cs.DeleteCluster(cluster.ID)
	if err != nil {
		return err
	}

	logger.Info("successfully deleted cluster")

	return nil
}
