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

// CreateOrUpdateSecret creates or update a secret
func (kc *KubeClient) CreateOrUpdateSecret(namespace string, secret *corev1.Secret) (metav1.Object, error) {
	ctx := context.TODO()
	_, err := kc.Clientset.CoreV1().Secrets(namespace).Get(ctx, secret.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	}

	return kc.Clientset.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
}

// GetSecrets returns a list of kubernetes secret objects if they exist.
//
// Any errors will be returned immediately and the remaining secrets will be
// skipped.
func (kc *KubeClient) GetSecrets(nameSpace string, secretsNames []string) ([]*corev1.Secret, error) {
	ctx := context.TODO()
	secrets := []*corev1.Secret{}
	for _, secretsName := range secretsNames {
		secret, err := kc.Clientset.CoreV1().Secrets(nameSpace).Get(ctx, secretsName, metav1.GetOptions{})
		if err != nil {
			return secrets, err
		}
		secrets = append(secrets, secret)
	}

	return secrets, nil
}
