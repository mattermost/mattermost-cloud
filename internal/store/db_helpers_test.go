// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrCreateProxyDatabaseResourcesForInstallation(t *testing.T) {
	logger := testlib.MakeLogger(t)
	store := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, store)

	expectedDatabaseResourceCounts := func(t *testing.T, sqlStore *SQLStore, expectedMultitenant, expectedLogical, expectedSchemas int) {
		t.Helper()
		multitenantDatabases, err := sqlStore.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
			MaxInstallationsLimit: model.NoInstallationsLimit,
			Paging:                model.AllPagesNotDeleted(),
		})
		require.NoError(t, err)
		assert.Len(t, multitenantDatabases, expectedMultitenant)

		logicalDatabases, err := sqlStore.GetLogicalDatabases(&model.LogicalDatabaseFilter{
			Paging: model.AllPagesNotDeleted(),
		})
		require.NoError(t, err)
		assert.Len(t, logicalDatabases, expectedLogical)

		databaseSchemas, err := sqlStore.GetDatabaseSchemas(&model.DatabaseSchemaFilter{
			Paging: model.AllPagesNotDeleted(),
		})
		require.NoError(t, err)
		assert.Len(t, databaseSchemas, expectedSchemas)

		var totalInstallationCount int
		for _, mulltitenantDatabase := range multitenantDatabases {
			totalInstallationCount += mulltitenantDatabase.Installations.Count()
		}
		assert.Equal(t, len(databaseSchemas), totalInstallationCount)
	}

	multitenantDatabase := &model.MultitenantDatabase{
		DatabaseType:                       model.DatabaseEngineTypePostgresProxy,
		MaxInstallationsPerLogicalDatabase: 3,
	}
	err := store.CreateMultitenantDatabase(multitenantDatabase)
	require.NoError(t, err)

	t.Run("create resources only once", func(t *testing.T) {
		installation1 := createAndCheckDummyInstallation(t, store)

		createdResources, err := store.GetOrCreateProxyDatabaseResourcesForInstallation(installation1.ID, multitenantDatabase.ID)
		require.NoError(t, err)
		expectedDatabaseResourceCounts(t, store, 1, 1, 1)

		existingResources, err := store.GetOrCreateProxyDatabaseResourcesForInstallation(installation1.ID, multitenantDatabase.ID)
		require.NoError(t, err)
		require.Equal(t, createdResources, existingResources)
		expectedDatabaseResourceCounts(t, store, 1, 1, 1)
	})

	t.Run("reuse exisiting logical database", func(t *testing.T) {
		installation2 := createAndCheckDummyInstallation(t, store)

		_, err = store.GetOrCreateProxyDatabaseResourcesForInstallation(installation2.ID, multitenantDatabase.ID)
		require.NoError(t, err)
		expectedDatabaseResourceCounts(t, store, 1, 1, 2)

		installation3 := createAndCheckDummyInstallation(t, store)

		_, err = store.GetOrCreateProxyDatabaseResourcesForInstallation(installation3.ID, multitenantDatabase.ID)
		require.NoError(t, err)
		expectedDatabaseResourceCounts(t, store, 1, 1, 3)
	})

	t.Run("create new logical database when max is hit", func(t *testing.T) {
		installation4 := createAndCheckDummyInstallation(t, store)

		_, err = store.GetOrCreateProxyDatabaseResourcesForInstallation(installation4.ID, multitenantDatabase.ID)
		require.NoError(t, err)
		expectedDatabaseResourceCounts(t, store, 1, 2, 4)
	})

	t.Run("respect changes to max", func(t *testing.T) {
		multitenantDatabase, err = store.GetMultitenantDatabase(multitenantDatabase.ID)
		require.NoError(t, err)

		multitenantDatabase.MaxInstallationsPerLogicalDatabase = 1
		err = store.UpdateMultitenantDatabase(multitenantDatabase)
		require.NoError(t, err)

		installation5 := createAndCheckDummyInstallation(t, store)

		_, err = store.GetOrCreateProxyDatabaseResourcesForInstallation(installation5.ID, multitenantDatabase.ID)
		require.NoError(t, err)
		expectedDatabaseResourceCounts(t, store, 1, 3, 5)
	})

	t.Run("invalid multitenant database", func(t *testing.T) {
		installation6 := createAndCheckDummyInstallation(t, store)

		_, err = store.GetOrCreateProxyDatabaseResourcesForInstallation(installation6.ID, model.NewID())
		require.Error(t, err)
		expectedDatabaseResourceCounts(t, store, 1, 3, 5)
	})
}
