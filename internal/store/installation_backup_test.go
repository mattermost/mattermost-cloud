// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"fmt"
	"testing"
	"time"

	"github.com/pborman/uuid"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsInstallationBackupRunning(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := setupStableInstallation(t, sqlStore)

	running, err := sqlStore.IsInstallationBackupRunning(installation.ID)
	require.NoError(t, err)
	require.False(t, running)

	backup := &model.InstallationBackup{
		InstallationID: installation.ID,
		State:          model.InstallationBackupStateBackupRequested,
	}

	err = sqlStore.CreateInstallationBackup(backup)
	require.NoError(t, err)

	running, err = sqlStore.IsInstallationBackupRunning(installation.ID)
	require.NoError(t, err)
	require.True(t, running)
}

func TestIsInstallationBackupBeingUsed(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := setupStableInstallation(t, sqlStore)

	// Create restoration and migration operations not associated with backup.
	notConnectedRestoration := &model.InstallationDBRestorationOperation{
		InstallationID: installation.ID,
		State:          model.InstallationStateDBRestorationInProgress,
	}
	err := sqlStore.CreateInstallationDBRestorationOperation(notConnectedRestoration)
	require.NoError(t, err)
	notConnectedMigration := &model.InstallationDBMigrationOperation{
		InstallationID: installation.ID,
		State:          model.InstallationStateDBRestorationInProgress,
	}
	err = sqlStore.CreateInstallationDBMigrationOperation(notConnectedMigration)
	require.NoError(t, err)

	backup := &model.InstallationBackup{
		InstallationID: installation.ID,
		State:          model.InstallationBackupStateBackupRequested,
	}
	err = sqlStore.CreateInstallationBackup(backup)
	require.NoError(t, err)

	isUsed, err := sqlStore.IsInstallationBackupBeingUsed(backup.ID)
	require.NoError(t, err)
	require.False(t, isUsed)

	// Restoration in progress.
	restorationOp := &model.InstallationDBRestorationOperation{
		InstallationID: installation.ID,
		BackupID:       backup.ID,
		State:          model.InstallationDBRestorationStateRequested,
	}
	err = sqlStore.CreateInstallationDBRestorationOperation(restorationOp)
	require.NoError(t, err)

	isUsed, err = sqlStore.IsInstallationBackupBeingUsed(backup.ID)
	require.NoError(t, err)
	require.True(t, isUsed)

	restorationOp.State = model.InstallationDBRestorationStateSucceeded
	err = sqlStore.UpdateInstallationDBRestorationOperation(restorationOp)
	require.NoError(t, err)

	isUsed, err = sqlStore.IsInstallationBackupBeingUsed(backup.ID)
	require.NoError(t, err)
	require.False(t, isUsed)

	// Migration in progress.
	migrationOp := &model.InstallationDBMigrationOperation{
		InstallationID: installation.ID,
		BackupID:       backup.ID,
		State:          model.InstallationDBMigrationStateRefreshSecrets,
	}
	err = sqlStore.CreateInstallationDBMigrationOperation(migrationOp)
	require.NoError(t, err)
	isUsed, err = sqlStore.IsInstallationBackupBeingUsed(backup.ID)
	require.NoError(t, err)
	require.True(t, isUsed)

	migrationOp.State = model.InstallationDBMigrationStateFailed
	err = sqlStore.UpdateInstallationDBMigrationOperation(migrationOp)
	require.NoError(t, err)
	isUsed, err = sqlStore.IsInstallationBackupBeingUsed(backup.ID)
	require.NoError(t, err)
	require.False(t, isUsed)
}

func TestCreateInstallationBackup(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := setupStableInstallation(t, sqlStore)

	backup := &model.InstallationBackup{
		InstallationID: installation.ID,
		State:          model.InstallationBackupStateBackupRequested,
	}

	err := sqlStore.CreateInstallationBackup(backup)
	require.NoError(t, err)
	assert.NotEmpty(t, backup.ID)
}

func TestGetInstallationBackup(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation1 := setupStableInstallation(t, sqlStore)
	installation2 := setupStableInstallation(t, sqlStore)

	backup1 := &model.InstallationBackup{
		InstallationID: installation1.ID,
		State:          model.InstallationBackupStateBackupRequested,
	}
	err := sqlStore.CreateInstallationBackup(backup1)
	require.NoError(t, err)

	backup2 := &model.InstallationBackup{
		InstallationID: installation2.ID,
		State:          model.InstallationBackupStateBackupRequested,
	}
	err = sqlStore.CreateInstallationBackup(backup2)
	require.NoError(t, err)

	fetchedMeta, err := sqlStore.GetInstallationBackup(backup1.ID)
	require.NoError(t, err)
	assert.Equal(t, backup1, fetchedMeta)

	t.Run("backup not found", func(t *testing.T) {
		fetchedMeta, err = sqlStore.GetInstallationBackup("non-existent")
		require.NoError(t, err)
		assert.Nil(t, fetchedMeta)
	})
}

func TestGetInstallationBackupsMetadata(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation1 := setupStableInstallation(t, sqlStore)
	installation2 := setupStableInstallation(t, sqlStore)
	clusterInstallation := &model.ClusterInstallation{
		InstallationID: installation1.ID,
	}
	err := sqlStore.CreateClusterInstallation(clusterInstallation)
	require.NoError(t, err)

	backupsMeta := []*model.InstallationBackup{
		{InstallationID: installation1.ID, State: model.InstallationBackupStateBackupRequested, ClusterInstallationID: clusterInstallation.ID},
		{InstallationID: installation1.ID, State: model.InstallationBackupStateBackupInProgress, ClusterInstallationID: clusterInstallation.ID},
		{InstallationID: installation1.ID, State: model.InstallationBackupStateBackupFailed},
		{InstallationID: installation2.ID, State: model.InstallationBackupStateBackupRequested},
		{InstallationID: installation2.ID, State: model.InstallationBackupStateBackupInProgress},
	}

	for i := range backupsMeta {
		err := sqlStore.CreateInstallationBackup(backupsMeta[i])
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond) // Ensure RequestAt is different for all installations.
	}

	err = sqlStore.DeleteInstallationBackup(backupsMeta[2].ID)
	require.NoError(t, err)

	for _, testCase := range []struct {
		description string
		filter      *model.InstallationBackupFilter
		fetchedIds  []string
	}{
		{
			description: "fetch all not deleted",
			filter:      &model.InstallationBackupFilter{Paging: model.AllPagesNotDeleted()},
			fetchedIds:  []string{backupsMeta[4].ID, backupsMeta[3].ID, backupsMeta[1].ID, backupsMeta[0].ID},
		},
		{
			description: "fetch all for installation 1",
			filter:      &model.InstallationBackupFilter{InstallationID: installation1.ID, Paging: model.AllPagesWithDeleted()},
			fetchedIds:  []string{backupsMeta[2].ID, backupsMeta[1].ID, backupsMeta[0].ID},
		},
		{
			description: "fetch all for cluster installation ",
			filter:      &model.InstallationBackupFilter{ClusterInstallationID: clusterInstallation.ID, Paging: model.AllPagesNotDeleted()},
			fetchedIds:  []string{backupsMeta[1].ID, backupsMeta[0].ID},
		},
		{
			description: "fetch for installation 1 without deleted",
			filter:      &model.InstallationBackupFilter{InstallationID: installation1.ID, Paging: model.AllPagesNotDeleted()},
			fetchedIds:  []string{backupsMeta[1].ID, backupsMeta[0].ID},
		},
		{
			description: "fetch requested installations",
			filter:      &model.InstallationBackupFilter{States: []model.InstallationBackupState{model.InstallationBackupStateBackupRequested}, Paging: model.AllPagesNotDeleted()},
			fetchedIds:  []string{backupsMeta[3].ID, backupsMeta[0].ID},
		},
		{
			description: "fetch with IDs",
			filter:      &model.InstallationBackupFilter{IDs: []string{backupsMeta[0].ID, backupsMeta[3].ID, backupsMeta[4].ID}, Paging: model.AllPagesNotDeleted()},
			fetchedIds:  []string{backupsMeta[4].ID, backupsMeta[3].ID, backupsMeta[0].ID},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			fetchedBackups, err := sqlStore.GetInstallationBackups(testCase.filter)
			require.NoError(t, err)
			assert.Equal(t, len(testCase.fetchedIds), len(fetchedBackups))

			for i, b := range fetchedBackups {
				assert.Equal(t, testCase.fetchedIds[i], b.ID)
			}
		})
	}
}

func TestGetUnlockedInstallationBackupPendingWork(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := setupStableInstallation(t, sqlStore)

	backup1 := &model.InstallationBackup{
		InstallationID: installation.ID,
		State:          model.InstallationBackupStateBackupRequested,
	}

	err := sqlStore.CreateInstallationBackup(backup1)
	require.NoError(t, err)
	assert.NotEmpty(t, backup1.ID)

	backup2 := &model.InstallationBackup{
		InstallationID: installation.ID,
		State:          model.InstallationBackupStateBackupSucceeded,
	}

	err = sqlStore.CreateInstallationBackup(backup2)
	require.NoError(t, err)
	assert.NotEmpty(t, backup1.ID)

	backupsMeta, err := sqlStore.GetUnlockedInstallationBackupPendingWork()
	require.NoError(t, err)
	assert.Equal(t, 1, len(backupsMeta))
	assert.Equal(t, backup1.ID, backupsMeta[0].ID)

	locaked, err := sqlStore.LockInstallationBackup(backup1.ID, "abc")
	require.NoError(t, err)
	assert.True(t, locaked)

	backupsMeta, err = sqlStore.GetUnlockedInstallationBackupPendingWork()
	require.NoError(t, err)
	assert.Equal(t, 0, len(backupsMeta))
}

func TestUpdateInstallationBackup(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := setupStableInstallation(t, sqlStore)

	backup := &model.InstallationBackup{
		InstallationID: installation.ID,
		State:          model.InstallationBackupStateBackupRequested,
	}

	err := sqlStore.CreateInstallationBackup(backup)
	require.NoError(t, err)
	assert.NotEmpty(t, backup.ID)

	t.Run("update state only", func(t *testing.T) {
		backup.State = model.InstallationBackupStateBackupSucceeded
		backup.StartAt = -1

		err = sqlStore.UpdateInstallationBackupState(backup)
		require.NoError(t, err)

		fetched, err := sqlStore.GetInstallationBackup(backup.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationBackupStateBackupSucceeded, fetched.State)
		assert.Equal(t, int64(0), fetched.StartAt)         // Assert start time not updated
		assert.Equal(t, "", fetched.ClusterInstallationID) // Assert CI ID not updated
	})

	t.Run("update data residency only", func(t *testing.T) {
		updatedResidence := &model.S3DataResidence{URL: "s3.amazon.com"}
		clusterInstallationID := "cluster-installation-1"

		backup.StartAt = -1
		backup.DataResidence = updatedResidence
		backup.ClusterInstallationID = clusterInstallationID

		err = sqlStore.UpdateInstallationBackupSchedulingData(backup)
		require.NoError(t, err)

		fetched, err := sqlStore.GetInstallationBackup(backup.ID)
		require.NoError(t, err)
		assert.Equal(t, updatedResidence, fetched.DataResidence)
		assert.Equal(t, clusterInstallationID, fetched.ClusterInstallationID)
		assert.Equal(t, int64(0), fetched.StartAt) // Assert start time not updated
	})

	t.Run("update start time", func(t *testing.T) {
		var startTime int64 = 10000
		originalCIId := backup.ClusterInstallationID

		backup.StartAt = startTime
		backup.ClusterInstallationID = "modified-ci-id"

		err = sqlStore.UpdateInstallationBackupStartTime(backup)
		require.NoError(t, err)

		fetched, err := sqlStore.GetInstallationBackup(backup.ID)
		require.NoError(t, err)
		assert.Equal(t, startTime, fetched.StartAt)
		assert.Equal(t, originalCIId, fetched.ClusterInstallationID) // Assert ClusterInstallationID not updated
	})
}

func setupStableInstallation(t *testing.T, sqlStore *SQLStore) *model.Installation {
	return setupInstallation(t, sqlStore, model.InstallationStateStable)
}

func setupHibernatingInstallation(t *testing.T, sqlStore *SQLStore) *model.Installation {
	return setupInstallation(t, sqlStore, model.InstallationStateHibernating)
}

func setupInstallation(t *testing.T, sqlStore *SQLStore, state string) *model.Installation {
	installation := &model.Installation{
		State: state,
		DNS:   fmt.Sprintf("dns-%s", uuid.NewRandom().String()[:6]),
	}

	err := sqlStore.CreateInstallation(installation, nil)
	require.NoError(t, err)

	return installation
}
