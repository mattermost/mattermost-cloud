// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	rbacv1 "k8s.io/api/rbac/v1"
	rbacbetav1 "k8s.io/api/rbac/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (kc *KubeClient) createOrUpdateRoleBindingV1(namespace string, binding *rbacv1.RoleBinding) (metav1.Object, error) {
	_, err := kc.Clientset.RbacV1().RoleBindings(namespace).Get(binding.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.RbacV1().RoleBindings(namespace).Create(binding)
	}

	return kc.Clientset.RbacV1().RoleBindings(namespace).Update(binding)
}

func (kc *KubeClient) createOrUpdateRoleBindingBetaV1(namespace string, binding *rbacbetav1.RoleBinding) (metav1.Object, error) {
	_, err := kc.Clientset.RbacV1beta1().RoleBindings(namespace).Get(binding.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.RbacV1beta1().RoleBindings(namespace).Create(binding)
	}

	return kc.Clientset.RbacV1beta1().RoleBindings(namespace).Update(binding)
}

func (kc *KubeClient) createOrUpdateClusterRoleBindingV1(binding *rbacv1.ClusterRoleBinding) (metav1.Object, error) {
	_, err := kc.Clientset.RbacV1().ClusterRoleBindings().Get(binding.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.RbacV1().ClusterRoleBindings().Create(binding)
	}

	return kc.Clientset.RbacV1().ClusterRoleBindings().Update(binding)
}

func (kc *KubeClient) createOrUpdateClusterRoleBindingBetaV1(binding *rbacbetav1.ClusterRoleBinding) (metav1.Object, error) {
	_, err := kc.Clientset.RbacV1beta1().ClusterRoleBindings().Get(binding.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.RbacV1beta1().ClusterRoleBindings().Create(binding)
	}

	return kc.Clientset.RbacV1beta1().ClusterRoleBindings().Update(binding)
}
