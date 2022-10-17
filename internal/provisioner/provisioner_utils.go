// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
