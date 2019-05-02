package k8s

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateNamespaces creates kubernetes namespaces.
func (kc *KubeClient) CreateNamespaces(namespaceNames []string) ([]*corev1.Namespace, error) {
	namespaces := []*corev1.Namespace{}
	for _, namespaceName := range namespaceNames {
		namespace, err := kc.CreateNamespace(namespaceName)
		if err != nil {
			return namespaces, err
		}
		namespaces = append(namespaces, namespace)
	}

	return namespaces, nil
}

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

// GetNamespaces returns a list of kubernetes namespace objects if they exist.
func (kc *KubeClient) GetNamespaces(namespaceNames []string) ([]*corev1.Namespace, error) {
	namespaces := []*corev1.Namespace{}
	for _, namespaceName := range namespaceNames {
		namespace, err := kc.GetNamespace(namespaceName)
		if err != nil {
			return namespaces, err
		}
		namespaces = append(namespaces, namespace)
	}

	return namespaces, nil
}

// GetNamespace returns a given kubernetes namespace object if it exists.
func (kc *KubeClient) GetNamespace(namespaceName string) (*corev1.Namespace, error) {
	clientset, err := kc.getKubeConfigClientset()
	if err != nil {
		return &corev1.Namespace{}, err
	}

	return clientset.CoreV1().Namespaces().Get(namespaceName, metav1.GetOptions{})
}
