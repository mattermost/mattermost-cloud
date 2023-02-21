// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	"context"

	"github.com/pkg/errors"
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
	ctx := context.TODO()
	_, err := kc.Clientset.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
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
		return kc.Clientset.CoreV1().Namespaces().Create(ctx, nsSpec, metav1.CreateOptions{})
	}

	return kc.Clientset.CoreV1().Namespaces().Update(ctx, nsSpec, metav1.UpdateOptions{})
}

// GetNamespace returns a kubernetes namespace object if it exists.
func (kc *KubeClient) GetNamespace(namespaceName string) (*corev1.Namespace, error) {
	ctx := context.TODO()
	namespace, err := kc.Clientset.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return namespace, err
	}

	return namespace, nil
}

// GetNamespaces returns a list of kubernetes namespace objects if they exist.
//
// Any errors will be returned immediately and the remaining namespaces will be
// skipped.
func (kc *KubeClient) GetNamespaces(namespaceNames []string) ([]*corev1.Namespace, error) {
	ctx := context.TODO()
	namespaces := []*corev1.Namespace{}
	for _, namespaceName := range namespaceNames {
		namespace, err := kc.Clientset.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
		if err != nil {
			if k8sErrors.IsNotFound(err) {
				continue
			}
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
	policy := metav1.DeletePropagationForeground
	gracePeriod := int64(0)
	deleteOpts := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
		PropagationPolicy:  &policy,
	}
	ctx := context.TODO()
	for _, namespaceName := range namespaceNames {
		err := kc.Clientset.CoreV1().Namespaces().Delete(ctx, namespaceName, deleteOpts)
		if err != nil {
			if !k8sErrors.IsNotFound(err) {
				return err
			}
		}
	}

	return nil
}

// finalizeNamespaces deletes kubernetes namespaces.
func (kc *KubeClient) finalizeNamespaces(namespaces []*corev1.Namespace) error {
	ctx := context.TODO()
	for i := range namespaces {
		_, err := kc.Clientset.CoreV1().Namespaces().Finalize(ctx, namespaces[i], metav1.UpdateOptions{})
		if err != nil {
			if !k8sErrors.IsNotFound(err) {
				return err
			}
		}
	}

	return nil
}

// DeleteNamespacesWithFinalizer deletes kubernetes namespaces with a finalizer cleanup.
func (kc *KubeClient) DeleteNamespacesWithFinalizer(namespaceNames []string) error {
	if err := kc.DeleteNamespaces(namespaceNames); err != nil {
		return errors.Wrap(err, "failed to delete namespaceNames")
	}

	namespaces, err := kc.GetNamespaces(namespaceNames)
	if err != nil {
		return errors.Wrap(err, "failed to get namespaceNames")
	}

	for i := range namespaces {
		namespaces[i].Spec.Finalizers = []corev1.FinalizerName{}
	}

	if err := kc.finalizeNamespaces(namespaces); err != nil {
		return errors.Wrap(err, "failed to finalize namespaceNames")
	}

	return nil
}
