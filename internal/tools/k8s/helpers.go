package k8s

import (
	"errors"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// WaitForPodRunning will poll a given kubernetes pod at a regular interval for
// it to enter the 'Running' state. If the pod fails to become ready before
// the provided timeout then an error will be returned.
func (kc *KubeClient) WaitForPodRunning(namespace, podName string, timeout int) (*corev1.Pod, error) {
	timer := time.NewTimer(time.Duration(timeout) * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			return &corev1.Pod{}, errors.New("timed out waiting for pod to become ready")
		default:
			pod, err := kc.Clientset.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
			if err == nil {
				if pod.Status.Phase == corev1.PodRunning {
					return pod, nil
				}
			}

			time.Sleep(5 * time.Second)
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
