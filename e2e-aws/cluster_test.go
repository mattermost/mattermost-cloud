// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"fmt"
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
)

func TestCluster(t *testing.T) {
	suite := SetupTest(t)
	t.Cleanup(func() {
		CleanupTest(t, suite)
	})

	t.Run("test cluster and installation create and destroy", func(t *testing.T) {
		// cluster create
		// clusterRequest := &model.CreateClusterRequest{
		// 	AllowInstallations: true,
		// }
		// err := clusterdictionary.ApplyToCreateClusterRequest(clusterdictionary.SizeAlefDev, clusterRequest)
		// assert.NoError(t, err)

		// cluster := testClusterCreation(t, suite, clusterRequest)

		t.Run("create simple installation", func(t *testing.T) {
			name := createUniqueName()
			installationRequest := &model.CreateInstallationRequest{
				OwnerID: testIdentifier,
				Name:    name,
				DNSNames: []string{
					fmt.Sprintf("%s.dev.cloud.mattermost.com", name),
				},
			}
			installation := testInstallationCreation(t, suite, installationRequest)
			testInstallationDeletion(t, suite, installation.ID)
		})

		t.Run("create installation with single tenant database (defaults)", func(t *testing.T) {
			name := createUniqueName()
			installationRequest := &model.CreateInstallationRequest{
				OwnerID: testIdentifier,
				Name:    name,
				DNSNames: []string{
					fmt.Sprintf("%s.dev.cloud.mattermost.com", name),
				},
				SingleTenantDatabaseConfig: model.SingleTenantDatabaseRequest{},
				Database:                   model.InstallationDatabaseSingleTenantRDSMySQL,
			}
			installation := testInstallationCreation(t, suite, installationRequest)
			testInstallationDeletion(t, suite, installation.ID)
		})

		t.Run("create installation with aws s3 filestore", func(t *testing.T) {
			name := createUniqueName()
			installationRequest := &model.CreateInstallationRequest{
				OwnerID: testIdentifier,
				Name:    name,
				DNSNames: []string{
					fmt.Sprintf("%s.dev.cloud.mattermost.com", name),
				},
				Filestore: model.InstallationFilestoreAwsS3,
			}
			installation := testInstallationCreation(t, suite, installationRequest)
			testInstallationDeletion(t, suite, installation.ID)
		})

		t.Run("create installation with s3 multitenant filestore", func(t *testing.T) {
			name := createUniqueName()
			installationRequest := &model.CreateInstallationRequest{
				OwnerID: testIdentifier,
				Name:    name,
				DNSNames: []string{
					fmt.Sprintf("%s.dev.cloud.mattermost.com", name),
				},
				Filestore: model.InstallationFilestoreMultiTenantAwsS3,
			}
			installation := testInstallationCreation(t, suite, installationRequest)
			testInstallationDeletion(t, suite, installation.ID)
		})

		t.Run("create installation with bifrost filestore", func(t *testing.T) {
			name := createUniqueName()
			installationRequest := &model.CreateInstallationRequest{
				OwnerID: testIdentifier,
				Name:    name,
				DNSNames: []string{
					fmt.Sprintf("%s.dev.cloud.mattermost.com", name),
				},
				Filestore: model.InstallationFilestoreBifrost,
			}
			installation := testInstallationCreation(t, suite, installationRequest)
			testInstallationDeletion(t, suite, installation.ID)
		})

		// cluster delete
		// testClusterDeletion(t, suite, cluster.ID)
	})
}
