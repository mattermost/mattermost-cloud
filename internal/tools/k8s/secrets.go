package k8s

import (
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (kc *KubeClient) createOrUpdateSecret(namespace string, secret *corev1.Secret) (metav1.Object, error) {
	_, err := kc.Clientset.CoreV1().Secrets(namespace).Get(secret.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.CoreV1().Secrets(namespace).Create(secret)
	}

	return kc.Clientset.CoreV1().Secrets(namespace).Update(secret)
}
