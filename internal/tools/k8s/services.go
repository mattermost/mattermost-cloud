package k8s

import (
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (kc *KubeClient) createOrUpdateService(namespace string, service *corev1.Service) (metav1.Object, error) {
	_, err := kc.Clientset.CoreV1().Services(namespace).Get(service.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.CoreV1().Services(namespace).Create(service)
	}

	return kc.Clientset.CoreV1().Services(namespace).Update(service)
}
