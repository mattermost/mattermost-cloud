// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api_test

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mocks "github.com/mattermost/mattermost-cloud/internal/mocks/model"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/internal/api"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/internal/testutil"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriggerInstallationDBMigration(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:         sqlStore,
		Supervisor:    &mockSupervisor{},
		EventProducer: testutil.SetupTestEventsProducer(sqlStore, logger),
		Logger:        logger,
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
	installation1.State = model.InstallationStateHibernating
	err = sqlStore.UpdateInstallation(installation1.Installation)
	require.NoError(t, err)

	currentDB := &model.MultitenantDatabase{
		RdsClusterID:  "cluster1",
		VpcID:         "vpc1",
		DatabaseType:  model.DatabaseEngineTypePostgres,
		Installations: model.MultitenantDatabaseInstallations{installation1.ID},
	}
	err = sqlStore.CreateMultitenantDatabase(currentDB)
	require.NoError(t, err)

	destinationDB := &model.MultitenantDatabase{
		RdsClusterID: "cluster2",
		VpcID:        "vpc1",
		DatabaseType: model.DatabaseEngineTypePostgres,
	}
	err = sqlStore.CreateMultitenantDatabase(destinationDB)
	require.NoError(t, err)

	migrationRequest := &model.InstallationDBMigrationRequest{
		InstallationID:         installation1.ID,
		DestinationDatabase:    model.InstallationDatabaseMultiTenantRDSPostgres,
		DestinationMultiTenant: &model.MultiTenantDBMigrationData{DatabaseID: destinationDB.ID},
	}

	migrationOperation, err := client.MigrateInstallationDatabase(migrationRequest)
	require.NoError(t, err)

	assert.Equal(t, model.InstallationDBMigrationStateRequested, migrationOperation.State)
	assert.Equal(t, installation1.ID, migrationOperation.InstallationID)

	installation, err := sqlStore.GetInstallation(installation1.ID, false, false)
	require.NoError(t, err)
	assert.Equal(t, model.InstallationStateDBMigrationInProgress, installation.State)

	t.Run("fail to trigger migration if states is not hibernating", func(t *testing.T) {
		migrationOperation, err = client.MigrateInstallationDatabase(migrationRequest)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})

	installation1.State = model.InstallationStateHibernating
	err = sqlStore.UpdateInstallation(installation1.Installation)
	require.NoError(t, err)

	t.Run("fail to trigger migration if other migration succeeded but not committed", func(t *testing.T) {
		succeededMigration := &model.InstallationDBMigrationOperation{State: model.InstallationDBMigrationStateSucceeded, InstallationID: installation1.ID}
		err = sqlStore.CreateInstallationDBMigrationOperation(succeededMigration)
		require.NoError(t, err)
		defer func() {
			err := sqlStore.DeleteInstallationDBRestorationOperation(succeededMigration.ID)
			assert.NoError(t, err)
		}()

		migrationOperation, err = client.MigrateInstallationDatabase(migrationRequest)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})

	t.Run("fail to trigger migration if destination database not supported", func(t *testing.T) {
		migrationRequest := &model.InstallationDBMigrationRequest{
			InstallationID:      installation1.ID,
			DestinationDatabase: model.InstallationDatabaseMysqlOperator,
		}
		migrationOperation, err = client.MigrateInstallationDatabase(migrationRequest)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})

	t.Run("fail to trigger migration if destination database not found", func(t *testing.T) {
		migrationRequest := &model.InstallationDBMigrationRequest{
			InstallationID:         installation1.ID,
			DestinationDatabase:    model.InstallationDatabaseMultiTenantRDSPostgres,
			DestinationMultiTenant: &model.MultiTenantDBMigrationData{DatabaseID: "unknown"},
		}
		migrationOperation, err = client.MigrateInstallationDatabase(migrationRequest)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})

	t.Run("fail to trigger migration if destination database same as current", func(t *testing.T) {
		migrationRequest := &model.InstallationDBMigrationRequest{
			InstallationID:         installation1.ID,
			DestinationDatabase:    model.InstallationDatabaseMultiTenantRDSPostgres,
			DestinationMultiTenant: &model.MultiTenantDBMigrationData{DatabaseID: "database1"},
		}
		migrationOperation, err = client.MigrateInstallationDatabase(migrationRequest)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})

	t.Run("fail to trigger migration if destination database in different vpc", func(t *testing.T) {
		destinationDB := &model.MultitenantDatabase{
			ID:           "database3",
			VpcID:        "vpc2",
			DatabaseType: model.DatabaseEngineTypePostgres,
		}
		err = sqlStore.CreateMultitenantDatabase(destinationDB)
		require.NoError(t, err)

		migrationRequest := &model.InstallationDBMigrationRequest{
			InstallationID:         installation1.ID,
			DestinationDatabase:    model.InstallationDatabaseMultiTenantRDSPostgres,
			DestinationMultiTenant: &model.MultiTenantDBMigrationData{DatabaseID: "database3"},
		}
		migrationOperation, err = client.MigrateInstallationDatabase(migrationRequest)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})
}

func TestGetInstallationDBMigrationOperations(t *testing.T) {
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

	migrationOperations := []*model.InstallationDBMigrationOperation{
		{
			InstallationID: installation1.ID,
			State:          model.InstallationDBMigrationStateRequested,
		},
		{
			InstallationID: installation1.ID,
			State:          model.InstallationDBMigrationStateFailed,
		},
		{
			InstallationID: installation2.ID,
			State:          model.InstallationDBMigrationStateRequested,
		},
		{
			InstallationID: installation2.ID,
			State:          model.InstallationDBMigrationStateRequested,
		},
		{
			InstallationID: installation2.ID,
			State:          model.InstallationDBMigrationStateSucceeded,
		},
	}

	for i := range migrationOperations {
		err := sqlStore.CreateInstallationDBMigrationOperation(migrationOperations[i])
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond)
	}

	for _, testCase := range []struct {
		description string
		filter      model.GetInstallationDBMigrationOperationsRequest
		found       []*model.InstallationDBMigrationOperation
	}{
		{
			description: "all not deleted",
			filter:      model.GetInstallationDBMigrationOperationsRequest{Paging: model.AllPagesNotDeleted()},
			found:       migrationOperations,
		},
		{
			description: "1 per page",
			filter:      model.GetInstallationDBMigrationOperationsRequest{Paging: model.Paging{PerPage: 1}},
			found:       []*model.InstallationDBMigrationOperation{migrationOperations[4]},
		},
		{
			description: "2nd page",
			filter:      model.GetInstallationDBMigrationOperationsRequest{Paging: model.Paging{PerPage: 1, Page: 1}},
			found:       []*model.InstallationDBMigrationOperation{migrationOperations[3]},
		},
		{
			description: "filter by installation ID",
			filter:      model.GetInstallationDBMigrationOperationsRequest{Paging: model.AllPagesNotDeleted(), InstallationID: installation1.ID},
			found:       []*model.InstallationDBMigrationOperation{migrationOperations[0], migrationOperations[1]},
		},
		{
			description: "filter by state",
			filter:      model.GetInstallationDBMigrationOperationsRequest{Paging: model.AllPagesNotDeleted(), State: string(model.InstallationDBMigrationStateRequested)},
			found:       []*model.InstallationDBMigrationOperation{migrationOperations[0], migrationOperations[2], migrationOperations[3]},
		},
		{
			description: "no results",
			filter:      model.GetInstallationDBMigrationOperationsRequest{Paging: model.AllPagesNotDeleted(), InstallationID: "no-existent"},
			found:       []*model.InstallationDBMigrationOperation{},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {

			backups, err := client.GetInstallationDBMigrationOperations(&testCase.filter)
			require.NoError(t, err)
			require.Equal(t, len(testCase.found), len(backups))

			for i := 0; i < len(testCase.found); i++ {
				assert.Equal(t, testCase.found[i], backups[len(testCase.found)-1-i])
			}
		})
	}
}

func TestGetInstallationDBMigrationOperation(t *testing.T) {
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

	migrationOp := &model.InstallationDBMigrationOperation{
		InstallationID: "installation",
		State:          model.InstallationDBMigrationStateRequested,
	}
	err := sqlStore.CreateInstallationDBMigrationOperation(migrationOp)
	require.NoError(t, err)

	fetchedOp, err := client.GetInstallationDBMigrationOperation(migrationOp.ID)
	require.NoError(t, err)
	assert.Equal(t, migrationOp, fetchedOp)

	t.Run("return 404 if operation not found", func(t *testing.T) {
		_, err = client.GetInstallationDBMigrationOperation("not-real")
		require.EqualError(t, err, "failed with status code 404")
	})
}

type dbProviderMock struct {
	mock *mocks.MockDatabase
}

func (dbp *dbProviderMock) GetDatabase(installationID, dbType string) model.Database {
	return dbp.mock
}

func TestCommitInstallationDBMigrationOperation(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)
	ctrl := gomock.NewController(t)

	dbMock := mocks.NewMockDatabase(ctrl)
	dbProviderMock := &dbProviderMock{mock: dbMock}

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
		DBProvider: dbProviderMock,
	})

	ts := httptest.NewServer(router)
	client := model.NewClient(ts.URL)

	migrationOp := &model.InstallationDBMigrationOperation{
		InstallationID: "installation",
		State:          model.InstallationDBMigrationStateSucceeded,
	}
	err := sqlStore.CreateInstallationDBMigrationOperation(migrationOp)
	require.NoError(t, err)

	gomock.InOrder(dbMock.EXPECT().
		TeardownMigrated(sqlStore, migrationOp, gomock.Any()).
		Return(nil))

	committedOp, err := client.CommitInstallationDBMigration(migrationOp.ID)
	require.NoError(t, err)
	assert.Equal(t, model.InstallationDBMigrationStateCommitted, committedOp.State)

	t.Run("fail if migration not succeeded", func(t *testing.T) {
		_, err := client.CommitInstallationDBMigration(committedOp.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")

		committedOp.State = model.InstallationDBMigrationStateFailed
		err = sqlStore.UpdateInstallationDBMigrationOperationState(committedOp)
		require.NoError(t, err)

		_, err = client.CommitInstallationDBMigration(committedOp.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})
}

func TestRollbackInstallationDBMigrationOperation(t *testing.T) {
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

	installation := &model.Installation{
		Name:  "name",
		State: model.InstallationStateHibernating,
	}
	err := sqlStore.CreateInstallation(installation, nil, nil)
	require.NoError(t, err)

	migrationOp := &model.InstallationDBMigrationOperation{
		InstallationID: installation.ID,
		State:          model.InstallationDBMigrationStateSucceeded,
	}
	err = sqlStore.CreateInstallationDBMigrationOperation(migrationOp)
	require.NoError(t, err)

	rollbackOP, err := client.RollbackInstallationDBMigration(migrationOp.ID)
	require.NoError(t, err)
	assert.Equal(t, model.InstallationDBMigrationStateRollbackRequested, rollbackOP.State)
	installation, err = sqlStore.GetInstallation(installation.ID, false, false)
	require.NoError(t, err)
	assert.Equal(t, model.InstallationStateDBMigrationRollbackInProgress, installation.State)

	t.Run("failed to trigger rollback when installation in non-hibernating state", func(t *testing.T) {
		_, err = client.RollbackInstallationDBMigration(migrationOp.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})
}
