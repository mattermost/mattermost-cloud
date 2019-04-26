package k8s

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateNamespace creates a kubernetes namespace.
func (kc *KubeClient) CreateNamespace(namespace string) (*corev1.Namespace, error) {
	clientset, err := kc.getKubeConfigClientset()
	if err != nil {
		return &corev1.Namespace{}, err
	}

	// Check if namespace exists first.
	ns, err := kc.GetNamespace(namespace)
	if err == nil {
		return ns, nil
	}

	nsSpec := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	return clientset.CoreV1().Namespaces().Create(nsSpec)
}

// GetNamespace returns a given kubernetes namespace object if it exists.
func (kc *KubeClient) GetNamespace(namespace string) (*corev1.Namespace, error) {
	clientset, err := kc.getKubeConfigClientset()
	if err != nil {
		return &corev1.Namespace{}, err
	}

	return clientset.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
}
