// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateOrUpdateNamespaces creates or update kubernetes namespaces
//
// Any errors will be returned immediately and the remaining namespaces will be
// skipped.
func (kc *KubeClient) CreateOrUpdateNamespaces(namespaceNames []string) ([]*corev1.Namespace, error) {
	namespaces := []*corev1.Namespace{}
	for _, namespaceName := range namespaceNames {
		namespace, err := kc.CreateOrUpdateNamespace(namespaceName)
		if err != nil {
			return namespaces, err
		}
		namespaces = append(namespaces, namespace)
	}

	return namespaces, nil
}

// CreateOrUpdateNamespace creates or update a namespace
func (kc *KubeClient) CreateOrUpdateNamespace(namespaceName string) (*corev1.Namespace, error) {
	_, err := kc.Clientset.CoreV1().Namespaces().Get(namespaceName, metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	nsSpec := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
			Labels: map[string]string{
				"name": namespaceName,
			},
		},
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.CoreV1().Namespaces().Create(nsSpec)
	}

	return kc.Clientset.CoreV1().Namespaces().Update(nsSpec)
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
