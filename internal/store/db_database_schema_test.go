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

func TestGetSchemaDatabase(t *testing.T) {
	logger := testlib.MakeLogger(t)
	store := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, store)

	multitenantDatabase := &model.MultitenantDatabase{
		DatabaseType: model.DatabaseEngineTypePostgresProxy,
	}
	err := store.CreateMultitenantDatabase(multitenantDatabase)
	require.NoError(t, err)

	logicalDatabase := &model.LogicalDatabase{
		MultitenantDatabaseID: multitenantDatabase.ID,
		Name:                  "ldb1",
	}
	createAndCheckLogicalDatabase(t, store, logicalDatabase)

	databaseSchema := &model.DatabaseSchema{
		LogicalDatabaseID: logicalDatabase.ID,
	}
	createAndCheckDatabaseSchema(t, store, databaseSchema)

	t.Run("success", func(t *testing.T) {
		databaseSchema2, err := store.GetDatabaseSchema(databaseSchema.ID)
		require.NoError(t, err)
		assert.NotNil(t, databaseSchema2)
	})

	t.Run("invalid id", func(t *testing.T) {
		databaseSchema, err := store.GetDatabaseSchema(model.NewID())
		require.NoError(t, err)
		assert.Nil(t, databaseSchema)
	})
}

func TestCreateDatabaseSchema(t *testing.T) {
	logger := testlib.MakeLogger(t)
	store := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, store)

	multitenantDatabase := &model.MultitenantDatabase{
		DatabaseType: model.DatabaseEngineTypePostgresProxy,
	}
	err := store.CreateMultitenantDatabase(multitenantDatabase)
	require.NoError(t, err)

	logicalDatabase := &model.LogicalDatabase{
		MultitenantDatabaseID: multitenantDatabase.ID,
		Name:                  "ldb1",
	}
	createAndCheckLogicalDatabase(t, store, logicalDatabase)

	t.Run("success", func(t *testing.T) {
		databaseSchema := &model.DatabaseSchema{
			LogicalDatabaseID: logicalDatabase.ID,
		}
		createAndCheckDatabaseSchema(t, store, databaseSchema)
	})
}

func TestDeleteDatabaseSchema(t *testing.T) {
	logger := testlib.MakeLogger(t)
	store := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, store)

	multitenantDatabase := &model.MultitenantDatabase{
		DatabaseType: model.DatabaseEngineTypePostgresProxy,
	}
	err := store.CreateMultitenantDatabase(multitenantDatabase)
	require.NoError(t, err)

	logicalDatabase := &model.LogicalDatabase{
		MultitenantDatabaseID: multitenantDatabase.ID,
		Name:                  "ldb1",
	}
	createAndCheckLogicalDatabase(t, store, logicalDatabase)

	databaseSchema := &model.DatabaseSchema{
		LogicalDatabaseID: logicalDatabase.ID,
	}
	createAndCheckDatabaseSchema(t, store, databaseSchema)

	t.Run("success", func(t *testing.T) {
		err = store.deleteDatabaseSchema(store.db, databaseSchema.ID)
		require.NoError(t, err)

		databaseSchema, err = store.GetDatabaseSchema(databaseSchema.ID)
		require.NoError(t, err)
		assert.True(t, databaseSchema.DeleteAt > 0)
	})
}

// Helpers

func createAndCheckDatabaseSchema(t *testing.T, store *SQLStore, databaseSchema *model.DatabaseSchema) {
	err := store.createDatabaseSchema(store.db, databaseSchema)
	require.NoError(t, err)
	assert.NotEmpty(t, databaseSchema.ID)
	assert.NotEmpty(t, databaseSchema.Name)
}
