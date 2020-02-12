package k8s

import (
	"k8s.io/api/storage/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpdateStorageClassVolumeBindingMode updates the storage class volume binding mode from immediate to WaitForFirstConsumer.
func (kc *KubeClient) UpdateStorageClassVolumeBindingMode(class string) (metav1.Object, error) {
	storageClass, err := kc.Clientset.StorageV1beta1().StorageClasses().Get(class, metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}
	bindingMode := v1beta1.VolumeBindingWaitForFirstConsumer
	storageClass.VolumeBindingMode = &bindingMode
	storageClass.ResourceVersion = ""

	err = kc.Clientset.StorageV1beta1().StorageClasses().Delete(class, &metav1.DeleteOptions{})
	if err != nil && k8sErrors.IsNotFound(err) {
		return nil, err
	}
	return kc.Clientset.StorageV1beta1().StorageClasses().Create(storageClass)
}
