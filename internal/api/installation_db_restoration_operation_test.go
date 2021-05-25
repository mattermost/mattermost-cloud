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

func TestTriggerInstallationDBRestoration(t *testing.T) {
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
			DNS:       "dns1.example.com",
			Database:  model.InstallationDatabaseMultiTenantRDSPostgres,
			Filestore: model.InstallationFilestoreBifrost,
		})
	require.NoError(t, err)
	backup1 := &model.InstallationBackup{InstallationID: installation1.ID, State: model.InstallationBackupStateBackupSucceeded}
	err = sqlStore.CreateInstallationBackup(backup1)
	require.NoError(t, err)

	t.Run("fail for not hibernated installation1", func(t *testing.T) {
		_, err = client.RestoreInstallationDatabase(installation1.ID, backup1.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})
	t.Run("fail for restoring installation1", func(t *testing.T) {
		installation1.State = model.InstallationStateDBRestorationInProgress
		err = sqlStore.UpdateInstallation(installation1.Installation)
		require.NoError(t, err)

		_, err = client.RestoreInstallationDatabase(installation1.ID, backup1.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})

	installation1.State = model.InstallationStateHibernating
	err = sqlStore.UpdateInstallation(installation1.Installation)
	require.NoError(t, err)

	restorationOp, err := client.RestoreInstallationDatabase(installation1.ID, backup1.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, restorationOp.ID)

	assert.Equal(t, model.InstallationDBRestorationStateRequested, restorationOp.State)
	assert.Equal(t, installation1.ID, restorationOp.InstallationID)

	fetchedInstallation, err := sqlStore.GetInstallation(installation1.ID, false, false)
	require.NoError(t, err)
	assert.Equal(t, model.InstallationStateDBRestorationInProgress, fetchedInstallation.State)
}

func TestGetInstallationDBRestorationOperations(t *testing.T) {
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

	restorationOperations := []*model.InstallationDBRestorationOperation{
		{
			InstallationID: installation1.ID,
			State:          model.InstallationDBRestorationStateRequested,
		},
		{
			InstallationID: installation1.ID,
			State:          model.InstallationDBRestorationStateFailed,
		},
		{
			InstallationID: installation2.ID,
			State:          model.InstallationDBRestorationStateRequested,
		},
		{
			InstallationID:        installation2.ID,
			State:                 model.InstallationDBRestorationStateRequested,
			ClusterInstallationID: "ci1",
		},
		{
			InstallationID:        installation2.ID,
			State:                 model.InstallationDBRestorationStateSucceeded,
			ClusterInstallationID: "ci1",
		},
	}

	for i := range restorationOperations {
		err := sqlStore.CreateInstallationDBRestorationOperation(restorationOperations[i])
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond)
	}

	for _, testCase := range []struct {
		description string
		filter      model.GetInstallationDBRestorationOperationsRequest
		found       []*model.InstallationDBRestorationOperation
	}{
		{
			description: "all not deleted",
			filter:      model.GetInstallationDBRestorationOperationsRequest{Paging: model.AllPagesNotDeleted()},
			found:       restorationOperations,
		},
		{
			description: "1 per page",
			filter:      model.GetInstallationDBRestorationOperationsRequest{Paging: model.Paging{PerPage: 1}},
			found:       []*model.InstallationDBRestorationOperation{restorationOperations[4]},
		},
		{
			description: "2nd page",
			filter:      model.GetInstallationDBRestorationOperationsRequest{Paging: model.Paging{PerPage: 1, Page: 1}},
			found:       []*model.InstallationDBRestorationOperation{restorationOperations[3]},
		},
		{
			description: "filter by installation ID",
			filter:      model.GetInstallationDBRestorationOperationsRequest{Paging: model.AllPagesNotDeleted(), InstallationID: installation1.ID},
			found:       []*model.InstallationDBRestorationOperation{restorationOperations[0], restorationOperations[1]},
		},
		{
			description: "filter by cluster installation ID",
			filter:      model.GetInstallationDBRestorationOperationsRequest{Paging: model.AllPagesNotDeleted(), ClusterInstallationID: "ci1"},
			found:       []*model.InstallationDBRestorationOperation{restorationOperations[3], restorationOperations[4]},
		},
		{
			description: "filter by state",
			filter:      model.GetInstallationDBRestorationOperationsRequest{Paging: model.AllPagesNotDeleted(), State: string(model.InstallationDBRestorationStateRequested)},
			found:       []*model.InstallationDBRestorationOperation{restorationOperations[0], restorationOperations[2], restorationOperations[3]},
		},
		{
			description: "no results",
			filter:      model.GetInstallationDBRestorationOperationsRequest{Paging: model.AllPagesNotDeleted(), InstallationID: "no-existent"},
			found:       []*model.InstallationDBRestorationOperation{},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {

			backups, err := client.GetInstallationDBRestorationOperations(&testCase.filter)
			require.NoError(t, err)
			require.Equal(t, len(testCase.found), len(backups))

			for i := 0; i < len(testCase.found); i++ {
				assert.Equal(t, testCase.found[i], backups[len(testCase.found)-1-i])
			}
		})
	}
}

func TestGetInstallationDBRestorationOperation(t *testing.T) {
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

	restorationOp := &model.InstallationDBRestorationOperation{
		BackupID:       "backup",
		InstallationID: "installation",
		State:          model.InstallationDBRestorationStateInProgress,
	}
	err := sqlStore.CreateInstallationDBRestorationOperation(restorationOp)
	require.NoError(t, err)

	fetchedOp, err := client.GetInstallationDBRestoration(restorationOp.ID)
	require.NoError(t, err)
	assert.Equal(t, restorationOp, fetchedOp)

	t.Run("return 404 if operation not found", func(t *testing.T) {
		_, err = client.GetInstallationDBRestoration("not-real")
		require.EqualError(t, err, "failed with status code 404")
	})
}
