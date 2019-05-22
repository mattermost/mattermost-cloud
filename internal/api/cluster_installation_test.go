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
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/stretchr/testify/require"
)

func TestGetClusterInstallations(t *testing.T) {
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

	t.Run("unknown cluster installation", func(t *testing.T) {
		clusterInstallation, err := client.GetClusterInstallation(model.NewID())
		require.NoError(t, err)
		require.Nil(t, clusterInstallation)
	})

	t.Run("no cluster installations", func(t *testing.T) {
		clusterInstallations, err := client.GetClusterInstallations(&model.GetClusterInstallationsRequest{
			Page:           0,
			PerPage:        10,
			IncludeDeleted: true,
		})
		require.NoError(t, err)
		require.Empty(t, clusterInstallations)
	})

	t.Run("parameter handling", func(t *testing.T) {
		t.Run("invalid page", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/cluster_installations?page=invalid&per_page=100", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("invalid perPage", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/cluster_installations?page=0&per_page=invalid", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("no paging parameters", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/cluster_installations", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("missing page", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/cluster_installations?per_page=100", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("missing perPage", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/cluster_installations?page=1", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})
	})

	t.Run("results", func(t *testing.T) {
		clusterID1 := model.NewID()
		clusterID2 := model.NewID()
		installationID1 := model.NewID()
		installationID2 := model.NewID()

		clusterInstallation1 := &model.ClusterInstallation{
			ClusterID:      clusterID1,
			InstallationID: installationID1,
			Namespace:      "cluster installation 1",
			State:          model.ClusterInstallationStateCreationRequested,
		}
		err := sqlStore.CreateClusterInstallation(clusterInstallation1)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		clusterInstallation2 := &model.ClusterInstallation{
			ClusterID:      clusterID1,
			InstallationID: installationID2,
			Namespace:      "cluster installation 2",
			State:          model.ClusterInstallationStateStable,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation2)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		clusterInstallation3 := &model.ClusterInstallation{
			ClusterID:      clusterID2,
			InstallationID: installationID1,
			Namespace:      "cluster installation 3",
			State:          model.ClusterInstallationStateDeletionRequested,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation3)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		clusterInstallation4 := &model.ClusterInstallation{
			ClusterID:      clusterID2,
			InstallationID: installationID2,
			Namespace:      "cluster installation 4",
			State:          model.ClusterInstallationStateDeleted,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation4)
		require.NoError(t, err)
		err = sqlStore.DeleteClusterInstallation(clusterInstallation4.ID)
		require.NoError(t, err)
		clusterInstallation4, err = client.GetClusterInstallation(clusterInstallation4.ID)
		require.NoError(t, err)

		t.Run("get cluster installation", func(t *testing.T) {
			t.Run("cluster installation 1", func(t *testing.T) {
				clusterInstallation, err := client.GetClusterInstallation(clusterInstallation1.ID)
				require.NoError(t, err)
				require.Equal(t, clusterInstallation1, clusterInstallation)
			})

			t.Run("get deleted cluster installation", func(t *testing.T) {
				clusterInstallation, err := client.GetClusterInstallation(clusterInstallation4.ID)
				require.NoError(t, err)
				require.Equal(t, clusterInstallation4, clusterInstallation)
			})
		})

		t.Run("get cluster installations", func(t *testing.T) {
			testCases := []struct {
				Description                    string
				GetClusterInstallationsRequest *model.GetClusterInstallationsRequest
				Expected                       []*model.ClusterInstallation
			}{
				{
					"page 0, perPage 2, exclude deleted",
					&model.GetClusterInstallationsRequest{
						Page:           0,
						PerPage:        2,
						IncludeDeleted: false,
					},
					[]*model.ClusterInstallation{clusterInstallation1, clusterInstallation2},
				},

				{
					"page 1, perPage 2, exclude deleted",
					&model.GetClusterInstallationsRequest{
						Page:           1,
						PerPage:        2,
						IncludeDeleted: false,
					},
					[]*model.ClusterInstallation{clusterInstallation3},
				},

				{
					"page 0, perPage 2, include deleted",
					&model.GetClusterInstallationsRequest{
						Page:           0,
						PerPage:        2,
						IncludeDeleted: true,
					},
					[]*model.ClusterInstallation{clusterInstallation1, clusterInstallation2},
				},

				{
					"page 1, perPage 2, include deleted",
					&model.GetClusterInstallationsRequest{
						Page:           1,
						PerPage:        2,
						IncludeDeleted: true,
					},
					[]*model.ClusterInstallation{clusterInstallation3, clusterInstallation4},
				},

				{
					"filter by cluster",
					&model.GetClusterInstallationsRequest{
						Page:           0,
						PerPage:        100,
						ClusterID:      clusterID1,
						IncludeDeleted: false,
					},
					[]*model.ClusterInstallation{clusterInstallation1, clusterInstallation2},
				},

				{
					"filter by installation",
					&model.GetClusterInstallationsRequest{
						Page:           0,
						PerPage:        100,
						InstallationID: installationID1,
						IncludeDeleted: false,
					},
					[]*model.ClusterInstallation{clusterInstallation1, clusterInstallation3},
				},

				{
					"filter by cluster + installation",
					&model.GetClusterInstallationsRequest{
						Page:           0,
						PerPage:        100,
						ClusterID:      clusterID2,
						InstallationID: installationID2,
						IncludeDeleted: false,
					},
					[]*model.ClusterInstallation{},
				},

				{
					"filter by cluster + installation, include deleted",
					&model.GetClusterInstallationsRequest{
						Page:           0,
						PerPage:        100,
						ClusterID:      clusterID2,
						InstallationID: installationID2,
						IncludeDeleted: true,
					},
					[]*model.ClusterInstallation{clusterInstallation4},
				},
			}

			for _, testCase := range testCases {
				t.Run(testCase.Description, func(t *testing.T) {
					clusterInstallations, err := client.GetClusterInstallations(testCase.GetClusterInstallationsRequest)
					require.NoError(t, err)
					require.Equal(t, testCase.Expected, clusterInstallations)
				})
			}
		})
	})
}

func TestGetClusterInstallationConfig(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:       sqlStore,
		Supervisor:  &mockSupervisor{},
		Provisioner: &mockProvisioner{},
		Logger:      logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	cluster := &model.Cluster{}
	err := sqlStore.CreateCluster(cluster)
	require.NoError(t, err)

	clusterInstallation1 := &model.ClusterInstallation{
		ClusterID:      cluster.ID,
		InstallationID: model.NewID(),
	}
	err = sqlStore.CreateClusterInstallation(clusterInstallation1)
	require.NoError(t, err)

	t.Run("unknown cluster installation", func(t *testing.T) {
		config, err := client.GetClusterInstallationConfig(model.NewID())
		require.NoError(t, err)
		require.Nil(t, config)
	})

	t.Run("success", func(t *testing.T) {
		config, err := client.GetClusterInstallationConfig(clusterInstallation1.ID)
		require.NoError(t, err)
		require.Contains(t, config, "ServiceSettings")
	})
}

func TestSetClusterInstallationConfig(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:       sqlStore,
		Supervisor:  &mockSupervisor{},
		Provisioner: &mockProvisioner{},
		Logger:      logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	cluster := &model.Cluster{}
	err := sqlStore.CreateCluster(cluster)
	require.NoError(t, err)

	clusterInstallation1 := &model.ClusterInstallation{
		ClusterID:      cluster.ID,
		InstallationID: model.NewID(),
	}
	err = sqlStore.CreateClusterInstallation(clusterInstallation1)
	require.NoError(t, err)

	config := map[string]interface{}{"ServiceSettings": map[string]interface{}{"SiteURL": "test.example.com"}}

	t.Run("unknown cluster installation", func(t *testing.T) {
		err := client.SetClusterInstallationConfig(model.NewID(), config)
		require.EqualError(t, err, "failed with status code 404")
	})

	t.Run("invalid payload", func(t *testing.T) {
		httpRequest, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/cluster_installation/%s/config", ts.URL, clusterInstallation1.ID), bytes.NewReader([]byte("invalid")))
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(httpRequest)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("empty payload", func(t *testing.T) {
		httpRequest, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/cluster_installation/%s/config", ts.URL, clusterInstallation1.ID), bytes.NewReader([]byte("")))
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(httpRequest)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("success", func(t *testing.T) {
		err := client.SetClusterInstallationConfig(clusterInstallation1.ID, config)
		require.NoError(t, err)
	})
}
