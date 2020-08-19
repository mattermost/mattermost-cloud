// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (kc *KubeClient) createOrUpdateService(namespace string, service *corev1.Service) (metav1.Object, error) {
	ctx := context.TODO()
	_, err := kc.Clientset.CoreV1().Services(namespace).Get(ctx, service.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
	}

	return kc.Clientset.CoreV1().Services(namespace).Update(ctx, service, metav1.UpdateOptions{})
}
