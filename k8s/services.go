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
	existing, err := kc.Clientset.CoreV1().Services(namespace).Get(ctx, service.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
	}

	// The following values need to be set to the existing resource or the k8s
	// API will complain.
	// TODO: revisit this and possibly use Patch instead of update.
	if len(existing.Spec.ClusterIP) != 0 {
		service.Spec.ClusterIP = existing.Spec.ClusterIP
	}
	if len(existing.ResourceVersion) != 0 {
		service.ResourceVersion = existing.ResourceVersion
	}
	return kc.Clientset.CoreV1().Services(namespace).Update(ctx, service, metav1.UpdateOptions{})
}
