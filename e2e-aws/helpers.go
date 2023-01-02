package main

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
)

func testClusterCreation(t *testing.T, suite *ClusterTestSuite, clusterRequest *model.CreateClusterRequest) *model.ClusterDTO {
	cluster, err := suite.Client().CreateCluster(clusterRequest)
	assert.NoError(t, err)

	err = suite.WaitForEvent(
		model.TypeCluster,
		cluster.ID,
		model.ClusterStateStable,
		model.ClusterStateCreationFailed,
		defaultClusterCreationTimeout,
	)
	assert.NoError(t, err)

	return cluster
}

func testClusterDeletion(t *testing.T, suite *ClusterTestSuite, clusterID string) {
	err := suite.Client().DeleteCluster(clusterID)
	assert.NoError(t, err)

	err = suite.WaitForEvent(
		model.TypeCluster,
		clusterID,
		model.ClusterStateStable,
		model.ClusterStateCreationFailed,
		defaultClusterCreationTimeout,
	)
	assert.NoError(t, err)
}

func testInstallationCreation(t *testing.T, suite *ClusterTestSuite, request *model.CreateInstallationRequest) *model.InstallationDTO {
	installation, err := suite.Client().CreateInstallation(request)
	assert.NoError(t, err)

	err = suite.WaitForEvent(
		model.TypeInstallation,
		installation.ID,
		model.InstallationStateStable,
		model.InstallationStateCreationFailed,
		defaultInstallationCreationTimeout,
	)
	assert.NoError(t, err)

	return installation
}

func testInstallationDeletion(t *testing.T, suite *ClusterTestSuite, installationID string) {
	err := suite.Client().DeleteInstallation(installationID)
	assert.NoError(t, err)

	err = suite.WaitForEvent(
		model.TypeInstallation,
		installationID,
		model.InstallationStateDeleted,
		model.InstallationStateDeletionFailed,
		defaultInstallationDeletionTimeout,
	)
	assert.NoError(t, err)
}
