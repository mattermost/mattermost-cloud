// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	rbacbetav1 "k8s.io/api/rbac/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (kc *KubeClient) createOrUpdateClusterRoleV1(account *rbacv1.ClusterRole) (metav1.Object, error) {
	ctx := context.TODO()
	_, err := kc.Clientset.RbacV1().ClusterRoles().Get(ctx, account.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.RbacV1().ClusterRoles().Create(ctx, account, metav1.CreateOptions{})
	}

	return kc.Clientset.RbacV1().ClusterRoles().Update(ctx, account, metav1.UpdateOptions{})
}

func (kc *KubeClient) createOrUpdateClusterRoleBetaV1(account *rbacbetav1.ClusterRole) (metav1.Object, error) {
	ctx := context.TODO()
	_, err := kc.Clientset.RbacV1beta1().ClusterRoles().Get(ctx, account.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.RbacV1beta1().ClusterRoles().Create(ctx, account, metav1.CreateOptions{})
	}

	return kc.Clientset.RbacV1beta1().ClusterRoles().Update(ctx, account, metav1.UpdateOptions{})
}

func (kc *KubeClient) createOrUpdateRoleV1(account *rbacv1.Role) (metav1.Object, error) {
	ctx := context.TODO()
	_, err := kc.Clientset.RbacV1().Roles(account.GetNamespace()).Get(ctx, account.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.RbacV1().Roles(account.GetNamespace()).Create(ctx, account, metav1.CreateOptions{})
	}

	return kc.Clientset.RbacV1().Roles(account.GetNamespace()).Update(ctx, account, metav1.UpdateOptions{})
}

func (kc *KubeClient) createOrUpdateRoleBetaV1(account *rbacbetav1.Role) (metav1.Object, error) {
	ctx := context.TODO()
	_, err := kc.Clientset.RbacV1beta1().Roles(account.GetNamespace()).Get(ctx, account.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.RbacV1beta1().Roles(account.GetNamespace()).Create(ctx, account, metav1.CreateOptions{})
	}

	return kc.Clientset.RbacV1beta1().Roles(account.GetNamespace()).Update(ctx, account, metav1.UpdateOptions{})
}
