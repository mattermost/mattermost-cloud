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

func TestGetLogicalDatabases(t *testing.T) {
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
		logicalDatabases, err := client.GetLogicalDatabases(&model.GetLogicalDatabasesRequest{
			Paging: model.AllPagesWithDeleted(),
		})
		require.NoError(t, err)
		require.Empty(t, logicalDatabases)
	})

	t.Run("parameter handling", func(t *testing.T) {
		t.Run("invalid page", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/databases/logical_databases?page=invalid&per_page=100", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("invalid perPage", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/databases/logical_databases?page=0&per_page=invalid", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("no paging parameters", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/databases/logical_databases", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("missing page", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/databases/logical_databases?per_page=100", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("missing perPage", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/databases/logical_databases?page=1", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})
	})

	t.Run("results", func(t *testing.T) {
		ldb1 := &model.LogicalDatabase{
			MultitenantDatabaseID: model.NewID(),
		}
		err := sqlStore.CreateLogicalDatabase(ldb1)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		ldb2 := &model.LogicalDatabase{
			MultitenantDatabaseID: model.NewID(),
		}
		err = sqlStore.CreateLogicalDatabase(ldb2)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		ldb3 := &model.LogicalDatabase{
			MultitenantDatabaseID: model.NewID(),
		}
		err = sqlStore.CreateLogicalDatabase(ldb3)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		t.Run("get databases", func(t *testing.T) {
			testCases := []struct {
				Description string
				Request     *model.GetLogicalDatabasesRequest
				Expected    []*model.LogicalDatabase
			}{
				{
					"page 0, perPage 2, exclude deleted",
					&model.GetLogicalDatabasesRequest{
						Paging: model.Paging{
							Page:           0,
							PerPage:        2,
							IncludeDeleted: false,
						},
					},
					[]*model.LogicalDatabase{ldb1, ldb2},
				},

				{
					"page 1, perPage 2, exclude deleted",
					&model.GetLogicalDatabasesRequest{
						Paging: model.Paging{
							Page:           1,
							PerPage:        2,
							IncludeDeleted: false,
						},
					},
					[]*model.LogicalDatabase{ldb3},
				},

				{
					"page 0, perPage 2, include deleted",
					&model.GetLogicalDatabasesRequest{
						Paging: model.Paging{
							Page:           0,
							PerPage:        2,
							IncludeDeleted: true,
						},
					},
					[]*model.LogicalDatabase{ldb1, ldb2},
				},

				{
					"page 1, perPage 2, include deleted",
					&model.GetLogicalDatabasesRequest{
						Paging: model.Paging{
							Page:           1,
							PerPage:        2,
							IncludeDeleted: true,
						},
					},
					[]*model.LogicalDatabase{ldb3},
				},
			}

			for _, testCase := range testCases {
				t.Run(testCase.Description, func(t *testing.T) {
					logicalDatabases, err := client.GetLogicalDatabases(testCase.Request)
					require.NoError(t, err)
					require.Equal(t, testCase.Expected, logicalDatabases)
				})
			}
		})
	})
}

func TestGetLogicalDatabase(t *testing.T) {
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

	logicalDatabase := &model.LogicalDatabase{
		MultitenantDatabaseID: model.NewID(),
	}

	err := sqlStore.CreateLogicalDatabase(logicalDatabase)
	require.NoError(t, err)
	assert.NotEmpty(t, logicalDatabase.ID)

	t.Run("success", func(t *testing.T) {
		fetchedLogicalDatabase, err := client.GetLogicalDatabase(logicalDatabase.ID)
		require.NoError(t, err)
		assert.Equal(t, logicalDatabase, fetchedLogicalDatabase)
	})

	t.Run("not found", func(t *testing.T) {
		fetchedLogicalDatabase, err := client.GetLogicalDatabase(model.NewID())
		require.NoError(t, err)
		assert.Nil(t, fetchedLogicalDatabase)
	})
}
