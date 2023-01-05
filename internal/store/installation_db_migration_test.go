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

func TestTriggerInstallationDBMigration(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := setupHibernatingInstallation(t, sqlStore)

	dbMigrationOp := &model.InstallationDBMigrationOperation{
		SourceDatabase:      "source",
		DestinationDatabase: "destination",
	}

	migrationOp, err := sqlStore.TriggerInstallationDBMigration(dbMigrationOp, installation)
	require.NoError(t, err)
	assert.Equal(t, installation.ID, migrationOp.InstallationID)
	assert.Equal(t, model.InstallationDBMigrationStateRequested, migrationOp.State)

	fetchOp, err := sqlStore.GetInstallationDBMigrationOperation(migrationOp.ID)
	require.NoError(t, err)
	assert.Equal(t, migrationOp, fetchOp)

	installation, err = sqlStore.GetInstallation(installation.ID, false, false)
	require.NoError(t, err)
	assert.Equal(t, model.InstallationStateDBMigrationInProgress, installation.State)
}

func TestTriggerInstallationDBMigrationRollback(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := setupHibernatingInstallation(t, sqlStore)

	dbMigrationOp := &model.InstallationDBMigrationOperation{
		SourceDatabase:      "source",
		DestinationDatabase: "destination",
		InstallationID:      installation.ID,
	}
	err := sqlStore.CreateInstallationDBMigrationOperation(dbMigrationOp)
	require.NoError(t, err)

	err = sqlStore.TriggerInstallationDBMigrationRollback(dbMigrationOp, installation)
	require.NoError(t, err)
	assert.Equal(t, model.InstallationDBMigrationStateRollbackRequested, dbMigrationOp.State)

	fetchOp, err := sqlStore.GetInstallationDBMigrationOperation(dbMigrationOp.ID)
	require.NoError(t, err)
	assert.Equal(t, dbMigrationOp, fetchOp)

	installation, err = sqlStore.GetInstallation(installation.ID, false, false)
	require.NoError(t, err)
	assert.Equal(t, model.InstallationStateDBMigrationRollbackInProgress, installation.State)
}

func TestInstallationDBMigrationOperation(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := setupHibernatingInstallation(t, sqlStore)

	dbMigrationOp := &model.InstallationDBMigrationOperation{
		InstallationID:         installation.ID,
		SourceMultiTenant:      &model.MultiTenantDBMigrationData{DatabaseID: "abcd"},
		DestinationMultiTenant: &model.MultiTenantDBMigrationData{DatabaseID: "efgh"},
		SourceDatabase:         model.InstallationDatabaseMultiTenantRDSPostgres,
		DestinationDatabase:    model.InstallationDatabaseMultiTenantRDSPostgres,
		BackupID:               "",
		RequestAt:              0,
		State:                  model.InstallationDBMigrationStateRequested,
	}

	err := sqlStore.CreateInstallationDBMigrationOperation(dbMigrationOp)
	require.NoError(t, err)
	assert.NotEmpty(t, dbMigrationOp.ID)

	fetchedRestoration, err := sqlStore.GetInstallationDBMigrationOperation(dbMigrationOp.ID)
	require.NoError(t, err)
	assert.Equal(t, dbMigrationOp, fetchedRestoration)

	t.Run("unknown restoration", func(t *testing.T) {
		fetchedRestoration, err = sqlStore.GetInstallationDBMigrationOperation("unknown")
		require.NoError(t, err)
		assert.Nil(t, fetchedRestoration)
	})
}

func TestGetInstallationDBMigrations(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation1 := setupHibernatingInstallation(t, sqlStore)
	installation2 := setupHibernatingInstallation(t, sqlStore)

	dbMigrations := []*model.InstallationDBMigrationOperation{
		{InstallationID: installation1.ID, State: model.InstallationDBMigrationStateRequested},
		{InstallationID: installation1.ID, State: model.InstallationDBMigrationStateBackupInProgress},
		{InstallationID: installation1.ID, State: model.InstallationDBMigrationStateFailed},
		{InstallationID: installation2.ID, State: model.InstallationDBMigrationStateRequested},
		{InstallationID: installation2.ID, State: model.InstallationDBMigrationStateBackupInProgress},
	}

	for i := range dbMigrations {
		err := sqlStore.CreateInstallationDBMigrationOperation(dbMigrations[i])
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond) // Ensure RequestAt is different for all installations.
	}

	for _, testCase := range []struct {
		description string
		filter      *model.InstallationDBMigrationFilter
		fetchedIds  []string
	}{
		{
			description: "fetch all",
			filter:      &model.InstallationDBMigrationFilter{Paging: model.AllPagesNotDeleted()},
			fetchedIds:  []string{dbMigrations[4].ID, dbMigrations[3].ID, dbMigrations[2].ID, dbMigrations[1].ID, dbMigrations[0].ID},
		},
		{
			description: "fetch all for installation 1",
			filter:      &model.InstallationDBMigrationFilter{InstallationID: installation1.ID, Paging: model.AllPagesNotDeleted()},
			fetchedIds:  []string{dbMigrations[2].ID, dbMigrations[1].ID, dbMigrations[0].ID},
		},
		{
			description: "fetch requested operations",
			filter:      &model.InstallationDBMigrationFilter{States: []model.InstallationDBMigrationOperationState{model.InstallationDBMigrationStateRequested}, Paging: model.AllPagesNotDeleted()},
			fetchedIds:  []string{dbMigrations[3].ID, dbMigrations[0].ID},
		},
		{
			description: "fetch with IDs",
			filter:      &model.InstallationDBMigrationFilter{IDs: []string{dbMigrations[0].ID, dbMigrations[3].ID, dbMigrations[4].ID}, Paging: model.AllPagesNotDeleted()},
			fetchedIds:  []string{dbMigrations[4].ID, dbMigrations[3].ID, dbMigrations[0].ID},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			fetchedBackups, err := sqlStore.GetInstallationDBMigrationOperations(testCase.filter)
			require.NoError(t, err)
			assert.Equal(t, len(testCase.fetchedIds), len(fetchedBackups))

			for i, b := range fetchedBackups {
				assert.Equal(t, testCase.fetchedIds[i], b.ID)
			}
		})
	}
}

func TestGetUnlockedInstallationDBRMigrationsPendingWork(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := setupHibernatingInstallation(t, sqlStore)

	dbMigration1 := &model.InstallationDBMigrationOperation{
		InstallationID: installation.ID,
		State:          model.InstallationDBMigrationStateRequested,
	}

	err := sqlStore.CreateInstallationDBMigrationOperation(dbMigration1)
	require.NoError(t, err)
	assert.NotEmpty(t, dbMigration1.ID)

	dbMigration2 := &model.InstallationDBMigrationOperation{
		InstallationID: installation.ID,
		State:          model.InstallationDBMigrationStateSucceeded,
	}

	err = sqlStore.CreateInstallationDBMigrationOperation(dbMigration2)
	require.NoError(t, err)
	assert.NotEmpty(t, dbMigration1.ID)

	backupsMeta, err := sqlStore.GetUnlockedInstallationDBMigrationOperationsPendingWork()
	require.NoError(t, err)
	assert.Equal(t, 1, len(backupsMeta))
	assert.Equal(t, dbMigration1.ID, backupsMeta[0].ID)

	locaked, err := sqlStore.LockInstallationDBMigrationOperation(dbMigration1.ID, "abc")
	require.NoError(t, err)
	assert.True(t, locaked)

	backupsMeta, err = sqlStore.GetUnlockedInstallationDBMigrationOperationsPendingWork()
	require.NoError(t, err)
	assert.Equal(t, 0, len(backupsMeta))
}

func TestUpdateInstallationDBMigration(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := setupHibernatingInstallation(t, sqlStore)

	dbMigration := &model.InstallationDBMigrationOperation{
		InstallationID: installation.ID,
		State:          model.InstallationDBMigrationStateRequested,
	}

	err := sqlStore.CreateInstallationDBMigrationOperation(dbMigration)
	require.NoError(t, err)
	assert.NotEmpty(t, dbMigration.ID)

	t.Run("update state only", func(t *testing.T) {
		dbMigration.State = model.InstallationDBMigrationStateSucceeded
		dbMigration.CompleteAt = -1
		dbMigration.InstallationDBRestorationOperationID = "test"

		err2 := sqlStore.UpdateInstallationDBMigrationOperationState(dbMigration)
		require.NoError(t, err2)

		fetched, err3 := sqlStore.GetInstallationDBMigrationOperation(dbMigration.ID)
		require.NoError(t, err3)
		assert.Equal(t, model.InstallationDBMigrationStateSucceeded, fetched.State)
		assert.Equal(t, int64(0), fetched.CompleteAt)                     // Assert complete time not updated
		assert.Equal(t, "", fetched.InstallationDBRestorationOperationID) // Assert ID not updated
	})

	t.Run("full update", func(t *testing.T) {
		dbMigration.InstallationDBRestorationOperationID = "test"
		dbMigration.CompleteAt = 100
		dbMigration.State = model.InstallationDBMigrationStateFailed
		err = sqlStore.UpdateInstallationDBMigrationOperation(dbMigration)
		require.NoError(t, err)

		fetched, err := sqlStore.GetInstallationDBMigrationOperation(dbMigration.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationDBMigrationStateFailed, fetched.State)
		assert.Equal(t, "test", fetched.InstallationDBRestorationOperationID)
		assert.Equal(t, int64(100), fetched.CompleteAt)
	})
}

func TestDeleteInstallationDBMigration(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	dbMigration := &model.InstallationDBMigrationOperation{
		InstallationID: "installation",
		State:          model.InstallationDBMigrationStateSucceeded,
	}

	err := sqlStore.CreateInstallationDBMigrationOperation(dbMigration)
	require.NoError(t, err)
	assert.NotEmpty(t, dbMigration.ID)

	err = sqlStore.DeleteInstallationDBMigrationOperation(dbMigration.ID)
	require.NoError(t, err)

	operation, err := sqlStore.GetInstallationDBMigrationOperation(dbMigration.ID)
	require.NoError(t, err)
	assert.True(t, operation.DeleteAt > 0)
}
