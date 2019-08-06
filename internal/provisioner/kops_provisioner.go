package provisioner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"time"

	mmv1alpha1 "github.com/mattermost/mattermost-operator/pkg/apis/mattermost/v1alpha1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/kubectl/scheme"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/k8s"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/internal/tools/terraform"
	"github.com/mattermost/mattermost-cloud/internal/tools/utils"
	"github.com/mattermost/mattermost-cloud/model"
)

// KopsProvisioner provisions clusters using kops+terraform.
type KopsProvisioner struct {
	clusterRootDir    string
	s3StateStore      string
	certificateSslARN string
	privateSubnetIds  string
	publicSubnetIds   string
	privateDNS        string
	logger            log.FieldLogger
}

// helmDeployment deploys Helm charts.
type helmDeployment struct {
	valuesPath          string
	chartName           string
	namespace           string
	chartDeploymentName string
	setArgument         string
}

// Array of helm apps that need DNS registration.
var helmApps = []string{"prometheus"}

// NewKopsProvisioner creates a new KopsProvisioner.
func NewKopsProvisioner(clusterRootDir, s3StateStore, certificateSslARN, privateSubnetIds, publicSubnetIds, privateDNS string, logger log.FieldLogger) *KopsProvisioner {
	return &KopsProvisioner{
		clusterRootDir:    clusterRootDir,
		s3StateStore:      s3StateStore,
		certificateSslARN: certificateSslARN,
		privateSubnetIds:  privateSubnetIds,
		publicSubnetIds:   publicSubnetIds,
		privateDNS:        privateDNS,
		logger:            logger,
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
func (provisioner *KopsProvisioner) CreateCluster(cluster *model.Cluster, aws aws.AWS) error {
	kopsMetadata, err := model.NewKopsMetadata(cluster.ProvisionerMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to parse provisioner metadata")
	}

	logger := provisioner.logger.WithField("cluster", cluster.ID)

	awsMetadata, err := model.NewAWSMetadata(cluster.ProviderMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to parse provider metadata")
	}

	clusterSize, err := kops.GetSize(cluster.Size)
	if err != nil {
		return err
	}

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

	logger.WithField("name", kopsMetadata.Name).Info("Creating cluster")
	kops, err := kops.New(provisioner.s3StateStore, logger)
	if err != nil {
		return err
	}
	defer kops.Close()
	err = kops.CreateCluster(kopsMetadata.Name, cluster.Provider, clusterSize, awsMetadata.Zones, provisioner.privateSubnetIds, provisioner.publicSubnetIds)
	if err != nil {
		return err
	}

	err = os.Rename(kops.GetOutputDirectory(), outputDir)
	if err != nil && err.(*os.LinkError).Err == syscall.EXDEV {
		err = utils.CopyDirectory(kops.GetOutputDirectory(), outputDir)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to rename kops output directory to '%s' using utils.CopyFolder", outputDir))
		}
	} else if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to rename kops output directory to '%s'", outputDir))
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
	logger.Infof("Waiting up to %d seconds for k8s cluster to become ready...", wait)
	err = kops.WaitForKubernetesReadiness(kopsMetadata.Name, wait)
	if err != nil {
		// Run non-silent validate one more time to log final cluster state
		// and return original timeout error.
		kops.ValidateCluster(kopsMetadata.Name, false)
		return err
	}

	// Set the ELB tags for the public subnets
	if provisioner.publicSubnetIds != "" {
		subnets := strings.Split(provisioner.publicSubnetIds, ",")
		for _, subnet := range subnets {
			logger.WithField("name", kopsMetadata.Name).Infof("Tagging subnet %s", subnet)
			err = aws.TagResource(subnet, fmt.Sprintf("kubernetes.io/cluster/%s", kopsMetadata.Name), "shared", logger)
			if err != nil {
				return errors.Wrap(err, "failed to tag subnet")
			}
		}
	}

	logger.WithField("name", kopsMetadata.Name).Info("Successfully deployed kubernetes")

	logger.Info("Installing Helm")
	err = helmSetup(logger, kops)
	if err != nil {
		return err
	}

	wait = 120
	logger.Infof("Waiting up to %d seconds for helm to become ready...", wait)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(wait)*time.Second)
	defer cancel()
	err = waitForHelmRunning(ctx, kops.GetKubeConfigPath())
	if err != nil {
		return err
	}

	prometheusDNS := fmt.Sprintf("%s.prometheus.%s", cluster.ID, provisioner.privateDNS)
	elasticsearchDNS := fmt.Sprintf("elasticsearch.%s", provisioner.privateDNS)

	helmDeployments := []helmDeployment{
		{
			valuesPath:          "helm-charts/private-nginx_values.yaml",
			chartName:           "stable/nginx-ingress",
			namespace:           "internal-nginx",
			chartDeploymentName: "private-nginx",
		}, {
			valuesPath:          "helm-charts/prometheus_values.yaml",
			chartName:           "stable/prometheus",
			namespace:           "prometheus",
			chartDeploymentName: "prometheus",
			setArgument:         fmt.Sprintf("server.ingress.hosts={%s}", prometheusDNS),
		}, {
			valuesPath:          "helm-charts/fluentd_values.yaml",
			chartName:           "stable/fluentd-elasticsearch",
			namespace:           "fluentd",
			chartDeploymentName: "fluentd",
			setArgument:         fmt.Sprintf("elasticsearch.host=%s", elasticsearchDNS),
		},
	}

	for _, deployment := range helmDeployments {
		logger.Infof("Installing helm chart %s", deployment.chartName)
		err = installHelmChart(deployment, kops.GetKubeConfigPath())
		if err != nil {
			return err
		}
	}

	logger.Infof("Waiting up to %d seconds for internal ELB to become ready...", wait)
	ctx, cancel = context.WithTimeout(context.Background(), time.Duration(wait)*time.Second)
	defer cancel()
	endpoint, err := getLoadBalancerEndpoint(ctx, "internal-nginx", logger, kops.GetKubeConfigPath())
	if err != nil {
		return err
	}

	for _, app := range helmApps {
		dns := fmt.Sprintf("%s.%s.%s", cluster.ID, app, provisioner.privateDNS)
		logger.Infof("Registering DNS %s for %s", dns, app)
		err = aws.CreateCNAME(dns, []string{endpoint}, logger)
		if err != nil {
			return err
		}
	}

	return nil
}

// ProvisionCluster installs all the baseline kubernetes resources needed for
// managing installations. This can be called on an already-provisioned cluster
// to reprovision with the newest version of the resources.
// TODO: Move helm configuration here as well.
func (provisioner *KopsProvisioner) ProvisionCluster(cluster *model.Cluster) error {
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

	operators := []string{"mysql-operator", "minio-operator", "mattermost-operator"}
	for _, operator := range operators {
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

	// Temporarily look for the kops output directory as a local folder named after
	// the cluster ID. See above.
	outputDir := path.Join(provisioner.clusterRootDir, cluster.ID)

	// Validate the provided cluster ID before we alter state in any way.
	_, err = os.Stat(outputDir)
	if err != nil {
		return errors.Wrapf(err, "failed to find cluster directory %q", outputDir)
	}

	terraformClient := terraform.New(outputDir, logger)
	defer terraformClient.Close()

	err = terraformClient.Init()
	if err != nil {
		return err
	}
	out, ok, err := terraformClient.Output("cluster_name")
	if err != nil {
		return err
	} else if !ok {
		logger.Info("No cluster_name in terraform config, assuming partially initialized")
	} else if out != kopsMetadata.Name {
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

	logger.Info("Upgrading cluster")

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
func (provisioner *KopsProvisioner) DeleteCluster(cluster *model.Cluster, aws aws.AWS) error {
	kopsMetadata, err := model.NewKopsMetadata(cluster.ProvisionerMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to parse provisioner metadata")
	}

	logger := provisioner.logger.WithField("cluster", cluster.ID)

	// Temporarily look for the kops output directory as a local folder named after
	// the cluster ID. See above.
	outputDir := path.Join(provisioner.clusterRootDir, cluster.ID)

	// Validate the provided cluster ID before we alter state in any way.
	_, err = os.Stat(outputDir)
	if os.IsNotExist(err) {
		logger.Info("No resources found, assuming cluster was never created")
		return nil
	} else if err != nil {
		return errors.Wrapf(err, "failed to find cluster directory %q", outputDir)
	}

	terraformClient := terraform.New(outputDir, logger)
	defer terraformClient.Close()

	err = terraformClient.Init()
	if err != nil {
		return err
	}

	if out, ok, err := terraformClient.Output("cluster_name"); err != nil {
		return err
	} else if !ok {
		logger.Info("No cluster_name in terraform config, skipping check")
	} else if out != kopsMetadata.Name {
		return fmt.Errorf("terraform cluster_name (%s) does not match kops_name from provided ID (%s)", out, kopsMetadata.Name)
	}

	kops, err := kops.New(provisioner.s3StateStore, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()

	if kopsMetadata.Name != "" {
		_, err = kops.GetCluster(kopsMetadata.Name)
		if err != nil {
			return err
		}
	}

	logger.Info("Deleting cluster")

	err = terraformClient.Destroy()
	if err != nil {
		return err
	}

	if kopsMetadata.Name != "" {
		err = kops.DeleteCluster(kopsMetadata.Name)
		if err != nil {
			return errors.Wrap(err, "failed to delete cluster")
		}
	}

	// Delete the ELB tags for the public subnets
	if kopsMetadata.Name != "" && provisioner.publicSubnetIds != "" {
		subnets := strings.Split(provisioner.publicSubnetIds, ",")
		for _, subnet := range subnets {
			logger.WithField("name", kopsMetadata.Name).Infof("Untagging subnet %s", subnet)
			err = aws.UntagResource(subnet, fmt.Sprintf("kubernetes.io/cluster/%s", kopsMetadata.Name), "shared", logger)
			if err != nil {
				return errors.Wrap(err, "failed to untag subnet")
			}
		}
	}

	err = os.RemoveAll(outputDir)
	if err != nil {
		return errors.Wrap(err, "failed to clean up output directory")
	}

	for _, app := range helmApps {
		logger.Infof("Deleting Route53 DNS Record for %s", app)
		dns := fmt.Sprintf("%s.%s.%s", cluster.ID, app, provisioner.privateDNS)
		err = aws.DeleteCNAME(dns, logger)
		if err != nil {
			return errors.Wrap(err, "failed to delete Route53 DNS record")
		}
	}

	logger.Info("Successfully deleted cluster")

	return nil
}

func makeClusterInstallationName(clusterInstallation *model.ClusterInstallation) string {
	// TODO: Once https://mattermost.atlassian.net/browse/MM-15467 is fixed, we can use the
	// full namespace as part of the name. For now, truncate to keep within the existing limit
	// of 60 characters.
	return fmt.Sprintf("mm-%s", clusterInstallation.Namespace[0:4])
}

// waitForNamespacesDeleted is used to check when all of the provided namespaces
// have been fully terminated.
func waitForNamespacesDeleted(ctx context.Context, namespaces []string, k8sClient *k8s.KubeClient) error {
	for {
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "timed out waiting for namespaces to become fully terminated")
		default:
			var shouldWait bool
			for _, namespace := range namespaces {
				_, err := k8sClient.Clientset.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
				if err != nil && k8sErrors.IsNotFound(err) {
					continue
				}

				shouldWait = true
				break
			}

			if !shouldWait {
				return nil
			}

			time.Sleep(5 * time.Second)
		}
	}
}

// waitForHelmRunning is used to check when Helm is ready to install charts.
func waitForHelmRunning(ctx context.Context, configPath string) error {
	for {
		cmd := exec.Command("helm", "ls", "--kubeconfig", configPath)
		var out bytes.Buffer
		cmd.Stderr = &out
		cmd.Run()
		if out.String() == "" {
			return nil
		}
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "timed out waiting for helm to become ready")
		case <-time.After(5 * time.Second):
		}
	}
}

// getLoadBalancerEndpoint is used to get the endpoint of the internal ingress.
func getLoadBalancerEndpoint(ctx context.Context, namespace string, logger log.FieldLogger, configPath string) (string, error) {
	k8sClient, err := k8s.New(configPath, logger)
	if err != nil {
		return "", err
	}
	for {
		services, err := k8sClient.Clientset.CoreV1().Services(namespace).List(metav1.ListOptions{})
		if err != nil {
			return "", err
		}
		for _, service := range services.Items {
			if service.Status.LoadBalancer.Ingress != nil {
				endpoint := service.Status.LoadBalancer.Ingress[0].Hostname
				if endpoint == "" {
					return "", errors.New("loadbalancer endpoint value is empty")
				}

				return endpoint, nil
			}
		}

		select {
		case <-ctx.Done():
			return "", errors.Wrap(ctx.Err(), "timed out waiting for helm to become ready")
		case <-time.After(5 * time.Second):
		}
	}
}

// installHelmChart is used to install Helm charts.
func installHelmChart(chart helmDeployment, configPath string) error {
	if chart.setArgument != "" {
		err := exec.Command("helm", "install", "--kubeconfig", configPath, "--set", chart.setArgument, "-f", chart.valuesPath, chart.chartName, "--namespace", chart.namespace, "--name", chart.chartDeploymentName).Run()
		if err != nil {
			return err
		}
	} else {
		err := exec.Command("helm", "install", "--kubeconfig", configPath, "-f", chart.valuesPath, chart.chartName, "--namespace", chart.namespace, "--name", chart.chartDeploymentName).Run()
		if err != nil {
			return err
		}
	}

	return nil
}

// helmSetup is used for the initial setup of Helm in cluster.
func helmSetup(logger log.FieldLogger, kops *kops.Cmd) error {
	k8sClient, err := k8s.New(kops.GetKubeConfigPath(), logger)
	if err != nil {
		return err
	}

	logger.Info("Initializing Helm in the cluster")
	err = exec.Command("helm", "--kubeconfig", kops.GetKubeConfigPath(), "init", "--upgrade").Run()
	if err != nil {
		return err
	}

	logger.Info("Creating Tiller service account")
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "tiller"},
	}
	_, err = k8sClient.Clientset.CoreV1().ServiceAccounts("kube-system").Create(serviceAccount)
	if err != nil {
		return err
	}
	logger.Info("Successfully created Tiller service account")

	logger.Info("Creating Tiller cluster role bind")
	roleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "tiller-cluster-rule"},
		Subjects: []rbacv1.Subject{
			{Kind: "ServiceAccount", Name: "tiller", Namespace: "kube-system"},
		},
		RoleRef: rbacv1.RoleRef{Kind: "ClusterRole", Name: "cluster-admin"},
	}

	_, err = k8sClient.Clientset.RbacV1().ClusterRoleBindings().Create(roleBinding)
	if err != nil {
		return err
	}
	logger.Info("Successfully created cluster role bind")

	logger.Info("Upgrade Helm")
	err = exec.Command("helm", "--kubeconfig", kops.GetKubeConfigPath(), "init", "--service-account", "tiller", "--upgrade").Run()
	if err != nil {
		return err
	}
	return nil
}

// CreateClusterInstallation creates a Mattermost installation within the given cluster.
func (provisioner *KopsProvisioner) CreateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	logger := provisioner.logger.WithFields(map[string]interface{}{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

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

	k8sClient, err := k8s.New(kops.GetKubeConfigPath(), logger)
	if err != nil {
		return err
	}

	_, err = k8sClient.CreateNamespaceIfDoesNotExist(clusterInstallation.Namespace)
	if err != nil {
		return errors.Wrapf(err, "failed to create namespace %s", clusterInstallation.Namespace)
	}

	var secretName string
	if installation.License != "" {
		secretName = fmt.Sprintf("%s-license", makeClusterInstallationName(clusterInstallation))
		_, err = k8sClient.CreateSecret(clusterInstallation.Namespace, secretName, "license", installation.License)
		if err != nil {
			return errors.Wrapf(err, "failed to create the license secret %s/%s", clusterInstallation.Namespace, secretName)
		}
	}

	_, err = k8sClient.MattermostClientset.MattermostV1alpha1().ClusterInstallations(clusterInstallation.Namespace).Create(&mmv1alpha1.ClusterInstallation{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterInstallation",
			APIVersion: "mattermost.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      makeClusterInstallationName(clusterInstallation),
			Namespace: clusterInstallation.Namespace,
			Labels: map[string]string{
				"installation":         installation.ID,
				"cluster-installation": clusterInstallation.ID,
			},
		},
		Spec: mmv1alpha1.ClusterInstallationSpec{
			Size:                   installation.Size,
			Version:                translateMattermostVersion(installation.Version),
			IngressName:            installation.DNS,
			UseServiceLoadBalancer: true,
			ServiceAnnotations: map[string]string{
				"service.beta.kubernetes.io/aws-load-balancer-backend-protocol":        "tcp",
				"service.beta.kubernetes.io/aws-load-balancer-ssl-cert":                provisioner.certificateSslARN,
				"service.beta.kubernetes.io/aws-load-balancer-ssl-ports":               "https",
				"service.beta.kubernetes.io/aws-load-balancer-connection-idle-timeout": "120",
			},
			MattermostLicenseSecret: secretName,
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to create cluster installation")
	}

	logger.Info("Successfully created cluster installation")

	return nil
}

// DeleteClusterInstallation deletes a Mattermost installation within the given cluster.
func (provisioner *KopsProvisioner) DeleteClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	logger := provisioner.logger.WithFields(map[string]interface{}{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	kops, err := kops.New(provisioner.s3StateStore, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()

	kopsMetadata, err := model.NewKopsMetadata(cluster.ProvisionerMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to parse provisioner metadata")
	}

	if kopsMetadata.Name == "" {
		logger.Infof("Cluster %s has no name, assuming cluster installation never existed.", cluster.ID)
		return nil
	}

	err = kops.ExportKubecfg(kopsMetadata.Name)
	if err != nil {
		return errors.Wrap(err, "failed to export kubecfg")
	}

	k8sClient, err := k8s.New(kops.GetKubeConfigPath(), logger)
	if err != nil {
		return err
	}

	name := makeClusterInstallationName(clusterInstallation)

	err = k8sClient.MattermostClientset.MattermostV1alpha1().ClusterInstallations(clusterInstallation.Namespace).Delete(name, nil)
	if k8sErrors.IsNotFound(err) {
		logger.Infof("Cluster installation %s not found, assuming already deleted.", name)
	} else if err != nil {
		return errors.Wrapf(err, "failed to delete cluster installation %s", clusterInstallation.ID)
	}

	if installation.License != "" {
		secretName := fmt.Sprintf("%s-license", makeClusterInstallationName(clusterInstallation))
		err = k8sClient.Clientset.CoreV1().Secrets(clusterInstallation.Namespace).Delete(secretName, nil)
		if k8sErrors.IsNotFound(err) {
			logger.Infof("Secret %s/%s not found. Maybe the license was not set for this installation or was already deleted", clusterInstallation.Namespace, secretName)
		} else if err != nil {
			return errors.Wrapf(err, "failed to delete secret %s/%s", clusterInstallation.Namespace, secretName)
		}
	}

	err = k8sClient.Clientset.CoreV1().Namespaces().Delete(clusterInstallation.Namespace, &metav1.DeleteOptions{})
	if k8sErrors.IsNotFound(err) {
		logger.Infof("Namespace %s not found, assuming already deleted.", clusterInstallation.Namespace)
	} else if err != nil {
		return errors.Wrapf(err, "failed to delete namespace %s", clusterInstallation.Namespace)
	}

	logger.Info("Successfully deleted cluster installation")

	return nil
}

// UpdateClusterInstallation updates the cluster installation spec to match the
// installation specification.
func (provisioner *KopsProvisioner) UpdateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	logger := provisioner.logger.WithFields(map[string]interface{}{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	kops, err := kops.New(provisioner.s3StateStore, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()

	kopsMetadata, err := model.NewKopsMetadata(cluster.ProvisionerMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to parse provisioner metadata")
	}

	if kopsMetadata.Name == "" {
		logger.Infof("Cluster %s has no name, assuming cluster installation never existed.", cluster.ID)
		return nil
	}

	err = kops.ExportKubecfg(kopsMetadata.Name)
	if err != nil {
		return errors.Wrap(err, "failed to export kubecfg")
	}

	k8sClient, err := k8s.New(kops.GetKubeConfigPath(), logger)
	if err != nil {
		return err
	}

	name := makeClusterInstallationName(clusterInstallation)

	cr, err := k8sClient.MattermostClientset.MattermostV1alpha1().ClusterInstallations(clusterInstallation.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to get cluster installation %s", clusterInstallation.ID)
	}

	logger.WithField("status", fmt.Sprintf("%+v", cr.Status)).Debug("Got cluster installation")

	version := translateMattermostVersion(installation.Version)
	if cr.Spec.Version == version {
		logger.Infof("Cluster installation already on version %s", version)
	}
	cr.Spec.Version = version

	cr.Spec.MattermostLicenseSecret = ""
	secretName := fmt.Sprintf("%s-license", name)
	if installation.License != "" {
		secretSpec := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: clusterInstallation.Namespace,
			},
			StringData: map[string]string{
				"license": installation.License,
			},
		}
		_, err = k8sClient.CreateOrUpdateSecret(clusterInstallation.Namespace, secretSpec)
		if err != nil {
			return errors.Wrapf(err, "failed to create the license secret %s/%s", clusterInstallation.Namespace, secretName)
		}
		cr.Spec.MattermostLicenseSecret = secretName
	} else {
		err = k8sClient.Clientset.CoreV1().Secrets(clusterInstallation.Namespace).Delete(secretName, nil)
		if k8sErrors.IsNotFound(err) {
			logger.Infof("Secret %s/%s not found. Maybe the license was not set for this installation or was already deleted", clusterInstallation.Namespace, secretName)
		}
	}

	_, err = k8sClient.MattermostClientset.MattermostV1alpha1().ClusterInstallations(clusterInstallation.Namespace).Update(cr)
	if err != nil {
		return errors.Wrapf(err, "failed to update cluster installation %s", clusterInstallation.ID)
	}

	logger.Infof("Updated cluster installation version to %s", installation.Version)

	return nil
}

// GetClusterInstallationResource gets the cluster installation resource from
// the kubernetes API.
func (provisioner *KopsProvisioner) GetClusterInstallationResource(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) (*mmv1alpha1.ClusterInstallation, error) {
	logger := provisioner.logger.WithFields(map[string]interface{}{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	kops, err := kops.New(provisioner.s3StateStore, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()

	kopsMetadata, err := model.NewKopsMetadata(cluster.ProvisionerMetadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse provisioner metadata")
	}

	if kopsMetadata.Name == "" {
		logger.Infof("Cluster %s has no name, assuming cluster installation never existed.", cluster.ID)
		return nil, nil
	}

	err = kops.ExportKubecfg(kopsMetadata.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to export kubecfg")
	}

	k8sClient, err := k8s.New(kops.GetKubeConfigPath(), logger)
	if err != nil {
		return nil, err
	}

	name := makeClusterInstallationName(clusterInstallation)

	cr, err := k8sClient.MattermostClientset.MattermostV1alpha1().ClusterInstallations(clusterInstallation.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return cr, errors.Wrapf(err, "failed to get cluster installation %s", clusterInstallation.ID)
	}

	logger.WithField("status", fmt.Sprintf("%+v", cr.Status)).Debug("Got cluster installation")

	return cr, nil
}

// ExecMattermostCLI invokes the Mattermost CLI for the given cluster installation with the given args.
func (provisioner *KopsProvisioner) ExecMattermostCLI(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) ([]byte, error) {
	return provisioner.execCLI(cluster, clusterInstallation, append([]string{"./bin/mattermost"}, args...)...)
}

func (provisioner *KopsProvisioner) execCLI(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) ([]byte, error) {
	logger := provisioner.logger.WithFields(map[string]interface{}{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

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

	podList, err := k8sClient.Clientset.CoreV1().Pods(clusterInstallation.Namespace).List(metav1.ListOptions{
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

	return k8sClient.RemoteCommand("POST", execRequest.URL())
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
