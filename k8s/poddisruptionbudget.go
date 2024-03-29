// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	"context"

	v1 "k8s.io/api/policy/v1"
	v1beta1 "k8s.io/api/policy/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (kc *KubeClient) createOrUpdatePodDisruptionBudgetBetaV1(namespace string, podDisruptionBudget *v1beta1.PodDisruptionBudget) (metav1.Object, error) {
	ctx := context.TODO()
	_, err := kc.Clientset.PolicyV1beta1().PodDisruptionBudgets(namespace).Get(ctx, podDisruptionBudget.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.PolicyV1beta1().PodDisruptionBudgets(namespace).Create(ctx, podDisruptionBudget, metav1.CreateOptions{})
	}

	return kc.Clientset.PolicyV1beta1().PodDisruptionBudgets(namespace).Update(ctx, podDisruptionBudget, metav1.UpdateOptions{})
}

func (kc *KubeClient) createOrUpdatePodDisruptionBudgetV1(namespace string, podDisruptionBudget *v1.PodDisruptionBudget) (metav1.Object, error) {
	ctx := context.TODO()
	pdb, err := kc.Clientset.PolicyV1().PodDisruptionBudgets(namespace).Get(ctx, podDisruptionBudget.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.PolicyV1().PodDisruptionBudgets(namespace).Create(ctx, podDisruptionBudget, metav1.CreateOptions{})
	}

	// TODO: Fix update fail due to the error (metadata.resourceVersion: Invalid value: 0x0: must be specified for an update)
	pdb.Spec = podDisruptionBudget.Spec
	return kc.Clientset.PolicyV1().PodDisruptionBudgets(namespace).Update(ctx, pdb, metav1.UpdateOptions{})
}
