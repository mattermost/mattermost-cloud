// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func provisionCluster(
	cluster *model.Cluster,
	kubeconfigPath string,
	awsClient aws.AWS,
	params ProvisioningParams,
	store model.ClusterUtilityDatabaseStoreInterface,
	logger logrus.FieldLogger) error {

	// Start by gathering resources that will be needed later. If any of this
	// fails then no cluster changes have been made which reduces risk.
	bifrostSecret, err := awsClient.GenerateBifrostUtilitySecret(cluster.ID, logger)
	if err != nil {
		return errors.Wrap(err, "failed to generate bifrost secret")
	}

	// Begin deploying the mattermost operator.
	k8sClient, err := k8s.NewFromFile(kubeconfigPath, logger)
	if err != nil {
		return errors.Wrap(err, "failed to initialize K8s client from kubeconfig")
	}

	mysqlOperatorNamespace := "mysql-operator"
	minioOperatorNamespace := "minio-operator"
	mattermostOperatorNamespace := "mattermost-operator"

	namespaces := []string{
		mattermostOperatorNamespace,
	}

	if params.DeployMysqlOperator {
		namespaces = append(namespaces, mysqlOperatorNamespace)
	}

	if params.DeployMinioOperator {
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

	err = k8sClient.Clientset.AppsV1().DaemonSets("kube-system").Delete(ctx, "k8s-spot-termination-handler", metav1.DeleteOptions{})
	if k8sErrors.IsNotFound(err) {
		logger.Info("DaemonSet k8s-spot-termination-handler not found; skipping...")
	} else if err != nil {
		return errors.Wrap(err, "failed to delete DaemonSet k8s-spot-termination-handler")
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
			Path:            "manifests/k8s-spot-termination-handler/k8s-spot-termination-handler.yaml",
			DeployNamespace: "kube-system",
		},
	}

	// Do not deploy calico if we use EKS
	if cluster.ProvisionerMetadataEKS == nil {
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

	err = k8sClient.CreateFromFiles(files)
	if err != nil {
		return err
	}

	// change the waiting time because creation can take more time
	// due container download / init / container creation / volume allocation
	wait = 240
	appsWithDeployment := map[string]string{
		"mattermost-operator": mattermostOperatorNamespace,
		"bifrost":             bifrostNamespace,
	}
	if cluster.ProvisionerMetadataEKS == nil {
		if (cluster.ProvisionerMetadataKops != nil && cluster.ProvisionerMetadataKops.Networking == "calico") ||
			(cluster.ProvisionerMetadataKops.ChangeRequest != nil && cluster.ProvisionerMetadataKops.ChangeRequest.Networking == "calico") {
			appsWithDeployment["calico-typha-horizontal-autoscaler"] = "kube-system"
		}
	}

	if params.DeployMinioOperator {
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
	if params.DeployMysqlOperator {
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

	wait = 240
	supportAppsWithDaemonSets := map[string]string{
		//"calico-node":                  "kube-system",
		"k8s-spot-termination-handler": "kube-system",
	}
	for daemonSet, namespace := range supportAppsWithDaemonSets {
		if daemonSet == "k8s-spot-termination-handler" && (len(os.Getenv(model.MattermostChannel)) > 0 || len(os.Getenv(model.MattermostWebhook)) > 0) {
			logger.Infof("Waiting up to %d seconds for %q daemonset to get it...", wait, daemonSet)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(wait)*time.Second)
			defer cancel()
			daemonSetObj, err := k8sClient.Clientset.AppsV1().DaemonSets(namespace).Get(ctx, daemonSet, metav1.GetOptions{})
			if err != nil {
				return errors.Wrapf(err, " failed to get daemonSet %s", daemonSet)
			}
			var payload []k8s.PatchStringValue
			if daemonSetObj.Spec.Selector != nil {
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

	if cluster.ProvisionerMetadataKops != nil {
		iamRole := fmt.Sprintf("nodes.%s", cluster.ProvisionerMetadataKops.Name)
		err = awsClient.AttachPolicyToRole(iamRole, aws.CustomNodePolicyName, logger)
		if err != nil {
			return errors.Wrap(err, "unable to attach custom node policy")
		}

		err = awsClient.AttachPolicyToRole(iamRole, aws.VeleroNodePolicyName, logger)
		if err != nil {
			return errors.Wrap(err, "unable to attach velero node policy")
		}
	}

	ugh, err := newUtilityGroupHandle(params, kubeconfigPath, cluster, awsClient, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create new cluster utility group handle")
	}

	err = ugh.ProvisionUtilityGroup()
	if err != nil {
		return errors.Wrap(err, "failed to upgrade all services in utility group")
	}

	prom, _ := k8sClient.GetNamespace("prometheus")

	if prom != nil && prom.Name != "" {
		err = prepareSloth(k8sClient, logger)
		if err != nil {
			return errors.Wrap(err, "failed to prepare Sloth")
		}
	}

	if err := createOrUpdateClusterSLOs(cluster, k8sClient, logger); err != nil {
		return errors.Wrap(err, "failed to create cluster slos")
	}

	// Sync PGBouncer configmap if there is any change
	var vpc string
	if cluster.ProvisionerMetadataKops != nil {
		vpc = cluster.ProvisionerMetadataKops.VPC
	} else if cluster.ProvisionerMetadataEKS != nil {
		vpc = cluster.ProvisionerMetadataEKS.VPC
	} else {
		return errors.New("cluster metadata is nil cannot determine VPC")
	}
	ctx, cancel = context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
	defer cancel()
	err = updatePGBouncerConfigMap(ctx, vpc, store, params.PGBouncerConfig, k8sClient, logger)
	if err != nil {
		return errors.Wrap(err, "failed to update configmap for pgbouncer-configmap")
	}
	logger.Info("pgbouncer configmap updated successfully")

	clusterName := cluster.ID
	if cluster.ProvisionerMetadataKops != nil {
		clusterName = cluster.ProvisionerMetadataKops.Name
	}

	logger.WithField("name", clusterName).Info("Successfully provisioned cluster")

	return nil
}
