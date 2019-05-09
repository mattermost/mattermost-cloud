package k8s

import (
	"context"
	"time"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// WaitForPodRunning will poll a given kubernetes pod at a regular interval for
// it to enter the 'Running' state. If the pod fails to become ready before
// the provided timeout then an error will be returned.
func (kc *KubeClient) WaitForPodRunning(ctx context.Context, namespace, podName string) (*corev1.Pod, error) {
	for {
		pod, err := kc.Clientset.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
		if err == nil {
			if pod.Status.Phase == corev1.PodRunning {
				return pod, nil
			}
		}

		select {
		case <-ctx.Done():
			return nil, errors.Wrap(ctx.Err(), "timed out waiting for pod to become ready")
		case <-time.After(5 * time.Second):
		}
	}
}

// GetPodsFromDeployment gets the pods that belong to a given deployment.
func (kc *KubeClient) GetPodsFromDeployment(namespace, deploymentName string) (*corev1.PodList, error) {
	deployment, err := kc.Clientset.AppsV1().Deployments(namespace).Get(deploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	set := labels.Set(deployment.GetLabels())
	listOptions := metav1.ListOptions{LabelSelector: set.AsSelector().String()}

	return kc.Clientset.CoreV1().Pods(namespace).List(listOptions)
}
