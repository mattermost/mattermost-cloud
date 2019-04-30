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
	"github.com/mattermost/mattermost-cloud/internal/model"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/stretchr/testify/require"
)

type mockSupervisor struct {
}

func (s *mockSupervisor) Do() error {
	return nil
}

func TestClusters(t *testing.T) {
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

	client := api.NewClient(ts.URL)

	t.Run("unknown cluster", func(t *testing.T) {
		t.Run("get cluster", func(t *testing.T) {
			cluster, err := client.GetCluster("unknown")
			require.NoError(t, err)
			require.Nil(t, cluster)
		})

	})

	t.Run("no clusters", func(t *testing.T) {
		clusters, err := client.GetClusters(&api.GetClustersRequest{
			Page:           0,
			PerPage:        10,
			IncludeDeleted: true,
		})
		require.NoError(t, err)
		require.Empty(t, clusters)
	})

	t.Run("get clusters", func(t *testing.T) {
		t.Run("invalid page", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/clusters?page=invalid&per_page=100", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("invalid perPage", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/clusters?page=0&per_page=invalid", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("no paging parameters", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/clusters", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("missing page", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/clusters?per_page=100", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("missing perPage", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/clusters?page=1", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})
	})

	t.Run("clusters", func(t *testing.T) {
		cluster1, err := client.CreateCluster(&api.CreateClusterRequest{
			Provider: model.ProviderAWS,
			Size:     model.SizeAlef500,
			Zones:    []string{"zone"},
		})
		require.NoError(t, err)
		require.NotNil(t, cluster1)
		require.Equal(t, model.ProviderAWS, cluster1.Provider)
		require.Equal(t, model.SizeAlef500, cluster1.Size)
		// require.Equal(t, []string{"zone"}, cluster1.Zones)

		actualCluster1, err := client.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, cluster1.ID, actualCluster1.ID)
		require.Equal(t, model.ProviderAWS, actualCluster1.Provider)
		require.Equal(t, model.SizeAlef500, actualCluster1.Size)
		// require.Equal(t, []string{"zone"}, actualCluster1.Zones)
		require.Equal(t, model.ClusterStateCreationRequested, actualCluster1.State)

		time.Sleep(1 * time.Millisecond)

		cluster2, err := client.CreateCluster(&api.CreateClusterRequest{
			Provider: model.ProviderAWS,
			Size:     model.SizeAlef500,
			Zones:    []string{"zone"},
		})
		require.NoError(t, err)
		require.NotNil(t, cluster2)
		require.Equal(t, model.ProviderAWS, cluster2.Provider)
		require.Equal(t, model.SizeAlef500, cluster2.Size)
		// require.Equal(t, []string{"zone"}, cluster2.Zones)

		actualCluster2, err := client.GetCluster(cluster2.ID)
		require.NoError(t, err)
		require.Equal(t, cluster2.ID, actualCluster2.ID)
		require.Equal(t, model.ProviderAWS, actualCluster2.Provider)
		require.Equal(t, model.SizeAlef500, actualCluster2.Size)
		// require.Equal(t, []string{"zone"}, actualCluster2.Zones)
		require.Equal(t, model.ClusterStateCreationRequested, actualCluster2.State)

		time.Sleep(1 * time.Millisecond)

		cluster3, err := client.CreateCluster(&api.CreateClusterRequest{
			Provider: model.ProviderAWS,
			Size:     model.SizeAlef500,
			Zones:    []string{"zone"},
		})
		require.NoError(t, err)
		require.NotNil(t, cluster3)
		require.Equal(t, model.ProviderAWS, cluster3.Provider)
		require.Equal(t, model.SizeAlef500, cluster3.Size)
		// require.Equal(t, []string{"zone"}, cluster3.Zones)

		actualCluster3, err := client.GetCluster(cluster3.ID)
		require.NoError(t, err)
		require.Equal(t, cluster3.ID, actualCluster3.ID)
		require.Equal(t, model.ProviderAWS, actualCluster3.Provider)
		require.Equal(t, model.SizeAlef500, actualCluster3.Size)
		// require.Equal(t, []string{"zone"}, actualCluster3.Zones)
		require.Equal(t, model.ClusterStateCreationRequested, actualCluster3.State)

		t.Run("get clusters, page 0, perPage 2, exclude deleted", func(t *testing.T) {
			clusters, err := client.GetClusters(&api.GetClustersRequest{
				Page:           0,
				PerPage:        2,
				IncludeDeleted: false,
			})
			require.NoError(t, err)
			require.Equal(t, []*model.Cluster{cluster1, cluster2}, clusters)
		})

		t.Run("get clusters, page 1, perPage 2, exclude deleted", func(t *testing.T) {
			clusters, err := client.GetClusters(&api.GetClustersRequest{
				Page:           1,
				PerPage:        2,
				IncludeDeleted: false,
			})
			require.NoError(t, err)
			require.Equal(t, []*model.Cluster{cluster3}, clusters)
		})

		t.Run("delete cluster", func(t *testing.T) {
			cluster2.State = model.ClusterStateStable
			err := sqlStore.UpdateCluster(cluster2)
			require.NoError(t, err)

			err = client.DeleteCluster(cluster2.ID)
			require.NoError(t, err)

			cluster2, err = client.GetCluster(cluster2.ID)
			require.NoError(t, err)
			require.Equal(t, model.ClusterStateDeletionRequested, cluster2.State)
		})

		t.Run("get clusters after deletion request", func(t *testing.T) {
			t.Run("page 0, perPage 2, exclude deleted", func(t *testing.T) {
				clusters, err := client.GetClusters(&api.GetClustersRequest{
					Page:           0,
					PerPage:        2,
					IncludeDeleted: false,
				})
				require.NoError(t, err)
				require.Equal(t, []*model.Cluster{cluster1, cluster2}, clusters)
			})

			t.Run("page 1, perPage 2, exclude deleted", func(t *testing.T) {
				clusters, err := client.GetClusters(&api.GetClustersRequest{
					Page:           1,
					PerPage:        2,
					IncludeDeleted: false,
				})
				require.NoError(t, err)
				require.Equal(t, []*model.Cluster{cluster3}, clusters)
			})

			t.Run("page 0, perPage 2, include deleted", func(t *testing.T) {
				clusters, err := client.GetClusters(&api.GetClustersRequest{
					Page:           0,
					PerPage:        2,
					IncludeDeleted: true,
				})
				require.NoError(t, err)
				require.Equal(t, []*model.Cluster{cluster1, cluster2}, clusters)
			})

			t.Run("page 1, perPage 2, include deleted", func(t *testing.T) {
				clusters, err := client.GetClusters(&api.GetClustersRequest{
					Page:           1,
					PerPage:        2,
					IncludeDeleted: true,
				})
				require.NoError(t, err)
				require.Equal(t, []*model.Cluster{cluster3}, clusters)
			})
		})

		err = sqlStore.DeleteCluster(cluster2.ID)
		require.NoError(t, err)

		cluster2, err = client.GetCluster(cluster2.ID)
		require.NoError(t, err)
		require.NotEqual(t, 0, cluster2.DeleteAt)

		t.Run("get clusters after actual deletion", func(t *testing.T) {
			t.Run("page 0, perPage 2, exclude deleted", func(t *testing.T) {
				clusters, err := client.GetClusters(&api.GetClustersRequest{
					Page:           0,
					PerPage:        2,
					IncludeDeleted: false,
				})
				require.NoError(t, err)
				require.Equal(t, []*model.Cluster{cluster1, cluster3}, clusters)
			})

			t.Run("page 1, perPage 2, exclude deleted", func(t *testing.T) {
				clusters, err := client.GetClusters(&api.GetClustersRequest{
					Page:           1,
					PerPage:        2,
					IncludeDeleted: false,
				})
				require.NoError(t, err)
				require.Equal(t, []*model.Cluster{}, clusters)
			})

			t.Run("page 0, perPage 2, include deleted", func(t *testing.T) {
				clusters, err := client.GetClusters(&api.GetClustersRequest{
					Page:           0,
					PerPage:        2,
					IncludeDeleted: true,
				})
				require.NoError(t, err)
				require.Equal(t, []*model.Cluster{cluster1, cluster2}, clusters)
			})

			t.Run("page 1, perPage 2, include deleted", func(t *testing.T) {
				clusters, err := client.GetClusters(&api.GetClustersRequest{
					Page:           1,
					PerPage:        2,
					IncludeDeleted: true,
				})
				require.NoError(t, err)
				require.Equal(t, []*model.Cluster{cluster3}, clusters)
			})
		})
	})
}

func TestCreateCluster(t *testing.T) {
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

	client := api.NewClient(ts.URL)

	t.Run("invalid payload", func(t *testing.T) {
		resp, err := http.Post(fmt.Sprintf("%s/api/clusters", ts.URL), "application/json", bytes.NewReader([]byte("invalid")))
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("empty payload", func(t *testing.T) {
		resp, err := http.Post(fmt.Sprintf("%s/api/clusters", ts.URL), "application/json", bytes.NewReader([]byte("")))
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, resp.StatusCode)
	})

	t.Run("invalid provider", func(t *testing.T) {
		_, err := client.CreateCluster(&api.CreateClusterRequest{
			Provider: "invalid",
			Size:     model.SizeAlef500,
			Zones:    []string{"zone"},
		})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("invalid size", func(t *testing.T) {
		_, err := client.CreateCluster(&api.CreateClusterRequest{
			Provider: model.ProviderAWS,
			Size:     "invalid",
			Zones:    []string{"zone"},
		})
		require.EqualError(t, err, "failed with status code 400")
	})
}

func TestRetryCreateCluster(t *testing.T) {
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

	client := api.NewClient(ts.URL)

	cluster1, err := client.CreateCluster(&api.CreateClusterRequest{
		Provider: model.ProviderAWS,
		Size:     model.SizeAlef500,
		Zones:    []string{"zone"},
	})
	require.NoError(t, err)

	t.Run("unknown cluster", func(t *testing.T) {
		err := client.RetryCreateCluster("unknown")
		require.EqualError(t, err, "failed with status code 404")
	})

	t.Run("while locked", func(t *testing.T) {
		cluster1.State = model.ClusterStateStable
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		locked, err := sqlStore.LockCluster(cluster1.ID)
		require.NoError(t, err)
		require.True(t, locked)
		defer func() {
			unlocked, err := sqlStore.UnlockCluster(cluster1.ID, false)
			require.NoError(t, err)
			require.True(t, unlocked)
		}()

		err = client.RetryCreateCluster(cluster1.ID)
		require.EqualError(t, err, "failed with status code 409")
	})

	t.Run("while creating", func(t *testing.T) {
		cluster1.State = model.ClusterStateCreationRequested
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		err = client.RetryCreateCluster(cluster1.ID)
		require.NoError(t, err)

		cluster1, err = client.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, model.ClusterStateCreationRequested, cluster1.State)
	})

	t.Run("while stable", func(t *testing.T) {
		cluster1.State = model.ClusterStateStable
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		err = client.RetryCreateCluster(cluster1.ID)
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("while creation failed", func(t *testing.T) {
		cluster1.State = model.ClusterStateCreationFailed
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		err = client.RetryCreateCluster(cluster1.ID)
		require.NoError(t, err)

		cluster1, err = client.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, model.ClusterStateCreationRequested, cluster1.State)
	})
}

func TestUpgradeCluster(t *testing.T) {
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

	client := api.NewClient(ts.URL)

	cluster1, err := client.CreateCluster(&api.CreateClusterRequest{
		Provider: model.ProviderAWS,
		Size:     model.SizeAlef500,
		Zones:    []string{"zone"},
	})
	require.NoError(t, err)

	t.Run("unknown cluster", func(t *testing.T) {
		err := client.UpgradeCluster("unknown", "latest")
		require.EqualError(t, err, "failed with status code 404")
	})

	t.Run("while locked", func(t *testing.T) {
		cluster1.State = model.ClusterStateStable
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		locked, err := sqlStore.LockCluster(cluster1.ID)
		require.NoError(t, err)
		require.True(t, locked)
		defer func() {
			unlocked, err := sqlStore.UnlockCluster(cluster1.ID, false)
			require.NoError(t, err)
			require.True(t, unlocked)
		}()

		err = client.UpgradeCluster(cluster1.ID, "latest")
		require.EqualError(t, err, "failed with status code 409")
	})

	t.Run("while upgrading", func(t *testing.T) {
		cluster1.State = model.ClusterStateUpgradeRequested
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		err = client.UpgradeCluster(cluster1.ID, "latest")
		require.NoError(t, err)

		cluster1, err = client.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, model.ClusterStateUpgradeRequested, cluster1.State)
	})

	t.Run("while stable", func(t *testing.T) {
		cluster1.State = model.ClusterStateStable
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		err = client.UpgradeCluster(cluster1.ID, "latest")
		require.NoError(t, err)

		cluster1, err = client.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, model.ClusterStateUpgradeRequested, cluster1.State)
	})

	t.Run("while stable, to invalid version", func(t *testing.T) {
		cluster1.State = model.ClusterStateStable
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		err = client.UpgradeCluster(cluster1.ID, "invalid")
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("while deleting", func(t *testing.T) {
		cluster1.State = model.ClusterStateDeletionRequested
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		err = client.UpgradeCluster(cluster1.ID, "latest")
		require.EqualError(t, err, "failed with status code 400")
	})
}

func TestDeleteCluster(t *testing.T) {
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

	client := api.NewClient(ts.URL)

	cluster1, err := client.CreateCluster(&api.CreateClusterRequest{
		Provider: model.ProviderAWS,
		Size:     model.SizeAlef500,
		Zones:    []string{"zone"},
	})
	require.NoError(t, err)

	t.Run("unknown cluster", func(t *testing.T) {
		err := client.DeleteCluster("unknown")
		require.EqualError(t, err, "failed with status code 404")
	})

	t.Run("while locked", func(t *testing.T) {
		cluster1.State = model.ClusterStateStable
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		locked, err := sqlStore.LockCluster(cluster1.ID)
		require.NoError(t, err)
		require.True(t, locked)
		defer func() {
			unlocked, err := sqlStore.UnlockCluster(cluster1.ID, false)
			require.NoError(t, err)
			require.True(t, unlocked)

			cluster1, err = client.GetCluster(cluster1.ID)
			require.NoError(t, err)
			require.Equal(t, int64(0), cluster1.LockAcquiredAt)
		}()

		err = client.DeleteCluster(cluster1.ID)
		require.EqualError(t, err, "failed with status code 409")
	})

	t.Run("while stable", func(t *testing.T) {
		cluster1.State = model.ClusterStateStable
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		err := client.DeleteCluster(cluster1.ID)
		require.NoError(t, err)

		cluster1, err = client.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, model.ClusterStateDeletionRequested, cluster1.State)
	})

	t.Run("while deleting", func(t *testing.T) {
		cluster1.State = model.ClusterStateDeletionRequested
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		err := client.DeleteCluster(cluster1.ID)
		require.NoError(t, err)

		cluster1, err = client.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, model.ClusterStateDeletionRequested, cluster1.State)
	})

	t.Run("while creating", func(t *testing.T) {
		cluster1.State = model.ClusterStateCreationRequested
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		err := client.DeleteCluster(cluster1.ID)
		require.EqualError(t, err, "failed with status code 400")
	})
}
