// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	appsbetav1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (kc *KubeClient) createOrUpdateDeploymentV1(namespace string, deployment *appsv1.Deployment) (metav1.Object, error) {
	ctx := context.TODO()
	_, err := kc.Clientset.AppsV1().Deployments(namespace).Get(ctx, deployment.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
	}

	return kc.Clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
}

func (kc *KubeClient) createOrUpdateDeploymentBetaV1(namespace string, deployment *appsbetav1.Deployment) (metav1.Object, error) {
	ctx := context.TODO()
	_, err := kc.Clientset.AppsV1beta1().Deployments(namespace).Get(ctx, deployment.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.AppsV1beta1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
	}

	return kc.Clientset.AppsV1beta1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
}

func (kc *KubeClient) createOrUpdateDeploymentBetaV2(namespace string, deployment *appsv1beta2.Deployment) (metav1.Object, error) {
	ctx := context.TODO()
	_, err := kc.Clientset.AppsV1beta2().Deployments(namespace).Get(ctx, deployment.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.AppsV1beta2().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
	}

	return kc.Clientset.AppsV1beta2().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
}
