// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/internal/api"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDatabaseSchemas(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Metrics:    &mockMetrics{},
		Logger:     logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	t.Run("no logical databases", func(t *testing.T) {
		logicalDatabases, err := client.GetDatabaseSchemas(&model.GetDatabaseSchemaRequest{
			Paging: model.AllPagesWithDeleted(),
		})
		require.NoError(t, err)
		require.Empty(t, logicalDatabases)
	})

	t.Run("parameter handling", func(t *testing.T) {
		t.Run("invalid page", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/databases/database_schemas?page=invalid&per_page=100", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("invalid perPage", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/databases/database_schemas?page=0&per_page=invalid", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("no paging parameters", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/databases/database_schemas", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("missing page", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/databases/database_schemas?per_page=100", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("missing perPage", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/databases/database_schemas?page=1", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})
	})

	t.Run("results", func(t *testing.T) {
		dbs1 := &model.DatabaseSchema{
			LogicalDatabaseID: model.NewID(),
		}
		err := sqlStore.CreateDatabaseSchema(dbs1)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		installationID := model.NewID()
		dbs2 := &model.DatabaseSchema{
			LogicalDatabaseID: model.NewID(),
			InstallationID:    installationID,
		}
		err = sqlStore.CreateDatabaseSchema(dbs2)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		dbs3 := &model.DatabaseSchema{
			LogicalDatabaseID: model.NewID(),
		}
		err = sqlStore.CreateDatabaseSchema(dbs3)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		t.Run("get databases", func(t *testing.T) {
			testCases := []struct {
				Description string
				Request     *model.GetDatabaseSchemaRequest
				Expected    []*model.DatabaseSchema
			}{
				{
					"page 0, perPage 2, exclude deleted",
					&model.GetDatabaseSchemaRequest{
						Paging: model.Paging{
							Page:           0,
							PerPage:        2,
							IncludeDeleted: false,
						},
					},
					[]*model.DatabaseSchema{dbs1, dbs2},
				},

				{
					"page 1, perPage 2, exclude deleted",
					&model.GetDatabaseSchemaRequest{
						Paging: model.Paging{
							Page:           1,
							PerPage:        2,
							IncludeDeleted: false,
						},
					},
					[]*model.DatabaseSchema{dbs3},
				},

				{
					"page 0, perPage 2, include deleted",
					&model.GetDatabaseSchemaRequest{
						Paging: model.Paging{
							Page:           0,
							PerPage:        2,
							IncludeDeleted: true,
						},
					},
					[]*model.DatabaseSchema{dbs1, dbs2},
				},

				{
					"page 1, perPage 2, include deleted",
					&model.GetDatabaseSchemaRequest{
						Paging: model.Paging{
							Page:           1,
							PerPage:        2,
							IncludeDeleted: true,
						},
					},
					[]*model.DatabaseSchema{dbs3},
				},

				{
					"installation id",
					&model.GetDatabaseSchemaRequest{
						Paging:         model.AllPagesWithDeleted(),
						InstallationID: installationID,
					},
					[]*model.DatabaseSchema{dbs2},
				},
			}

			for _, testCase := range testCases {
				t.Run(testCase.Description, func(t *testing.T) {
					logicalDatabases, err := client.GetDatabaseSchemas(testCase.Request)
					require.NoError(t, err)
					require.Equal(t, testCase.Expected, logicalDatabases)
				})
			}
		})
	})
}

func TestGetDatabaseSchema(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Metrics:    &mockMetrics{},
		Logger:     logger,
	})

	ts := httptest.NewServer(router)
	client := model.NewClient(ts.URL)

	databaseSchema := &model.DatabaseSchema{
		InstallationID: model.NewID(),
	}

	err := sqlStore.CreateDatabaseSchema(databaseSchema)
	require.NoError(t, err)
	assert.NotEmpty(t, databaseSchema.ID)

	t.Run("success", func(t *testing.T) {
		fetchedDatabaseSchema, err := client.GetDatabaseSchema(databaseSchema.ID)
		require.NoError(t, err)
		assert.Equal(t, databaseSchema, fetchedDatabaseSchema)
	})

	t.Run("not found", func(t *testing.T) {
		fetchedDatabaseSchema, err := client.GetDatabaseSchema(model.NewID())
		require.NoError(t, err)
		assert.Nil(t, fetchedDatabaseSchema)
	})
}
