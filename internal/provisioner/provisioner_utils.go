// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func attachPolicyRoles(cluster *model.Cluster, awsClient aws.AWS, logger log.FieldLogger) error {
	if cluster.Provisioner != model.ProvisionerKops {
		logger.Debugf("Cluster provisioner type is not %s (%s), skipping policy attachment", model.ProvisionerKops, cluster.Provisioner)
		return nil
	}

	logger.Debug("Attaching cluster policies...")

	iamRoleMaster := fmt.Sprintf("masters.%s", cluster.ProvisionerMetadataKops.Name)
	err := awsClient.AttachPolicyToRole(iamRoleMaster, aws.CustomNodePolicyName, logger)
	if err != nil {
		return errors.Wrap(err, "unable to attach custom node policy to master")
	}

	iamRole := fmt.Sprintf("nodes.%s", cluster.ProvisionerMetadataKops.Name)
	err = awsClient.AttachPolicyToRole(iamRole, aws.CustomNodePolicyName, logger)
	if err != nil {
		return errors.Wrap(err, "unable to attach custom node policy")
	}

	err = awsClient.AttachPolicyToRole(iamRole, aws.VeleroNodePolicyName, logger)
	if err != nil {
		return errors.Wrap(err, "unable to attach velero node policy")
	}

	return nil
}

func getPublicLoadBalancerEndpoint(kubeconfigPath string, namespace string, logger log.FieldLogger) (string, error) {
	k8sClient, err := k8s.NewFromFile(kubeconfigPath, logger)
	if err != nil {
		return "", err
	}

	ctx := context.TODO()
	services, err := k8sClient.Clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	for _, service := range services.Items {
		if !strings.HasSuffix(service.Name, "internal") {
			if service.Status.LoadBalancer.Ingress != nil {
				endpoint := service.Status.LoadBalancer.Ingress[0].Hostname
				if endpoint == "" {
					return "", errors.New("loadbalancer endpoint value is empty")
				}

				return endpoint, nil
			}
		}
	}

	return "", errors.New("failed to get NGINX load balancer endpoint")
}

func getClusterResources(kubeconfigPath string, onlySchedulable bool, logger log.FieldLogger) (*k8s.ClusterResources, error) {
	k8sClient, err := k8s.NewFromFile(kubeconfigPath, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create k8s client from file")
	}

	ctx := context.TODO()
	nodes, err := k8sClient.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list nodes")
	}

	var allPods []v1.Pod
	var totalCPU, totalMemory, totalPodCount, totalNodeCount, workerNodeCount, skippedNodeCount int64
	for _, node := range nodes.Items {
		var skipNode bool
		totalNodeCount++

		if onlySchedulable {
			if node.Spec.Unschedulable {
				logger.Debugf("Ignoring unschedulable node %s", node.GetName())
				skippedNodeCount++
				skipNode = true
			}

			// TODO: handle scheduling taints in a more robust way.
			// This is a quick and dirty check for scheduling issues that could
			// lead to false positives. In the future, we should use a scheduling
			// library to perform the check instead.
			for _, taint := range node.Spec.Taints {
				if taint.Effect == v1.TaintEffectNoSchedule || taint.Effect == v1.TaintEffectPreferNoSchedule {
					logger.Debugf("Ignoring node %s with taint '%s'", node.GetName(), taint.ToString())
					skippedNodeCount++
					skipNode = true
					break
				}
			}
		}

		if !skipNode {
			nodePods, err := k8sClient.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
				FieldSelector: fmt.Sprintf("spec.nodeName=%s", node.GetName()),
			})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to list pods for node %s", node.GetName())
			}

			allPods = append(allPods, nodePods.Items...)
			totalCPU += node.Status.Allocatable.Cpu().MilliValue()
			totalMemory += node.Status.Allocatable.Memory().MilliValue()
			totalPodCount += node.Status.Allocatable.Pods().Value()
			workerNodeCount++
		}
	}

	usedCPU, usedMemory := k8s.CalculateTotalPodMilliResourceRequests(allPods)

	logger.Debugf("Resource usage calculated from %d pods on %d worker nodes", len(allPods), workerNodeCount)

	return &k8s.ClusterResources{
		TotalNodeCount:   totalNodeCount,
		SkippedNodeCount: skippedNodeCount,
		WorkerNodeCount:  workerNodeCount,
		MilliTotalCPU:    totalCPU,
		MilliUsedCPU:     usedCPU,
		MilliTotalMemory: totalMemory,
		MilliUsedMemory:  usedMemory,
		TotalPodCount:    totalPodCount,
		UsedPodCount:     int64(len(allPods)),
	}, nil
}

// waitForNamespacesDeleted is used to check when all the provided namespaces
// have been fully terminated.
func waitForNamespacesDeleted(ctx context.Context, namespaces []string, k8sClient *k8s.KubeClient) error {
	for {
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "timed out waiting for namespaces to become fully terminated")
		default:
			var shouldWait bool
			for _, namespace := range namespaces {
				_, err := k8sClient.Clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
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

// createCustomErrorPagesConfigMap creates a ConfigMap with custom error pages.
// The URL in the 403 error page is taken from the PORTAL_URL environment variable.
func createCustomErrorPagesConfigMap(k8sClient *k8s.KubeClient, namespace string) error {
	portalURL := os.Getenv("PORTAL_URL")
	if portalURL == "" {
		return errors.New("PORTAL_URL environment variable is not set")
	}

	// ConfigMap data
	configMapData := map[string]string{
		"404": `<!DOCTYPE html>
<html>
  <head><title>PAGE NOT FOUND</title></head>
  <body>PAGE NOT FOUND</body>
</html>`,
		"503": `<!DOCTYPE html>
<html>
  <head><title>CUSTOM SERVICE UNAVAILABLE</title></head>
  <body>CUSTOM SERVICE UNAVAILABLE</body>
</html>`,
		"403": fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="ie=edge">
    <meta http-equiv="refresh" content="0; url=%s/cloud/inaccessible" />
    <title>HTML 5 Boilerplate</title>
</head>
<body>
</body>
</html>`, portalURL),
	}

	// Create ConfigMap
	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "custom-error-pages",
			Namespace: namespace,
		},
		Data: configMapData,
	}

	_, err := k8sClient.Clientset.CoreV1().ConfigMaps(namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create ConfigMap")
	}

	return nil
}

// ensureNamespaceExists checks if a namespace exists and creates it if it does not.
func ensureNamespaceExists(k8sClient *k8s.KubeClient, namespace string) error {
	_, err := k8sClient.Clientset.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
	if k8sErrors.IsNotFound(err) {
		ns := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		_, err = k8sClient.Clientset.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create namespace")
		}
	} else if err != nil {
		return errors.Wrap(err, "failed to get namespace")
	}
	return nil
}
