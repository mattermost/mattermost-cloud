// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriggerInstallationRestoration(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := setupHibernatingInstallation(t, sqlStore)

	backup := &model.InstallationBackup{
		InstallationID: installation.ID,
	}
	err := sqlStore.CreateInstallationBackup(backup)
	require.NoError(t, err)

	restorationOp, err := sqlStore.TriggerInstallationRestoration(installation, backup)
	require.NoError(t, err)
	assert.Equal(t, installation.ID, restorationOp.InstallationID)
	assert.Equal(t, backup.ID, restorationOp.BackupID)

	fetchOp, err := sqlStore.GetInstallationDBRestorationOperation(restorationOp.ID)
	require.NoError(t, err)
	assert.Equal(t, restorationOp, fetchOp)

	installation, err = sqlStore.GetInstallation(installation.ID, false, false)
	require.NoError(t, err)
	assert.Equal(t, model.InstallationStateDBRestorationInProgress, installation.State)
}

func TestInstallationDBRestoration(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := setupHibernatingInstallation(t, sqlStore)

	dbRestoration := &model.InstallationDBRestorationOperation{
		InstallationID:        installation.ID,
		BackupID:              "test",
		State:                 model.InstallationDBRestorationStateRequested,
		ClusterInstallationID: "",
		CompleteAt:            0,
	}

	err := sqlStore.CreateInstallationDBRestorationOperation(dbRestoration)
	require.NoError(t, err)
	assert.NotEmpty(t, dbRestoration.ID)

	fetchedRestoration, err := sqlStore.GetInstallationDBRestorationOperation(dbRestoration.ID)
	require.NoError(t, err)
	assert.Equal(t, dbRestoration, fetchedRestoration)

	t.Run("unknown restoration", func(t *testing.T) {
		fetchedRestoration, err = sqlStore.GetInstallationDBRestorationOperation("unknown")
		require.NoError(t, err)
		assert.Nil(t, fetchedRestoration)
	})
}

func TestGetInstallationDBRestorations(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation1 := setupHibernatingInstallation(t, sqlStore)
	installation2 := setupHibernatingInstallation(t, sqlStore)
	clusterInstallation := &model.ClusterInstallation{
		InstallationID: installation1.ID,
	}
	err := sqlStore.CreateClusterInstallation(clusterInstallation)
	require.NoError(t, err)

	dbRestorations := []*model.InstallationDBRestorationOperation{
		{InstallationID: installation1.ID, State: model.InstallationDBRestorationStateRequested, ClusterInstallationID: clusterInstallation.ID},
		{InstallationID: installation1.ID, State: model.InstallationDBRestorationStateInProgress, ClusterInstallationID: clusterInstallation.ID},
		{InstallationID: installation1.ID, State: model.InstallationDBRestorationStateFailed},
		{InstallationID: installation2.ID, State: model.InstallationDBRestorationStateRequested},
		{InstallationID: installation2.ID, State: model.InstallationDBRestorationStateInProgress},
	}

	for i := range dbRestorations {
		err := sqlStore.CreateInstallationDBRestorationOperation(dbRestorations[i])
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond) // Ensure RequestAt is different for all installations.
	}

	for _, testCase := range []struct {
		description string
		filter      *model.InstallationDBRestorationFilter
		fetchedIds  []string
	}{
		{
			description: "fetch all",
			filter:      &model.InstallationDBRestorationFilter{Paging: model.AllPagesNotDeleted()},
			fetchedIds:  []string{dbRestorations[4].ID, dbRestorations[3].ID, dbRestorations[2].ID, dbRestorations[1].ID, dbRestorations[0].ID},
		},
		{
			description: "fetch all for installation 1",
			filter:      &model.InstallationDBRestorationFilter{InstallationID: installation1.ID, Paging: model.AllPagesNotDeleted()},
			fetchedIds:  []string{dbRestorations[2].ID, dbRestorations[1].ID, dbRestorations[0].ID},
		},
		{
			description: "fetch all for cluster installation ",
			filter:      &model.InstallationDBRestorationFilter{ClusterInstallationID: clusterInstallation.ID, Paging: model.AllPagesNotDeleted()},
			fetchedIds:  []string{dbRestorations[1].ID, dbRestorations[0].ID},
		},
		{
			description: "fetch requested installations",
			filter:      &model.InstallationDBRestorationFilter{States: []model.InstallationDBRestorationState{model.InstallationDBRestorationStateRequested}, Paging: model.AllPagesNotDeleted()},
			fetchedIds:  []string{dbRestorations[3].ID, dbRestorations[0].ID},
		},
		{
			description: "fetch with IDs",
			filter:      &model.InstallationDBRestorationFilter{IDs: []string{dbRestorations[0].ID, dbRestorations[3].ID, dbRestorations[4].ID}, Paging: model.AllPagesNotDeleted()},
			fetchedIds:  []string{dbRestorations[4].ID, dbRestorations[3].ID, dbRestorations[0].ID},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			fetchedBackups, err := sqlStore.GetInstallationDBRestorationOperations(testCase.filter)
			require.NoError(t, err)
			assert.Equal(t, len(testCase.fetchedIds), len(fetchedBackups))

			for i, b := range fetchedBackups {
				assert.Equal(t, testCase.fetchedIds[i], b.ID)
			}
		})
	}
}

func TestGetUnlockedInstallationDBRestorationsPendingWork(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := setupHibernatingInstallation(t, sqlStore)

	dbRestoration1 := &model.InstallationDBRestorationOperation{
		InstallationID: installation.ID,
		State:          model.InstallationDBRestorationStateRequested,
	}

	err := sqlStore.CreateInstallationDBRestorationOperation(dbRestoration1)
	require.NoError(t, err)
	assert.NotEmpty(t, dbRestoration1.ID)

	dbRestoration2 := &model.InstallationDBRestorationOperation{
		InstallationID: installation.ID,
		State:          model.InstallationDBRestorationStateSucceeded,
	}

	err = sqlStore.CreateInstallationDBRestorationOperation(dbRestoration2)
	require.NoError(t, err)
	assert.NotEmpty(t, dbRestoration1.ID)

	backupsMeta, err := sqlStore.GetUnlockedInstallationDBRestorationOperationsPendingWork()
	require.NoError(t, err)
	assert.Equal(t, 1, len(backupsMeta))
	assert.Equal(t, dbRestoration1.ID, backupsMeta[0].ID)

	locaked, err := sqlStore.LockInstallationDBRestorationOperation(dbRestoration1.ID, "abc")
	require.NoError(t, err)
	assert.True(t, locaked)

	backupsMeta, err = sqlStore.GetUnlockedInstallationDBRestorationOperationsPendingWork()
	require.NoError(t, err)
	assert.Equal(t, 0, len(backupsMeta))
}

func TestUpdateInstallationDBRestoration(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := setupHibernatingInstallation(t, sqlStore)

	dbRestoration := &model.InstallationDBRestorationOperation{
		InstallationID: installation.ID,
		State:          model.InstallationDBRestorationStateRequested,
	}

	err := sqlStore.CreateInstallationDBRestorationOperation(dbRestoration)
	require.NoError(t, err)
	assert.NotEmpty(t, dbRestoration.ID)

	t.Run("update state only", func(t *testing.T) {
		dbRestoration.State = model.InstallationDBRestorationStateSucceeded
		dbRestoration.CompleteAt = -1

		err2 := sqlStore.UpdateInstallationDBRestorationOperationState(dbRestoration)
		require.NoError(t, err2)

		fetched, err3 := sqlStore.GetInstallationDBRestorationOperation(dbRestoration.ID)
		require.NoError(t, err3)
		assert.Equal(t, model.InstallationDBRestorationStateSucceeded, fetched.State)
		assert.Equal(t, int64(0), fetched.CompleteAt)      // Assert complete time not updated
		assert.Equal(t, "", fetched.ClusterInstallationID) // Assert CI ID not updated
	})

	t.Run("full update", func(t *testing.T) {
		dbRestoration.ClusterInstallationID = "test"
		dbRestoration.CompleteAt = 100
		dbRestoration.State = model.InstallationDBRestorationStateFailed
		err2 := sqlStore.UpdateInstallationDBRestorationOperation(dbRestoration)
		require.NoError(t, err2)

		fetched, err3 := sqlStore.GetInstallationDBRestorationOperation(dbRestoration.ID)
		require.NoError(t, err3)
		assert.Equal(t, model.InstallationDBRestorationStateFailed, fetched.State)
		assert.Equal(t, "test", fetched.ClusterInstallationID)
		assert.Equal(t, int64(100), fetched.CompleteAt)
	})
}

func TestDeleteInstallationDBRestoration(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	dbRestoration := &model.InstallationDBRestorationOperation{
		InstallationID: "installation",
		State:          model.InstallationDBRestorationStateRequested,
	}

	err := sqlStore.CreateInstallationDBRestorationOperation(dbRestoration)
	require.NoError(t, err)
	assert.NotEmpty(t, dbRestoration.ID)

	err = sqlStore.DeleteInstallationDBRestorationOperation(dbRestoration.ID)
	require.NoError(t, err)

	operation, err := sqlStore.GetInstallationDBRestorationOperation(dbRestoration.ID)
	require.NoError(t, err)
	assert.True(t, operation.DeleteAt > 0)
}
