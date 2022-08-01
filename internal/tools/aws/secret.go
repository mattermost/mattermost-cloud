// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InstallationDBSecret represents data required for creating database
// secret for an Installation.
type InstallationDBSecret struct {
	InstallationSecretName string
	ConnectionString       string
	DBCheckURL             string
	ReadReplicasURL        string
}

// ToK8sSecret creates Kubernetes secret from InstallationDBSecret.
func (s InstallationDBSecret) ToK8sSecret(disableDBCheck bool) *corev1.Secret {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: s.InstallationSecretName,
		},
		StringData: map[string]string{
			"DB_CONNECTION_STRING":              s.ConnectionString,
			"MM_SQLSETTINGS_DATASOURCEREPLICAS": s.ReadReplicasURL,
		},
	}
	if !disableDBCheck && s.DBCheckURL != "" {
		secret.StringData["DB_CONNECTION_CHECK_URL"] = s.DBCheckURL
	}

	return &secret
}
