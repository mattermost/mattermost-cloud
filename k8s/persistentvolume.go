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

func (kc *KubeClient) createOrUpdatePersistentVolume(volume *corev1.PersistentVolume) (metav1.Object, error) {
	ctx := context.TODO()
	_, err := kc.Clientset.CoreV1().PersistentVolumes().Get(ctx, volume.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.CoreV1().PersistentVolumes().Create(ctx, volume, metav1.CreateOptions{})
	}

	return kc.Clientset.CoreV1().PersistentVolumes().Update(ctx, volume, metav1.UpdateOptions{})
}

func (kc *KubeClient) createOrUpdatePersistentVolumeClaim(namespace string, volumeClaim *corev1.PersistentVolumeClaim) (metav1.Object, error) {
	ctx := context.TODO()
	_, err := kc.Clientset.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, volumeClaim.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, volumeClaim, metav1.CreateOptions{})
	}

	return kc.Clientset.CoreV1().PersistentVolumeClaims(namespace).Update(ctx, volumeClaim, metav1.UpdateOptions{})
}
