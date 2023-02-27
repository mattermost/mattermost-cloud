// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	"context"

	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/api/storage/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpdateStorageClassVolumeBindingMode updates the storage class volume binding mode from immediate to WaitForFirstConsumer.
func (kc *KubeClient) UpdateStorageClassVolumeBindingMode(class string) (metav1.Object, error) {
	ctx := context.TODO()
	storageClass, err := kc.Clientset.StorageV1beta1().StorageClasses().Get(ctx, class, metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}
	bindingMode := v1beta1.VolumeBindingWaitForFirstConsumer
	storageClass.VolumeBindingMode = &bindingMode
	storageClass.ResourceVersion = ""

	err = kc.Clientset.StorageV1beta1().StorageClasses().Delete(ctx, class, metav1.DeleteOptions{})
	if err != nil && k8sErrors.IsNotFound(err) {
		return nil, err
	}
	return kc.Clientset.StorageV1beta1().StorageClasses().Create(ctx, storageClass, metav1.CreateOptions{})
}

func (kc *KubeClient) createOrUpdateStorageClass(storage *storagev1.StorageClass) (metav1.Object, error) {
	ctx := context.TODO()
	_, err := kc.Clientset.StorageV1().StorageClasses().Get(ctx, storage.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.StorageV1().StorageClasses().Create(ctx, storage, metav1.CreateOptions{})
	}

	return kc.Clientset.StorageV1().StorageClasses().Update(ctx, storage, metav1.UpdateOptions{})
}
