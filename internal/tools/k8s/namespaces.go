package k8s

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateNamespacesIfDoesNotExist creates kubernetes namespaces if they don't
// exist already.
//
// Any errors will be returned immediately and the remaining namespaces will be
// skipped.
func (kc *KubeClient) CreateNamespacesIfDoesNotExist(namespaceNames []string) ([]*corev1.Namespace, error) {
	namespaces := []*corev1.Namespace{}
	for _, namespaceName := range namespaceNames {
		namespace, err := kc.CreateNamespaceIfDoesNotExist(namespaceName)
		if err != nil {
			return namespaces, err
		}
		namespaces = append(namespaces, namespace)
	}

	return namespaces, nil
}

// CreateNamespaceIfDoesNotExist creates a kubernetes namespace if it doesn't
// exist already.
func (kc *KubeClient) CreateNamespaceIfDoesNotExist(namespaceName string) (*corev1.Namespace, error) {
	ns, err := kc.Clientset.CoreV1().Namespaces().Get(namespaceName, metav1.GetOptions{})
	if err == nil {
		return ns, nil
	}

	nsSpec := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name: namespaceName,
		Annotations: map[string]string{
			"sidecar.jaegertracing.io/inject": "true",
		},
	}}
	return kc.Clientset.CoreV1().Namespaces().Create(nsSpec)
}

// GetNamespaces returns a list of kubernetes namespace objects if they exist.
//
// Any errors will be returned immediately and the remaining namespaces will be
// skipped.
func (kc *KubeClient) GetNamespaces(namespaceNames []string) ([]*corev1.Namespace, error) {
	namespaces := []*corev1.Namespace{}
	for _, namespaceName := range namespaceNames {
		namespace, err := kc.Clientset.CoreV1().Namespaces().Get(namespaceName, metav1.GetOptions{})
		if err != nil {
			return namespaces, err
		}
		namespaces = append(namespaces, namespace)
	}

	return namespaces, nil
}

// DeleteNamespaces deletes kubernetes namespaces.
//
// Any errors will be returned immediately and the remaining namespaces will be
// skipped.
func (kc *KubeClient) DeleteNamespaces(namespaceNames []string) error {
	for _, namespaceName := range namespaceNames {
		err := kc.Clientset.CoreV1().Namespaces().Delete(namespaceName, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}
