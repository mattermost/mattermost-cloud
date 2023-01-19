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

func TestGetLogicalDatabase(t *testing.T) {
	logger := testlib.MakeLogger(t)
	store := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, store)

	multitenantDatabase := &model.MultitenantDatabase{
		DatabaseType: model.DatabaseEngineTypePostgresProxy,
	}
	err := store.CreateMultitenantDatabase(multitenantDatabase)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		logicalDatabase := &model.LogicalDatabase{
			MultitenantDatabaseID: multitenantDatabase.ID,
			Name:                  "ldb1",
		}
		createAndCheckLogicalDatabase(t, store, logicalDatabase)
		_, err := store.GetLogicalDatabase(logicalDatabase.ID)
		require.NoError(t, err)
	})

	t.Run("invalid id", func(t *testing.T) {
		logicalDatabase, err := store.GetLogicalDatabase(model.NewID())
		require.NoError(t, err)
		assert.Nil(t, logicalDatabase)
	})
}

func TestCreateLogicalDatabase(t *testing.T) {
	logger := testlib.MakeLogger(t)
	store := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, store)

	multitenantDatabase := &model.MultitenantDatabase{
		DatabaseType: model.DatabaseEngineTypePostgresProxy,
	}
	err := store.CreateMultitenantDatabase(multitenantDatabase)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		logicalDatabase := &model.LogicalDatabase{
			MultitenantDatabaseID: multitenantDatabase.ID,
			Name:                  "ldb1",
		}
		createAndCheckLogicalDatabase(t, store, logicalDatabase)
	})
}

func TestDeleteLogicalDatabase(t *testing.T) {
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
		err = store.DeleteLogicalDatabase(logicalDatabase.ID)
		require.NoError(t, err)

		logicalDatabase, err = store.GetLogicalDatabase(logicalDatabase.ID)
		require.NoError(t, err)
		assert.True(t, logicalDatabase.DeleteAt > 0)
	})
}

// Helpers

func createAndCheckLogicalDatabase(t *testing.T, store *SQLStore, logicalDatabase *model.LogicalDatabase) {
	err := store.CreateLogicalDatabase(logicalDatabase)
	require.NoError(t, err)
	assert.NotEmpty(t, logicalDatabase.ID)
}
