// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (kc *KubeClient) createOrUpdateConfigMap(namespace string, configmap *corev1.ConfigMap) (metav1.Object, error) {
	_, err := kc.Clientset.CoreV1().ConfigMaps(namespace).Get(configmap.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.CoreV1().ConfigMaps(namespace).Create(configmap)
	}

	return kc.Clientset.CoreV1().ConfigMaps(namespace).Update(configmap)
}
