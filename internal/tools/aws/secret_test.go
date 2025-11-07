// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInstallationDBSecret_ToK8sSecret(t *testing.T) {
	for _, testCase := range []struct {
		description        string
		installationSecret InstallationDBSecret
		disableDBCheck     bool
		expectedSecret     *corev1.Secret
	}{
		{
			description: "full secret, do not disable check",
			installationSecret: InstallationDBSecret{
				InstallationSecretName: "secret",
				ConnectionString:       "postgres://localhost",
				DBCheckURL:             "postgres://check",
				ReadReplicasURL:        "postgres://read",
			},
			disableDBCheck: false,
			expectedSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "secret",
				},
				StringData: map[string]string{
					"DB_CONNECTION_STRING":              "postgres://localhost",
					"MM_SQLSETTINGS_DATASOURCEREPLICAS": "postgres://read",
					"DB_CONNECTION_CHECK_URL":           "postgres://check",
				},
			},
		},
		{
			description: "full secret, disable check",
			installationSecret: InstallationDBSecret{
				InstallationSecretName: "secret",
				ConnectionString:       "postgres://localhost",
				DBCheckURL:             "postgres://check",
				ReadReplicasURL:        "postgres://read",
			},
			disableDBCheck: true,
			expectedSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "secret",
				},
				StringData: map[string]string{
					"DB_CONNECTION_STRING":              "postgres://localhost",
					"MM_SQLSETTINGS_DATASOURCEREPLICAS": "postgres://read",
				},
			},
		},
		{
			description: "secret without check",
			installationSecret: InstallationDBSecret{
				InstallationSecretName: "secret",
				ConnectionString:       "postgres://localhost",
				ReadReplicasURL:        "postgres://read",
			},
			disableDBCheck: false,
			expectedSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "secret",
				},
				StringData: map[string]string{
					"DB_CONNECTION_STRING":              "postgres://localhost",
					"MM_SQLSETTINGS_DATASOURCEREPLICAS": "postgres://read",
				},
			},
		},
		{
			description: "MySQL secret with datasource URL",
			installationSecret: InstallationDBSecret{
				InstallationSecretName: "mysql-secret",
				ConnectionString:       "mysql://user:pass@tcp(db.example.com:3306)/mattermost?charset=utf8mb4,utf8&readTimeout=30s&writeTimeout=30s",
				DBCheckURL:             "http://db.example.com:3306",
				ReadReplicasURL:        "user:pass@tcp(db-ro.example.com:3306)/mattermost?charset=utf8mb4,utf8&readTimeout=30s&writeTimeout=30s",
				DataSourceURL:          "user:pass@tcp(db.example.com:3306)/mattermost?charset=utf8mb4,utf8&readTimeout=30s&writeTimeout=30s",
			},
			disableDBCheck: false,
			expectedSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysql-secret",
				},
				StringData: map[string]string{
					"DB_CONNECTION_STRING":              "mysql://user:pass@tcp(db.example.com:3306)/mattermost?charset=utf8mb4,utf8&readTimeout=30s&writeTimeout=30s",
					"MM_SQLSETTINGS_DATASOURCEREPLICAS": "user:pass@tcp(db-ro.example.com:3306)/mattermost?charset=utf8mb4,utf8&readTimeout=30s&writeTimeout=30s",
					"DB_CONNECTION_CHECK_URL":           "http://db.example.com:3306",
					"MM_SQLSETTINGS_DATASOURCE":         "user:pass@tcp(db.example.com:3306)/mattermost?charset=utf8mb4,utf8&readTimeout=30s&writeTimeout=30s",
				},
			},
		},
		{
			description: "MySQL secret without datasource URL",
			installationSecret: InstallationDBSecret{
				InstallationSecretName: "mysql-secret-no-datasource",
				ConnectionString:       "mysql://user:pass@tcp(db.example.com:3306)/mattermost?charset=utf8mb4,utf8&readTimeout=30s&writeTimeout=30s",
				ReadReplicasURL:        "user:pass@tcp(db-ro.example.com:3306)/mattermost?charset=utf8mb4,utf8&readTimeout=30s&writeTimeout=30s",
				DataSourceURL:          "", // Empty datasource URL
			},
			disableDBCheck: false,
			expectedSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysql-secret-no-datasource",
				},
				StringData: map[string]string{
					"DB_CONNECTION_STRING":              "mysql://user:pass@tcp(db.example.com:3306)/mattermost?charset=utf8mb4,utf8&readTimeout=30s&writeTimeout=30s",
					"MM_SQLSETTINGS_DATASOURCEREPLICAS": "user:pass@tcp(db-ro.example.com:3306)/mattermost?charset=utf8mb4,utf8&readTimeout=30s&writeTimeout=30s",
				},
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			k8sSecret := testCase.installationSecret.ToK8sSecret(testCase.disableDBCheck)
			assert.Equal(t, testCase.expectedSecret, k8sSecret)
		})
	}
}
