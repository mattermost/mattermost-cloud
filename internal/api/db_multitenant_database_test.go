// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api_test

import (
	"bytes"
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
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMultitenantDatabases(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	t.Run("no databases", func(t *testing.T) {
		databases, err := client.GetMultitenantDatabases(&model.GetMultitenantDatabasesRequest{
			Paging: model.AllPagesWithDeleted(),
		})
		require.NoError(t, err)
		require.Empty(t, databases)
	})

	t.Run("parameter handling", func(t *testing.T) {
		t.Run("invalid page", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/databases?page=invalid&per_page=100", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("invalid perPage", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/databases?page=0&per_page=invalid", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("no paging parameters", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/databases", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("missing page", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/databases?per_page=100", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("missing perPage", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/databases?page=1", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})
	})

	t.Run("results", func(t *testing.T) {
		database1 := &model.MultitenantDatabase{
			RdsClusterID: model.NewID(),
		}
		err := sqlStore.CreateMultitenantDatabase(database1)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		database2 := &model.MultitenantDatabase{
			RdsClusterID: model.NewID(),
		}
		err = sqlStore.CreateMultitenantDatabase(database2)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		database3 := &model.MultitenantDatabase{
			RdsClusterID: model.NewID(),
		}
		err = sqlStore.CreateMultitenantDatabase(database3)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		t.Run("get databases", func(t *testing.T) {
			testCases := []struct {
				Description string
				Request     *model.GetMultitenantDatabasesRequest
				Expected    []*model.MultitenantDatabase
			}{
				{
					"page 0, perPage 2, exclude deleted",
					&model.GetMultitenantDatabasesRequest{
						Paging: model.Paging{
							Page:           0,
							PerPage:        2,
							IncludeDeleted: false,
						},
					},
					[]*model.MultitenantDatabase{database1, database2},
				},

				{
					"page 1, perPage 2, exclude deleted",
					&model.GetMultitenantDatabasesRequest{
						Paging: model.Paging{
							Page:           1,
							PerPage:        2,
							IncludeDeleted: false,
						},
					},
					[]*model.MultitenantDatabase{database3},
				},

				{
					"page 0, perPage 2, include deleted",
					&model.GetMultitenantDatabasesRequest{
						Paging: model.Paging{
							Page:           0,
							PerPage:        2,
							IncludeDeleted: true,
						},
					},
					[]*model.MultitenantDatabase{database1, database2},
				},

				{
					"page 1, perPage 2, include deleted",
					&model.GetMultitenantDatabasesRequest{
						Paging: model.Paging{
							Page:           1,
							PerPage:        2,
							IncludeDeleted: true,
						},
					},
					[]*model.MultitenantDatabase{database3},
				},
			}

			for _, testCase := range testCases {
				t.Run(testCase.Description, func(t *testing.T) {
					databases, err := client.GetMultitenantDatabases(testCase.Request)
					require.NoError(t, err)
					require.Equal(t, testCase.Expected, databases)
				})
			}
		})
	})
}

func TestGetMultitenantDatabase(t *testing.T) {
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

	multitenantDatabase := &model.MultitenantDatabase{
		VpcID: model.NewID(),
	}

	err := sqlStore.CreateMultitenantDatabase(multitenantDatabase)
	require.NoError(t, err)
	assert.NotEmpty(t, multitenantDatabase.ID)

	t.Run("success", func(t *testing.T) {
		fetchedMultitenantDatabase, err := client.GetMultitenantDatabase(multitenantDatabase.ID)
		require.NoError(t, err)
		assert.Equal(t, multitenantDatabase, fetchedMultitenantDatabase)
	})

	t.Run("not found", func(t *testing.T) {
		fetchedMultitenantDatabase, err := client.GetMultitenantDatabase(model.NewID())
		require.NoError(t, err)
		assert.Nil(t, fetchedMultitenantDatabase)
	})
}

func TestUpdateMultitenantDatabase(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	database1 := &model.MultitenantDatabase{
		ID:                                 model.NewID(),
		DatabaseType:                       model.DatabaseEngineTypePostgresProxy,
		MaxInstallationsPerLogicalDatabase: 5,
	}
	err := sqlStore.CreateMultitenantDatabase(database1)
	require.NoError(t, err)

	t.Run("invalid payload", func(t *testing.T) {
		httpRequest, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/database/%s", ts.URL, database1.ID), bytes.NewReader([]byte("invalid")))
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(httpRequest)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("empty payload", func(t *testing.T) {
		httpRequest, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/database/%s", ts.URL, database1.ID), bytes.NewReader([]byte("")))
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(httpRequest)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("unknown database", func(t *testing.T) {
		databases, err := client.UpdateMultitenantDatabase(model.NewID(), &model.PatchMultitenantDatabaseRequest{})
		require.EqualError(t, err, "failed with status code 404")
		require.Nil(t, databases)
	})

	t.Run("update", func(t *testing.T) {
		database1, err := client.UpdateMultitenantDatabase(database1.ID,
			&model.PatchMultitenantDatabaseRequest{
				MaxInstallationsPerLogicalDatabase: iToP(10),
			})
		require.NoError(t, err)
		assert.Equal(t, int64(10), database1.MaxInstallationsPerLogicalDatabase)

		database1, err = sqlStore.GetMultitenantDatabase(database1.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(10), database1.MaxInstallationsPerLogicalDatabase)
	})
}

type mockAWSClient struct {
	clusterExists bool
	expectedRDSID string
}

func (m mockAWSClient) SwitchClusterTags(clusterID string, targetClusterID string, logger log.FieldLogger) error {
	return nil
}

func (m mockAWSClient) RDSDBCLusterExists(awsID string) (bool, error) {
	if awsID != m.expectedRDSID {
		panic("expected different RDS ID")
	}
	return m.clusterExists, nil
}

func TestDeleteMultitenantDatabase(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)

	router := mux.NewRouter()
	context := &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
		AwsClient:  mockAWSClient{clusterExists: false, expectedRDSID: "rds-id"},
	}
	api.Register(router, context)
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	t.Run("fail when deleting invalid cluster", func(t *testing.T) {
		err := client.DeleteMultitenantDatabase("invalid-cluster", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "404")
	})

	makeDBCluster := func() *model.MultitenantDatabase {
		database1 := &model.MultitenantDatabase{
			ID:           model.NewID(),
			DatabaseType: model.DatabaseEngineTypePostgres,
			RdsClusterID: "rds-id",
		}
		err := sqlStore.CreateMultitenantDatabase(database1)
		require.NoError(t, err)
		return database1
	}

	db := makeDBCluster()

	// Delete without force when cluster does not exist.
	err := client.DeleteMultitenantDatabase(db.ID, false)
	require.NoError(t, err)
	fetched, err := sqlStore.GetMultitenantDatabase(db.ID)
	require.NoError(t, err)
	assert.True(t, fetched.DeleteAt > 0)

	t.Run("do not fail when already deleted", func(t *testing.T) {
		err = client.DeleteMultitenantDatabase(db.ID, false)
		require.NoError(t, err)
	})

	// Fail without force if cluster does not exist.
	context.AwsClient = mockAWSClient{clusterExists: true, expectedRDSID: "rds-id"}
	db2 := makeDBCluster()

	err = client.DeleteMultitenantDatabase(db2.ID, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")

	// Succeed with force even if cluster exists.
	err = client.DeleteMultitenantDatabase(db2.ID, true)
	require.NoError(t, err)
}
