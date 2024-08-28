// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"fmt"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/provisioner/pgbouncer"
	"github.com/mattermost/mattermost-cloud/internal/provisioner/prometheus"
	"github.com/mattermost/mattermost-cloud/internal/provisioner/utility"
	"github.com/mattermost/mattermost-cloud/internal/tools/argocd"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/git"
	"github.com/mattermost/mattermost-cloud/internal/tools/helm"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func provisionCluster(
	cluster *model.Cluster,
	kubeconfigPath string,
	tempDir string,
	awsClient aws.AWS,
	gitClient git.Client,
	argocdClient argocd.Client,
	params ProvisioningParams,
	store model.ClusterUtilityDatabaseStoreInterface,
	logger logrus.FieldLogger) error {

	if err := attachPolicyRoles(cluster, awsClient, logger); err != nil {
		return errors.Wrap(err, "failed to attach policy roles to cluster")
	}

	// Register cluster in argocd
	if cluster.UtilityMetadata.ManagedByArgocd && !cluster.UtilityMetadata.ArgocdClusterRegister.Registered {
		logger.Debug("Starting argocd cluster registration")
		clusterRegister, err := NewClusterRegisterHandle(cluster, gitClient, awsClient.GetCloudEnvironmentName(), tempDir, logger)
		if err != nil {
			return errors.Wrap(err, "failed to create new cluster register handle")
		}
		if err = clusterRegister.clusterRegister(params.S3StateStore); err != nil {
			return errors.Wrap(err, "failed to register cluster in argocd")
		}

		cluster.UtilityMetadata.ArgocdClusterRegister.Registered = true
		logger.Infof("Cluster: %s successfully registered in argocd", cluster.ID)

	} else {
		logger.WithFields(logrus.Fields{
			"ManagedByArgocd": cluster.UtilityMetadata.ManagedByArgocd,
			"Registered":      cluster.UtilityMetadata.ArgocdClusterRegister.Registered,
		}).Info("Skipping argocd cluster registration")
	}

	// Start by gathering resources that will be needed later. If any of this
	// fails then no cluster changes have been made which reduces risk.
	deployPerseus := true
	perseusSecret, err := awsClient.GeneratePerseusUtilitySecret(cluster.ID, logger)
	if err != nil {
		// NOTE: for now, there is no guarantee that all of the cluster VPCs will have
		// the necessary resources created for Perseus. If the necessary resources are
		// not available then warnings will be logged and Perseus won't be deployed.
		// TODO: revisit this after perseus testing is complete.
		logger.Debug(errors.Wrap(err, "Failed to generate perseus secret; skipping perseus utility deployment").Error())
		deployPerseus = false
	}

	bifrostSecret, err := awsClient.GenerateBifrostUtilitySecret(cluster.ID, logger)
	if err != nil {
		return errors.Wrap(err, "failed to generate bifrost secret")
	}

	// Mattermost Operator Helm Migration
	// There are three possible states that a cluster can be in when
	// provisioning happens:
	// 1. New cluster with Mattermost Operator not installed
	// 2. Existing cluster with "old" resources manually installed
	// 3. Existing cluster with "new" helm chart installed
	// The migration process checks if the cluster is in state `2` and prepares
	// it to be migrated. Once all clusters have been migrated then the
	// migration check can be removed.
	err = migrateMattermostOperatorToHelm(kubeconfigPath, logger)
	if err != nil {
		return errors.Wrap(err, "failed to migrate operator to helm")
	}

	// Begin deploying the mattermost operator.
	k8sClient, err := k8s.NewFromFile(kubeconfigPath, logger)
	if err != nil {
		return errors.Wrap(err, "failed to initialize K8s client from kubeconfig")
	}
	helmClient, err := helm.New(kubeconfigPath, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create a new helm client")
	}
	found, _ := helmClient.HelmChartFoundAndDeployed("mattermost-operator", "mattermost-operator")
	if !found {
		logger.Info("Performing first-time Mattermost Operator helm chart installation")

		err = helmClient.RepoAdd("mattermost", "https://helm.mattermost.com")
		if err != nil {
			return errors.Wrap(err, "failed to add mattermost helm repo")
		}
		err = helmClient.RunGenericCommand(
			"install", "mattermost-operator", "mattermost/mattermost-operator",
			"-n", "mattermost-operator",
			"-f", "helm-charts/mattermost-operator.yaml",
			"--create-namespace",
		)
		if err != nil {
			return errors.Wrap(err, "failed to install mattermost operator chart")
		}
	}

	mattermostOperatorNamespace := "mattermost-operator"
	mysqlOperatorNamespace := "mysql-operator"
	minioOperatorNamespace := "minio-operator"

	namespaces := []string{}
	if params.DeployMysqlOperator {
		namespaces = append(namespaces, mysqlOperatorNamespace)
	}
	if params.DeployMinioOperator {
		namespaces = append(namespaces, minioOperatorNamespace)
	}

	// Remove previously-installed operator namespaces and resources.
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

	// The perseus and bifrost utilities cannot have downtime so they are not
	// part of the standard namespace cleanup and recreation flow. We always
	// only update both.
	perseusNamespace := "perseus"
	namespaces = append(namespaces, perseusNamespace)
	bifrostNamespace := "bifrost"
	namespaces = append(namespaces, bifrostNamespace)

	logger.Info("Creating utility namespaces")
	_, err = k8sClient.CreateOrUpdateNamespaces(namespaces)
	if err != nil {
		return errors.Wrap(err, "failed to create bifrost namespace")
	}

	if deployPerseus {
		logger.Info("Creating or updating perseus secret")
		_, err = k8sClient.CreateOrUpdateSecret(perseusNamespace, perseusSecret)
		if err != nil {
			return errors.Wrap(err, "failed to create perseus secret")
		}
	}

	logger.Info("Creating or updating bifrost secret")
	_, err = k8sClient.CreateOrUpdateSecret(bifrostNamespace, bifrostSecret)
	if err != nil {
		return errors.Wrap(err, "failed to create bifrost secret")
	}

	// Notes about the Mattermost Operator CRDs:
	//
	// The Mattermost Helm chart will install the Mattermost CRDs if they are
	// missing on a given k8s cluster, but it will not upgrade them. This follows
	// the general helm documentation for CRDs:
	// https://helm.sh/docs/chart_best_practices/custom_resource_definitions/#some-caveats-and-explanations
	//
	// For now, we will update the CRDs the same way we did before by updating
	// them from manifest files in this repo. We may want to change this in the
	// future. Perhaps we should follow how nginx does it?
	// https://github.com/nginxinc/kubernetes-ingress/tree/main/charts/nginx-ingress#upgrading-the-crds
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
			Path:            "manifests/bifrost/bifrost.yaml",
			DeployNamespace: bifrostNamespace,
		}, {
			Path:            "manifests/external-snapshotter/external-snapshotter.yaml",
			DeployNamespace: "kube-system",
		},
	}

	if cluster.Provisioner == model.ProvisionerKops {
		// Only deploy calico CNI at cluster creation time if networking option is calico
		if cluster.ProvisionerMetadataKops.ChangeRequest != nil &&
			len(cluster.ProvisionerMetadataKops.ChangeRequest.Networking) != 0 &&
			cluster.ProvisionerMetadataKops.ChangeRequest.Networking == "calico" {
			files = append(files, k8s.ManifestFile{
				Path:            "manifests/calico-cni.yaml",
				DeployNamespace: "kube-system",
			})
		}

		// Only deploy or re-provision calico netpol if current networking option is other than calico
		if (cluster.ProvisionerMetadataKops.ChangeRequest != nil &&
			len(cluster.ProvisionerMetadataKops.ChangeRequest.Networking) != 0 && cluster.ProvisionerMetadataKops.ChangeRequest.Networking != "calico") ||
			(len(cluster.ProvisionerMetadataKops.Networking) > 0 && cluster.ProvisionerMetadataKops.Networking != "calico") {
			files = append(files, k8s.ManifestFile{
				Path:            "manifests/calico-network-policy-only.yaml",
				DeployNamespace: "kube-system",
			})
		}
	}

	if deployPerseus {
		files = append(files, k8s.ManifestFile{
			Path:            "manifests/perseus/perseus.yaml",
			DeployNamespace: perseusNamespace,
		})
	}

	if params.DeployMysqlOperator {
		files = append(files, k8s.ManifestFile{
			Path:            "manifests/operator-manifests/mysql/mysql-operator.yaml",
			DeployNamespace: mysqlOperatorNamespace,
		})
	}

	if params.DeployMinioOperator {
		files = append(files, k8s.ManifestFile{
			Path:            "manifests/operator-manifests/minio/minio-operator.yaml",
			DeployNamespace: minioOperatorNamespace,
		})
	}

	var manifestFiles []k8s.ManifestFile
	if cluster.Provisioner == model.ProvisionerEKS {
		manifestFiles = append(manifestFiles, k8s.ManifestFile{
			// some manifest requires 'kops-csi-1-21' storageClass
			// which is not available by default in EKS
			// TODO: we need separate manifest/helm for kops & eks
			Path: "manifests/storageclass.yaml",
		})
	}

	manifestFiles = append(manifestFiles, files...)

	err = k8sClient.CreateFromFiles(manifestFiles)
	if err != nil {
		return err
	}

	if params.MattermostOperatorHelmDir != "" {
		logger.Debugf("Upgrading Mattermost Operator helm chart from local chart %s", params.MattermostOperatorHelmDir)

		err = helmClient.FullyUpgradeLocalChart("mattermost-operator", params.MattermostOperatorHelmDir, "mattermost-operator", "helm-charts/mattermost-operator.yaml")
		if err != nil {
			return errors.Wrap(err, "failed to upgrade local mattermost helm chart")
		}
	} else {
		logger.Debug("Upgrading Mattermost Operator helm chart")

		err = helmClient.RepoAdd("mattermost", "https://helm.mattermost.com")
		if err != nil {
			return errors.Wrap(err, "failed to add mattermost helm repo")
		}

		err = helmClient.RepoUpdate()
		if err != nil {
			return errors.Wrap(err, "failed to ensure helm repos are updated")
		}

		err = helmClient.RunGenericCommand(
			"upgrade", "mattermost-operator", "mattermost/mattermost-operator",
			"-n", "mattermost-operator",
			"-f", "helm-charts/mattermost-operator.yaml",
		)
		if err != nil {
			return errors.Wrap(err, "failed to upgrade mattermost helm chart")
		}
	}

	// change the waiting time because creation can take more time
	// due container download / init / container creation / volume allocation
	wait = 240
	appsWithDeployment := map[string]string{
		"mattermost-operator": mattermostOperatorNamespace,
		"bifrost":             bifrostNamespace,
	}
	if cluster.Provisioner == model.ProvisionerKops {
		if (cluster.ProvisionerMetadataKops != nil && cluster.ProvisionerMetadataKops.Networking == "calico") ||
			(cluster.ProvisionerMetadataKops.ChangeRequest != nil && cluster.ProvisionerMetadataKops.ChangeRequest.Networking == "calico") {
			appsWithDeployment["calico-typha-horizontal-autoscaler"] = "kube-system"
		}
	}

	if deployPerseus {
		appsWithDeployment["perseus"] = perseusNamespace
	}

	if params.DeployMinioOperator {
		appsWithDeployment["minio-operator"] = minioOperatorNamespace
	}

	for deployment, namespace := range appsWithDeployment {
		var pods *v1.PodList
		pods, err = k8sClient.GetPodsFromDeployment(namespace, deployment)
		if err != nil {
			return err
		}
		if len(pods.Items) == 0 {
			return fmt.Errorf("no pods found from %q deployment", deployment)
		}

		for _, pod := range pods.Items {
			logger.Infof("Waiting up to %d seconds for %q pod %q to start...", wait, deployment, pod.GetName())
			ctxGetPods, cancelGetPods := context.WithTimeout(context.Background(), time.Duration(wait)*time.Second)
			defer cancelGetPods()
			_, err = k8sClient.WaitForPodRunning(ctxGetPods, namespace, pod.GetName())
			if err != nil {
				return err
			}
			logger.Infof("Successfully deployed service pod %q", pod.GetName())
		}
	}

	var operatorsWithStatefulSet []string
	if params.DeployMysqlOperator {
		operatorsWithStatefulSet = append(operatorsWithStatefulSet, "mysql-operator")
	}

	for _, operator := range operatorsWithStatefulSet {
		var pods *v1.PodList
		pods, err = k8sClient.GetPodsFromStatefulset(operator, operator)
		if err != nil {
			return err
		}
		if len(pods.Items) == 0 {
			return fmt.Errorf("no pods found from %q statefulSet", operator)
		}

		for _, pod := range pods.Items {
			logger.Infof("Waiting up to %d seconds for %q pod %q to start...", wait, operator, pod.GetName())
			ctxPodRunning, cancelPodRunning := context.WithTimeout(context.Background(), time.Duration(wait)*time.Second)
			defer cancelPodRunning()
			var podRunning *v1.Pod
			podRunning, err = k8sClient.WaitForPodRunning(ctxPodRunning, operator, pod.GetName())
			if err != nil {
				return err
			}
			logger.Infof("Successfully deployed service pod %q", podRunning.GetName())
		}
	}

	ugh, err := utility.NewUtilityGroupHandle(params.AllowCIDRRangeList, kubeconfigPath, tempDir, cluster, awsClient, gitClient, argocdClient, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create new cluster utility group handle")
	}

	err = ugh.ProvisionUtilityGroup()
	if err != nil {
		return errors.Wrap(err, "failed to upgrade all services in utility group")
	}

	prom, _ := k8sClient.GetNamespace(prometheus.Namespace)

	if prom != nil && prom.Name != "" {
		err = prometheus.PrepareSloth(k8sClient, logger)
		if err != nil {
			return errors.Wrap(err, "failed to prepare Sloth")
		}
	}

	// Sync PGBouncer configmap if there is any change
	vpc := cluster.VpcID()
	if vpc == "" {
		vpc, err = awsClient.GetClaimedVPC(cluster.ID, logger)
		if err != nil {
			return errors.Wrap(err, "failed to lookup cluster VPC")
		}
	}

	ctx, cancel = context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
	defer cancel()
	err = pgbouncer.UpdatePGBouncerConfigMap(ctx, vpc, store, cluster.PgBouncerConfig, k8sClient, logger)
	if err != nil {
		return errors.Wrap(err, "failed to update configmap for pgbouncer-configmap")
	}
	logger.Info("pgbouncer configmap updated successfully")

	var clusterName string
	if cluster.Provisioner == model.ProvisionerKops {
		clusterName = cluster.ProvisionerMetadataKops.Name
	} else if cluster.Provisioner == model.ProvisionerEKS {
		clusterName = cluster.ProvisionerMetadataEKS.Name
	}

	logger.WithField("name", clusterName).Info("Successfully provisioned cluster")

	return nil
}

// migrateMattermostOperatorToHelm performs the migration steps needed to move
// the Mattermost operator deployment over to helm.
// TODO: this can be removed once all migrations are complete.
func migrateMattermostOperatorToHelm(kubeconfigPath string, logger logrus.FieldLogger) error {
	logger = logger.WithField("helm-migration", "mattermost-operator")
	logger.Info("Checking status of Mattermost Operator helm migration...")

	k8sClient, err := k8s.NewFromFile(kubeconfigPath, logger)
	if err != nil {
		return errors.Wrap(err, "failed to initialize K8s client from kubeconfig")
	}
	helmClient, err := helm.New(kubeconfigPath, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create a new helm client")
	}

	_, deployed := helmClient.HelmChartFoundAndDeployed("mattermost-operator", "mattermost-operator")
	if deployed {
		logger.Info("Mattermost Operator helm chart is already deployed; skipping migration process")
		return nil
	}

	logger.Info("Checking if this is an unprovisioned cluster...")
	ctx := context.TODO()
	_, err = k8sClient.Clientset.AppsV1().Deployments("mattermost-operator").Get(ctx, "mattermost-operator", metav1.GetOptions{})
	if err != nil && k8sErrors.IsNotFound(err) {
		logger.Info("No Mattermost Operator deployment found; proceeding as if this is a new cluster")
		return nil
	}
	if err != nil {
		return errors.Wrap(err, "encountered unexpected error try to retrieve Mattermost Operator deployment")
	}

	logger.Info("Beginning helm adoption process for the Mattermost Operator resources")

	// Some resources must be "adopted" via specifc metadata changes:
	// https://github.com/helm/helm/pull/7649
	err = applyHelmMigrationMetadata(k8sClient)
	if err != nil {
		return errors.Wrap(err, "prepare mattermost-operator resources for helm migration")
	}

	logger.Info("Metadata migration complete; starting namespace cleanup...")

	// The remaining operator resources exist in the Mattermost operator
	// namespace. Some of these resources need to have fields updated that
	// are immutable. To get around this we will delete all of the resources
	// and let the helm chart recreate them.
	namespace := "mattermost-operator"
	logger.Infof("Cleaning up namespace %s", namespace)
	ctx = context.TODO()
	err = k8sClient.Clientset.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
	if k8sErrors.IsNotFound(err) {
		logger.Infof("Namespace %s not found; skipping...", namespace)
	} else if err != nil {
		return errors.Wrapf(err, "failed to delete namespace %s", namespace)
	}

	wait := 60
	logger.Infof("Waiting up to %d seconds for namespaces to be terminated...", wait)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(wait)*time.Second)
	defer cancel()
	err = waitForNamespacesDeleted(ctx, []string{namespace}, k8sClient)
	if err != nil {
		return err
	}

	return nil
}

func applyHelmMigrationMetadata(k8sClient *k8s.KubeClient) error {
	ctx := context.TODO()
	clusterRole, err := k8sClient.Clientset.RbacV1().ClusterRoles().Get(ctx, "mattermost-operator", metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get mattermost-operator clusterrole")
	}
	if clusterRole.Annotations == nil {
		clusterRole.Annotations = make(map[string]string)
	}
	if clusterRole.Labels == nil {
		clusterRole.Labels = make(map[string]string)
	}
	clusterRole.Annotations["meta.helm.sh/release-name"] = "mattermost-operator"
	clusterRole.Annotations["meta.helm.sh/release-namespace"] = "mattermost-operator"
	clusterRole.Labels["app.kubernetes.io/managed-by"] = "Helm"
	_, err = k8sClient.Clientset.RbacV1().ClusterRoles().Update(ctx, clusterRole, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update mattermost-operator clusterrole")
	}

	clusterRoleBinding, err := k8sClient.Clientset.RbacV1().ClusterRoleBindings().Get(ctx, "mattermost-operator", metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get mattermost-operator clusterrolebinding")
	}
	if clusterRoleBinding.Annotations == nil {
		clusterRoleBinding.Annotations = make(map[string]string)
	}
	if clusterRoleBinding.Labels == nil {
		clusterRoleBinding.Labels = make(map[string]string)
	}
	clusterRoleBinding.Annotations["meta.helm.sh/release-name"] = "mattermost-operator"
	clusterRoleBinding.Annotations["meta.helm.sh/release-namespace"] = "mattermost-operator"
	clusterRoleBinding.Labels["app.kubernetes.io/managed-by"] = "Helm"
	_, err = k8sClient.Clientset.RbacV1().ClusterRoleBindings().Update(ctx, clusterRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update mattermost-operator clusterrolebinding")
	}

	return nil
}
