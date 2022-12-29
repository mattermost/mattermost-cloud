// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/clusterdictionary"
	"github.com/mattermost/mattermost-cloud/model"
)

func TestCluster(t *testing.T) {
	suite := SetupTest(t)
	t.Cleanup(func() {
		CleanupTest(t, suite)
	})

	t.Run("test cluster and installation create and destroy", func(t *testing.T) {
		// cluster create
		clusterRequest := &model.CreateClusterRequest{
			AllowInstallations: true,
		}
		if err := clusterdictionary.ApplyToCreateClusterRequest(clusterdictionary.SizeAlefDev, clusterRequest); err != nil {
			t.Error(err)
		}
		cluster, err := suite.Client().CreateCluster(clusterRequest)
		if err != nil {
			t.Error(err)
		}

		err = suite.WaitForEvent(
			model.TypeCluster,
			cluster.ID,
			model.ClusterStateStable,
			model.ClusterStateCreationFailed,
			time.Minute*30,
		)
		if err != nil {
			t.Error(err)
		}

		// installation create
		name := createUniqueName()
		installationRequest := &model.CreateInstallationRequest{
			OwnerID: testIdentifier,
			Name:    name,
			DNSNames: []string{
				fmt.Sprintf("%s.dev.cloud.mattermost.com", name),
			},
		}
		installation, err := suite.Client().CreateInstallation(installationRequest)
		if err != nil {
			t.Error(err)
		}

		err = suite.WaitForEvent(
			model.TypeInstallation,
			installation.ID,
			model.InstallationStateStable,
			model.InstallationStateCreationFailed,
			time.Minute*10,
		)
		if err != nil {
			t.Error(err)
		}

		// installation dekete
		err = suite.Client().DeleteInstallation(installation.ID)
		if err != nil {
			t.Error(err)
		}

		err = suite.WaitForEvent(
			model.TypeInstallation,
			installation.ID,
			model.InstallationStateDeleted,
			model.InstallationStateDeletionFailed,
			time.Minute*10,
		)
		if err != nil {
			t.Error(err)
		}

		// cluster delete
		if err := suite.Client().DeleteCluster(cluster.ID); err != nil {
			t.Error(err)
		}

		err = suite.WaitForEvent(
			model.TypeCluster,
			cluster.ID,
			model.ClusterStateDeleted,
			model.ClusterStateDeletionFailed,
			time.Minute*30,
		)
		if err != nil {
			t.Error(err)
		}
	})
}
