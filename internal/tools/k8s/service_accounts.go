package k8s

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (kc *KubeClient) createServiceAccount(namespace string, account *corev1.ServiceAccount) (metav1.Object, error) {
	client := kc.Clientset.CoreV1().ServiceAccounts(namespace)

	result, err := client.Create(account)
	if err != nil {
		return nil, err
	}

	return result.GetObjectMeta(), nil
}
