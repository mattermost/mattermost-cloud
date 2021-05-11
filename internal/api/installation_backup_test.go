// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api_test

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/internal/api"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/internal/testutil"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestInstallationBackup(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})

	ts := httptest.NewServer(router)
	client := model.NewClient(ts.URL)
	installation1, err := client.CreateInstallation(
		&model.CreateInstallationRequest{
			OwnerID:   "owner",
			Version:   "version",
			DNS:       "dns1.example.com",
			Affinity:  model.InstallationAffinityMultiTenant,
			Database:  model.InstallationDatabaseMultiTenantRDSPostgres,
			Filestore: model.InstallationFilestoreBifrost,
		})
	require.NoError(t, err)

	t.Run("fail for not hibernated installation1", func(t *testing.T) {
		_, err = client.CreateInstallationBackup(installation1.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})

	installation1.State = model.InstallationStateHibernating
	err = sqlStore.UpdateInstallation(installation1.Installation)
	require.NoError(t, err)

	backup, err := client.CreateInstallationBackup(installation1.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, backup.ID)

	t.Run("fail to request multiple backups for same installation1", func(t *testing.T) {
		_, err = client.CreateInstallationBackup(installation1.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})

	t.Run("can request backup for different installation", func(t *testing.T) {
		installation2, err := client.CreateInstallation(
			&model.CreateInstallationRequest{
				OwnerID:   "owner",
				Version:   "version",
				DNS:       "dns2.example.com",
				Affinity:  model.InstallationAffinityMultiTenant,
				Database:  model.InstallationDatabaseMultiTenantRDSPostgres,
				Filestore: model.InstallationFilestoreBifrost,
			})
		require.NoError(t, err)

		installation2.State = model.InstallationStateHibernating
		err = sqlStore.UpdateInstallation(installation2.Installation)
		require.NoError(t, err)

		backup2, err := client.CreateInstallationBackup(installation2.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, backup2.ID)
	})
}

func TestGetInstallationBackups(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})

	ts := httptest.NewServer(router)
	client := model.NewClient(ts.URL)

	installation1 := testutil.CreateBackupCompatibleInstallation(t, sqlStore)
	installation2 := testutil.CreateBackupCompatibleInstallation(t, sqlStore)

	backup := []*model.InstallationBackup{
		{
			InstallationID: installation1.ID,
			State:          model.InstallationBackupStateBackupRequested,
		},
		{
			InstallationID: installation1.ID,
			State:          model.InstallationBackupStateBackupFailed,
		},
		{
			InstallationID: installation2.ID,
			State:          model.InstallationBackupStateBackupRequested,
		},
		{
			InstallationID:        installation2.ID,
			State:                 model.InstallationBackupStateBackupRequested,
			ClusterInstallationID: "ci1",
		},
		{
			InstallationID:        installation2.ID,
			State:                 model.InstallationBackupStateBackupSucceeded,
			ClusterInstallationID: "ci1",
		},
	}

	for i := range backup {
		err := sqlStore.CreateInstallationBackup(backup[i])
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond)
	}
	deletedMeta := &model.InstallationBackup{InstallationID: "deleted"}
	err := sqlStore.CreateInstallationBackup(deletedMeta)
	require.NoError(t, err)
	err = sqlStore.DeleteInstallationBackup(deletedMeta.ID)
	require.NoError(t, err)
	deletedMeta, err = sqlStore.GetInstallationBackup(deletedMeta.ID)

	for _, testCase := range []struct {
		description string
		filter      model.GetInstallationBackupsRequest
		found       []*model.InstallationBackup
	}{
		{
			description: "all",
			filter:      model.GetInstallationBackupsRequest{Paging: model.AllPagesWithDeleted()},
			found:       append(backup, deletedMeta),
		},
		{
			description: "all not deleted",
			filter:      model.GetInstallationBackupsRequest{Paging: model.AllPagesNotDeleted()},
			found:       backup,
		},
		{
			description: "1 per page",
			filter:      model.GetInstallationBackupsRequest{Paging: model.Paging{PerPage: 1}},
			found:       []*model.InstallationBackup{backup[4]},
		},
		{
			description: "2nd page",
			filter:      model.GetInstallationBackupsRequest{Paging: model.Paging{PerPage: 1, Page: 1}},
			found:       []*model.InstallationBackup{backup[3]},
		},
		{
			description: "filter by installation ID",
			filter:      model.GetInstallationBackupsRequest{Paging: model.AllPagesNotDeleted(), InstallationID: installation1.ID},
			found:       []*model.InstallationBackup{backup[0], backup[1]},
		},
		{
			description: "filter by cluster installation ID",
			filter:      model.GetInstallationBackupsRequest{Paging: model.AllPagesNotDeleted(), ClusterInstallationID: "ci1"},
			found:       []*model.InstallationBackup{backup[3], backup[4]},
		},
		{
			description: "filter by state",
			filter:      model.GetInstallationBackupsRequest{Paging: model.AllPagesNotDeleted(), State: string(model.InstallationBackupStateBackupRequested)},
			found:       []*model.InstallationBackup{backup[0], backup[2], backup[3]},
		},
		{
			description: "no results",
			filter:      model.GetInstallationBackupsRequest{Paging: model.AllPagesNotDeleted(), InstallationID: "no-existent"},
			found:       []*model.InstallationBackup{},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {

			backups, err := client.GetInstallationBackups(&testCase.filter)
			require.NoError(t, err)
			require.Equal(t, len(testCase.found), len(backups))

			for i := 0; i < len(testCase.found); i++ {
				assert.Equal(t, testCase.found[i], backups[len(testCase.found)-1-i])
			}

		})
	}
}

func TestGetInstallationBackup(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})

	ts := httptest.NewServer(router)
	client := model.NewClient(ts.URL)

	installation1 := testutil.CreateBackupCompatibleInstallation(t, sqlStore)

	backup, err := client.CreateInstallationBackup(installation1.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, backup.ID)

	fetchedMeta, err := client.GetInstallationBackup(backup.ID)
	require.NoError(t, err)
	assert.Equal(t, backup, fetchedMeta)

	t.Run("return 404 if backup not found", func(t *testing.T) {
		_, err = client.GetInstallationBackup("not-real")
		require.EqualError(t, err, "failed with status code 404")
	})
}

func TestDeleteInstallationBackup(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})

	ts := httptest.NewServer(router)
	client := model.NewClient(ts.URL)

	installation1 := testutil.CreateBackupCompatibleInstallation(t, sqlStore)

	backup, err := client.CreateInstallationBackup(installation1.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, backup.ID)

	t.Run("unknown backup", func(t *testing.T) {
		err = client.DeleteInstallationBackup(model.NewID())
		require.EqualError(t, err, "failed with status code 404")
	})

	t.Run("while locked", func(t *testing.T) {
		_, err = sqlStore.LockInstallationBackup(backup.ID, "locker")
		require.NoError(t, err)
		defer func() {
			_, err = sqlStore.UnlockInstallationBackup(backup.ID, "locker", true)
			require.NoError(t, err)
		}()

		err = client.DeleteInstallationBackup(backup.ID)
		require.EqualError(t, err, "failed with status code 409")
	})

	t.Run("while api-security-locked", func(t *testing.T) {
		err = sqlStore.LockInstallationBackupAPI(backup.ID)
		require.NoError(t, err)
		defer func() {
			err = sqlStore.UnlockInstallationBackupAPI(backup.ID)
			require.NoError(t, err)
		}()

		err = client.DeleteInstallationBackup(backup.ID)
		require.EqualError(t, err, "failed with status code 403")
	})

	t.Run("while restoration is using backup", func(t *testing.T) {
		restorationOp := &model.InstallationDBRestorationOperation{
			BackupID: backup.ID,
			State:    model.InstallationDBRestorationStateInProgress,
		}
		err := sqlStore.CreateInstallationDBRestorationOperation(restorationOp)
		require.NoError(t, err)
		defer func() {
			restorationOp.State = model.InstallationDBRestorationStateFailed
			err = sqlStore.UpdateInstallationDBRestorationOperation(restorationOp)
			assert.NoError(t, err)
		}()

		err = client.DeleteInstallationBackup(backup.ID)
		require.EqualError(t, err, "failed with status code 400")
	})

	err = client.DeleteInstallationBackup(backup.ID)
	require.NoError(t, err)

	fetchedBackup, err := client.GetInstallationBackup(backup.ID)
	require.NoError(t, err)
	assert.Equal(t, model.InstallationBackupStateDeletionRequested, fetchedBackup.State)

	t.Run("cannot deleted already deleted", func(t *testing.T) {
		backup.State = model.InstallationBackupStateDeleted
		err = sqlStore.UpdateInstallationBackupState(backup)
		require.NoError(t, err)

		err = client.DeleteInstallationBackup(backup.ID)
		require.EqualError(t, err, "failed with status code 400")
	})
}

func TestBackupAPILock(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})

	ts := httptest.NewServer(router)
	client := model.NewClient(ts.URL)

	installation1 := testutil.CreateBackupCompatibleInstallation(t, sqlStore)

	backup, err := client.CreateInstallationBackup(installation1.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, backup.ID)

	err = client.LockAPIForBackup(backup.ID)
	require.NoError(t, err)
	fetchedMeta, err := client.GetInstallationBackup(backup.ID)
	require.NoError(t, err)
	assert.True(t, fetchedMeta.APISecurityLock)

	err = client.UnlockAPIForBackup(backup.ID)
	require.NoError(t, err)
	fetchedMeta, err = client.GetInstallationBackup(backup.ID)
	require.NoError(t, err)
	assert.False(t, fetchedMeta.APISecurityLock)
}
