// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (kc *KubeClient) createOrUpdateDaemonSetV1(namespace string, daemonSet *appsv1.DaemonSet) (metav1.Object, error) {
	ctx := context.TODO()
	_, err := kc.Clientset.AppsV1().DaemonSets(namespace).Get(ctx, daemonSet.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.AppsV1().DaemonSets(namespace).Create(ctx, daemonSet, metav1.CreateOptions{})
	}

	return kc.Clientset.AppsV1().DaemonSets(namespace).Update(ctx, daemonSet, metav1.UpdateOptions{})
}
