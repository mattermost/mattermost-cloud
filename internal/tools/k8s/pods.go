package k8s

import (
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetPods returns a list of pods in a given namespace.
func (kc *KubeClient) GetPods(namespace string) ([]apiv1.Pod, error) {
	clientset, err := kc.getKubeConfigClientset()
	if err != nil {
		return nil, err
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return pods.Items, nil
}
