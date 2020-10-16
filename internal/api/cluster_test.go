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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	client := model.NewClient(ts.URL)

	t.Run("unknown cluster", func(t *testing.T) {
		cluster, err := client.GetCluster(model.NewID())
		require.NoError(t, err)
		require.Nil(t, cluster)
	})

	t.Run("no clusters", func(t *testing.T) {
		clusters, err := client.GetClusters(&model.GetClustersRequest{
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
		cluster1, err := client.CreateCluster(&model.CreateClusterRequest{
			Provider: model.ProviderAWS,
			Zones:    []string{"zone"},
		})
		require.NoError(t, err)
		require.NotNil(t, cluster1)
		require.Equal(t, model.ProviderAWS, cluster1.Provider)
		// require.Equal(t, []string{"zone"}, cluster1.Zones)

		actualCluster1, err := client.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, cluster1.ID, actualCluster1.ID)
		require.Equal(t, model.ProviderAWS, actualCluster1.Provider)
		// require.Equal(t, []string{"zone"}, actualCluster1.Zones)
		require.Equal(t, model.ClusterStateCreationRequested, actualCluster1.State)

		time.Sleep(1 * time.Millisecond)

		cluster2, err := client.CreateCluster(&model.CreateClusterRequest{
			Provider: model.ProviderAWS,
			Zones:    []string{"zone"},
		})
		require.NoError(t, err)
		require.NotNil(t, cluster2)
		require.Equal(t, model.ProviderAWS, cluster2.Provider)
		// require.Equal(t, []string{"zone"}, cluster2.Zones)

		actualCluster2, err := client.GetCluster(cluster2.ID)
		require.NoError(t, err)
		require.Equal(t, cluster2.ID, actualCluster2.ID)
		require.Equal(t, model.ProviderAWS, actualCluster2.Provider)
		// require.Equal(t, []string{"zone"}, actualCluster2.Zones)
		require.Equal(t, model.ClusterStateCreationRequested, actualCluster2.State)

		time.Sleep(1 * time.Millisecond)

		cluster3, err := client.CreateCluster(&model.CreateClusterRequest{
			Provider: model.ProviderAWS,
			Zones:    []string{"zone"},
		})
		require.NoError(t, err)
		require.NotNil(t, cluster3)
		require.Equal(t, model.ProviderAWS, cluster3.Provider)
		// require.Equal(t, []string{"zone"}, cluster3.Zones)

		actualCluster3, err := client.GetCluster(cluster3.ID)
		require.NoError(t, err)
		require.Equal(t, cluster3.ID, actualCluster3.ID)
		require.Equal(t, model.ProviderAWS, actualCluster3.Provider)
		// require.Equal(t, []string{"zone"}, actualCluster3.Zones)
		require.Equal(t, model.ClusterStateCreationRequested, actualCluster3.State)

		t.Run("get clusters, page 0, perPage 2, exclude deleted", func(t *testing.T) {
			clusters, err := client.GetClusters(&model.GetClustersRequest{
				Page:           0,
				PerPage:        2,
				IncludeDeleted: false,
			})
			require.NoError(t, err)
			require.Equal(t, []*model.Cluster{cluster1, cluster2}, clusters)
		})

		t.Run("get clusters, page 1, perPage 2, exclude deleted", func(t *testing.T) {
			clusters, err := client.GetClusters(&model.GetClustersRequest{
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
				clusters, err := client.GetClusters(&model.GetClustersRequest{
					Page:           0,
					PerPage:        2,
					IncludeDeleted: false,
				})
				require.NoError(t, err)
				require.Equal(t, []*model.Cluster{cluster1, cluster2}, clusters)
			})

			t.Run("page 1, perPage 2, exclude deleted", func(t *testing.T) {
				clusters, err := client.GetClusters(&model.GetClustersRequest{
					Page:           1,
					PerPage:        2,
					IncludeDeleted: false,
				})
				require.NoError(t, err)
				require.Equal(t, []*model.Cluster{cluster3}, clusters)
			})

			t.Run("page 0, perPage 2, include deleted", func(t *testing.T) {
				clusters, err := client.GetClusters(&model.GetClustersRequest{
					Page:           0,
					PerPage:        2,
					IncludeDeleted: true,
				})
				require.NoError(t, err)
				require.Equal(t, []*model.Cluster{cluster1, cluster2}, clusters)
			})

			t.Run("page 1, perPage 2, include deleted", func(t *testing.T) {
				clusters, err := client.GetClusters(&model.GetClustersRequest{
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
				clusters, err := client.GetClusters(&model.GetClustersRequest{
					Page:           0,
					PerPage:        2,
					IncludeDeleted: false,
				})
				require.NoError(t, err)
				require.Equal(t, []*model.Cluster{cluster1, cluster3}, clusters)
			})

			t.Run("page 1, perPage 2, exclude deleted", func(t *testing.T) {
				clusters, err := client.GetClusters(&model.GetClustersRequest{
					Page:           1,
					PerPage:        2,
					IncludeDeleted: false,
				})
				require.NoError(t, err)
				require.Equal(t, []*model.Cluster{}, clusters)
			})

			t.Run("page 0, perPage 2, include deleted", func(t *testing.T) {
				clusters, err := client.GetClusters(&model.GetClustersRequest{
					Page:           0,
					PerPage:        2,
					IncludeDeleted: true,
				})
				require.NoError(t, err)
				require.Equal(t, []*model.Cluster{cluster1, cluster2}, clusters)
			})

			t.Run("page 1, perPage 2, include deleted", func(t *testing.T) {
				clusters, err := client.GetClusters(&model.GetClustersRequest{
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

	client := model.NewClient(ts.URL)

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
		_, err := client.CreateCluster(&model.CreateClusterRequest{
			Provider: "invalid",
			Zones:    []string{"zone"},
		})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("valid", func(t *testing.T) {
		cluster, err := client.CreateCluster(&model.CreateClusterRequest{
			Provider: model.ProviderAWS,
			Zones:    []string{"zone"},
		})
		require.NoError(t, err)
		require.Equal(t, model.ProviderAWS, cluster.Provider)
		require.Equal(t, model.ClusterStateCreationRequested, cluster.State)
		// TODO: more fields...
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

	client := model.NewClient(ts.URL)

	cluster1, err := client.CreateCluster(&model.CreateClusterRequest{
		Provider: model.ProviderAWS,
		Zones:    []string{"zone"},
	})
	require.NoError(t, err)

	t.Run("unknown cluster", func(t *testing.T) {
		err := client.RetryCreateCluster(model.NewID())
		require.EqualError(t, err, "failed with status code 404")
	})

	t.Run("while locked", func(t *testing.T) {
		cluster1.State = model.ClusterStateStable
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		lockerID := model.NewID()

		locked, err := sqlStore.LockCluster(cluster1.ID, lockerID)
		require.NoError(t, err)
		require.True(t, locked)
		defer func() {
			unlocked, err := sqlStore.UnlockCluster(cluster1.ID, lockerID, false)
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

func TestProvisionCluster(t *testing.T) {
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

	cluster1, err := client.CreateCluster(&model.CreateClusterRequest{
		Provider: model.ProviderAWS,
		Zones:    []string{"zone"},
	})
	require.NoError(t, err)

	t.Run("unknown cluster", func(t *testing.T) {
		clusterResp, err := client.ProvisionCluster(model.NewID(), nil)
		require.EqualError(t, err, "failed with status code 404")
		assert.Nil(t, clusterResp)
	})

	t.Run("while locked", func(t *testing.T) {
		cluster1.State = model.ClusterStateStable
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		lockerID := model.NewID()

		locked, err := sqlStore.LockCluster(cluster1.ID, lockerID)
		require.NoError(t, err)
		require.True(t, locked)
		defer func() {
			unlocked, err := sqlStore.UnlockCluster(cluster1.ID, lockerID, false)
			require.NoError(t, err)
			require.True(t, unlocked)
		}()

		clusterResp, err := client.ProvisionCluster(cluster1.ID, nil)
		require.EqualError(t, err, "failed with status code 409")
		assert.Nil(t, clusterResp)
	})

	t.Run("while api-security-locked", func(t *testing.T) {
		err = sqlStore.LockClusterAPI(cluster1.ID)
		require.NoError(t, err)

		clusterResp, err := client.ProvisionCluster(cluster1.ID, nil)
		require.EqualError(t, err, "failed with status code 403")
		assert.Nil(t, clusterResp)

		err = sqlStore.UnlockClusterAPI(cluster1.ID)
		require.NoError(t, err)
	})

	t.Run("while provisioning", func(t *testing.T) {
		cluster1.State = model.ClusterStateProvisioningRequested
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		clusterResp, err := client.ProvisionCluster(cluster1.ID, nil)
		require.NoError(t, err)
		assert.NotNil(t, clusterResp)

		cluster1, err = client.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, model.ClusterStateProvisioningRequested, cluster1.State)
	})

	t.Run("after provisioning failed", func(t *testing.T) {
		cluster1.State = model.ClusterStateProvisioningFailed
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		clusterResp, err := client.ProvisionCluster(cluster1.ID, nil)
		require.NoError(t, err)
		assert.NotNil(t, clusterResp)

		cluster1, err = client.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, model.ClusterStateProvisioningRequested, cluster1.State)
	})

	t.Run("while upgrading", func(t *testing.T) {
		cluster1.State = model.ClusterStateUpgradeRequested
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		clusterResp, err := client.ProvisionCluster(cluster1.ID, nil)
		require.EqualError(t, err, "failed with status code 400")
		assert.Nil(t, clusterResp)

		cluster1, err = client.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, model.ClusterStateUpgradeRequested, cluster1.State)
	})

	t.Run("after upgrade failed", func(t *testing.T) {
		cluster1.State = model.ClusterStateUpgradeFailed
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		clusterResp, err := client.ProvisionCluster(cluster1.ID, nil)
		require.EqualError(t, err, "failed with status code 400")
		assert.Nil(t, clusterResp)

		cluster1, err = client.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, model.ClusterStateUpgradeFailed, cluster1.State)
	})

	t.Run("while stable", func(t *testing.T) {
		cluster1.State = model.ClusterStateStable
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		clusterResp, err := client.ProvisionCluster(cluster1.ID, nil)
		require.NoError(t, err)
		assert.NotNil(t, clusterResp)

		cluster1, err = client.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, model.ClusterStateProvisioningRequested, cluster1.State)
	})

	t.Run("while deleting", func(t *testing.T) {
		cluster1.State = model.ClusterStateDeletionRequested
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		clusterResp, err := client.ProvisionCluster(cluster1.ID, nil)
		require.EqualError(t, err, "failed with status code 400")
		assert.Nil(t, clusterResp)
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

	client := model.NewClient(ts.URL)

	cluster1, err := client.CreateCluster(&model.CreateClusterRequest{
		Provider: model.ProviderAWS,
		Zones:    []string{"zone"},
	})
	require.NoError(t, err)

	t.Run("unknown cluster", func(t *testing.T) {
		clusterResp, err := client.UpgradeCluster(model.NewID(), &model.PatchUpgradeClusterRequest{Version: sToP("latest")})
		require.EqualError(t, err, "failed with status code 404")
		assert.Nil(t, clusterResp)
	})

	t.Run("while locked", func(t *testing.T) {
		cluster1.State = model.ClusterStateStable
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		lockerID := model.NewID()

		locked, err := sqlStore.LockCluster(cluster1.ID, lockerID)
		require.NoError(t, err)
		require.True(t, locked)
		defer func() {
			unlocked, err := sqlStore.UnlockCluster(cluster1.ID, lockerID, false)
			require.NoError(t, err)
			require.True(t, unlocked)
		}()

		clusterResp, err := client.UpgradeCluster(cluster1.ID, &model.PatchUpgradeClusterRequest{Version: sToP("latest")})
		require.EqualError(t, err, "failed with status code 409")
		assert.Nil(t, clusterResp)
	})

	t.Run("while api-security-locked", func(t *testing.T) {
		err = sqlStore.LockClusterAPI(cluster1.ID)
		require.NoError(t, err)

		version := "latest"
		clusterResp, err := client.UpgradeCluster(cluster1.ID, &model.PatchUpgradeClusterRequest{Version: &version})
		require.EqualError(t, err, "failed with status code 403")
		assert.Nil(t, clusterResp)

		err = sqlStore.UnlockClusterAPI(cluster1.ID)
		require.NoError(t, err)
	})

	t.Run("while upgrading", func(t *testing.T) {
		cluster1.State = model.ClusterStateUpgradeRequested
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		version := "latest"
		clusterResp, err := client.UpgradeCluster(cluster1.ID, &model.PatchUpgradeClusterRequest{Version: &version})
		require.NoError(t, err)
		assert.NotNil(t, clusterResp)

		cluster1, err = client.GetCluster(cluster1.ID)
		require.NoError(t, err)
		assert.Equal(t, model.ClusterStateUpgradeRequested, cluster1.State)
		assert.Equal(t, version, cluster1.ProvisionerMetadataKops.ChangeRequest.Version)
		assert.Empty(t, cluster1.ProvisionerMetadataKops.AMI)
	})

	t.Run("after upgrade failed", func(t *testing.T) {
		cluster1.State = model.ClusterStateUpgradeFailed
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		version := "latest"
		clusterResp, err := client.UpgradeCluster(cluster1.ID, &model.PatchUpgradeClusterRequest{Version: &version})
		require.NoError(t, err)
		assert.NotNil(t, clusterResp)

		cluster1, err = client.GetCluster(cluster1.ID)
		require.NoError(t, err)
		assert.Equal(t, model.ClusterStateUpgradeRequested, cluster1.State)
		assert.Equal(t, version, cluster1.ProvisionerMetadataKops.ChangeRequest.Version)
		assert.Empty(t, cluster1.ProvisionerMetadataKops.AMI)
	})

	t.Run("while stable, to latest", func(t *testing.T) {
		cluster1.State = model.ClusterStateStable
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		version := "latest"
		clusterResp, err := client.UpgradeCluster(cluster1.ID, &model.PatchUpgradeClusterRequest{Version: &version})
		require.NoError(t, err)
		assert.NotNil(t, clusterResp)

		cluster1, err = client.GetCluster(cluster1.ID)
		require.NoError(t, err)
		assert.Equal(t, model.ClusterStateUpgradeRequested, cluster1.State)
		assert.Equal(t, version, cluster1.ProvisionerMetadataKops.ChangeRequest.Version)
		assert.Empty(t, cluster1.ProvisionerMetadataKops.AMI)
	})

	t.Run("while stable, to valid version", func(t *testing.T) {
		cluster1.State = model.ClusterStateStable
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		version := "1.14.1"
		clusterResp, err := client.UpgradeCluster(cluster1.ID, &model.PatchUpgradeClusterRequest{Version: &version})
		require.NoError(t, err)
		assert.NotNil(t, clusterResp)

		cluster1, err = client.GetCluster(cluster1.ID)
		require.NoError(t, err)
		assert.Equal(t, model.ClusterStateUpgradeRequested, cluster1.State)
		assert.Equal(t, version, cluster1.ProvisionerMetadataKops.ChangeRequest.Version)
		assert.Empty(t, cluster1.ProvisionerMetadataKops.AMI)
	})

	t.Run("while stable, to invalid version", func(t *testing.T) {
		cluster1.State = model.ClusterStateStable
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		clusterResp, err := client.UpgradeCluster(cluster1.ID, &model.PatchUpgradeClusterRequest{Version: sToP("invalid")})
		require.EqualError(t, err, "failed with status code 400")
		assert.Nil(t, clusterResp)
	})

	t.Run("while stable, to valid version and new AMI", func(t *testing.T) {
		cluster1.State = model.ClusterStateStable
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		version := "1.14.1"
		ami := "mattermost-os"
		clusterResp, err := client.UpgradeCluster(cluster1.ID, &model.PatchUpgradeClusterRequest{
			Version: &version,
			KopsAMI: &ami,
		})
		require.NoError(t, err)
		assert.NotNil(t, clusterResp)

		cluster1, err = client.GetCluster(cluster1.ID)
		require.NoError(t, err)
		assert.Equal(t, model.ClusterStateUpgradeRequested, cluster1.State)
		assert.Equal(t, version, cluster1.ProvisionerMetadataKops.ChangeRequest.Version)
		assert.Equal(t, ami, cluster1.ProvisionerMetadataKops.ChangeRequest.AMI)
	})

	t.Run("while deleting", func(t *testing.T) {
		cluster1.State = model.ClusterStateDeletionRequested
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		clusterResp, err := client.UpgradeCluster(cluster1.ID, &model.PatchUpgradeClusterRequest{Version: sToP("latest")})
		require.EqualError(t, err, "failed with status code 400")
		assert.Nil(t, clusterResp)
	})
}

func TestUpdateClusterConfiguration(t *testing.T) {
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

	cluster1, err := client.CreateCluster(&model.CreateClusterRequest{
		Provider:           model.ProviderAWS,
		Zones:              []string{"zone"},
		AllowInstallations: true,
	})
	require.NoError(t, err)

	cluster1.ProvisionerMetadataKops.NodeMinCount = 5
	err = sqlStore.UpdateCluster(cluster1)
	require.NoError(t, err)

	t.Run("unknown cluster", func(t *testing.T) {
		clusterResp, err := client.UpdateCluster(model.NewID(), &model.UpdateClusterRequest{})
		require.EqualError(t, err, "failed with status code 404")
		assert.Nil(t, clusterResp)
	})

	t.Run("while locked", func(t *testing.T) {
		cluster1.State = model.ClusterStateStable
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		lockerID := model.NewID()

		locked, err := sqlStore.LockCluster(cluster1.ID, lockerID)
		require.NoError(t, err)
		require.True(t, locked)
		defer func() {
			unlocked, err := sqlStore.UnlockCluster(cluster1.ID, lockerID, false)
			require.NoError(t, err)
			require.True(t, unlocked)
		}()

		clusterResp, err := client.UpdateCluster(cluster1.ID, &model.UpdateClusterRequest{})
		require.EqualError(t, err, "failed with status code 409")
		assert.Nil(t, clusterResp)
	})

	t.Run("while api-security-locked", func(t *testing.T) {
		err = sqlStore.LockClusterAPI(cluster1.ID)
		require.NoError(t, err)

		clusterResp, err := client.UpdateCluster(cluster1.ID, &model.UpdateClusterRequest{})
		require.EqualError(t, err, "failed with status code 403")
		assert.Nil(t, clusterResp)

		err = sqlStore.UnlockClusterAPI(cluster1.ID)
		require.NoError(t, err)
	})

	t.Run("while stable", func(t *testing.T) {
		cluster1.State = model.ClusterStateStable
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		clusterResp, err := client.UpdateCluster(cluster1.ID, &model.UpdateClusterRequest{AllowInstallations: false})
		require.NoError(t, err)
		assert.NotNil(t, clusterResp)

		cluster1, err = client.GetCluster(cluster1.ID)
		require.NoError(t, err)
		assert.Equal(t, model.ClusterStateStable, cluster1.State)
		assert.False(t, cluster1.AllowInstallations)
	})
}

func TestResizeCluster(t *testing.T) {
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

	cluster1, err := client.CreateCluster(&model.CreateClusterRequest{
		Provider: model.ProviderAWS,
		Zones:    []string{"zone"},
	})
	require.NoError(t, err)

	cluster1.ProvisionerMetadataKops.NodeMinCount = 5
	err = sqlStore.UpdateCluster(cluster1)
	require.NoError(t, err)

	t.Run("unknown cluster", func(t *testing.T) {
		clusterResp, err := client.ResizeCluster(model.NewID(), &model.PatchClusterSizeRequest{})
		require.EqualError(t, err, "failed with status code 404")
		assert.Nil(t, clusterResp)
	})

	t.Run("while locked", func(t *testing.T) {
		cluster1.State = model.ClusterStateStable
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		lockerID := model.NewID()

		locked, err := sqlStore.LockCluster(cluster1.ID, lockerID)
		require.NoError(t, err)
		require.True(t, locked)
		defer func() {
			unlocked, err := sqlStore.UnlockCluster(cluster1.ID, lockerID, false)
			require.NoError(t, err)
			require.True(t, unlocked)
		}()

		clusterResp, err := client.ResizeCluster(cluster1.ID, &model.PatchClusterSizeRequest{})
		require.EqualError(t, err, "failed with status code 409")
		assert.Nil(t, clusterResp)
	})

	t.Run("while api-security-locked", func(t *testing.T) {
		err = sqlStore.LockClusterAPI(cluster1.ID)
		require.NoError(t, err)

		clusterResp, err := client.ResizeCluster(cluster1.ID, &model.PatchClusterSizeRequest{NodeInstanceType: sToP("test1")})
		require.EqualError(t, err, "failed with status code 403")
		assert.Nil(t, clusterResp)

		err = sqlStore.UnlockClusterAPI(cluster1.ID)
		require.NoError(t, err)
	})

	t.Run("while resizing", func(t *testing.T) {
		cluster1.State = model.ClusterStateResizeRequested
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		clusterResp, err := client.ResizeCluster(cluster1.ID, &model.PatchClusterSizeRequest{NodeInstanceType: sToP("test1")})
		require.NoError(t, err)
		assert.NotNil(t, clusterResp)

		cluster1, err = client.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, model.ClusterStateResizeRequested, cluster1.State)
		assert.Equal(t, "test1", cluster1.ProvisionerMetadataKops.ChangeRequest.NodeInstanceType)
	})

	t.Run("after resize failed", func(t *testing.T) {
		cluster1.State = model.ClusterStateResizeFailed
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		clusterResp, err := client.ResizeCluster(cluster1.ID, &model.PatchClusterSizeRequest{NodeInstanceType: sToP("test2")})
		require.NoError(t, err)
		assert.NotNil(t, clusterResp)

		cluster1, err = client.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, model.ClusterStateResizeRequested, cluster1.State)
		assert.Equal(t, "test2", cluster1.ProvisionerMetadataKops.ChangeRequest.NodeInstanceType)
	})

	t.Run("while stable", func(t *testing.T) {
		cluster1.State = model.ClusterStateStable
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		clusterResp, err := client.ResizeCluster(cluster1.ID, &model.PatchClusterSizeRequest{NodeInstanceType: sToP("test3")})
		require.NoError(t, err)
		assert.NotNil(t, clusterResp)

		cluster1, err = client.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, model.ClusterStateResizeRequested, cluster1.State)
		assert.Equal(t, "test3", cluster1.ProvisionerMetadataKops.ChangeRequest.NodeInstanceType)
	})

	t.Run("while stable, to max node count lower than cluster min", func(t *testing.T) {
		cluster1.State = model.ClusterStateStable
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		max := int64(1)
		clusterResp, err := client.ResizeCluster(cluster1.ID, &model.PatchClusterSizeRequest{NodeMaxCount: &max})
		require.EqualError(t, err, "failed with status code 400")
		assert.Nil(t, clusterResp)
	})

	t.Run("while stable, to invalid size", func(t *testing.T) {
		cluster1.State = model.ClusterStateStable
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		min := int64(10)
		max := int64(5)
		clusterResp, err := client.ResizeCluster(cluster1.ID, &model.PatchClusterSizeRequest{NodeMinCount: &min, NodeMaxCount: &max})
		require.EqualError(t, err, "failed with status code 400")
		assert.Nil(t, clusterResp)
	})

	t.Run("while upgrading", func(t *testing.T) {
		cluster1.State = model.ClusterStateUpgradeRequested
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		clusterResp, err := client.ResizeCluster(cluster1.ID, &model.PatchClusterSizeRequest{})
		require.EqualError(t, err, "failed with status code 400")
		assert.Nil(t, clusterResp)
	})

	t.Run("while deleting", func(t *testing.T) {
		cluster1.State = model.ClusterStateDeletionRequested
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		clusterResp, err := client.ResizeCluster(cluster1.ID, &model.PatchClusterSizeRequest{})
		require.EqualError(t, err, "failed with status code 400")
		assert.Nil(t, clusterResp)
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

	client := model.NewClient(ts.URL)

	cluster1, err := client.CreateCluster(&model.CreateClusterRequest{
		Provider: model.ProviderAWS,
		Zones:    []string{"zone"},
	})
	require.NoError(t, err)

	// cluster2 will have a cluster installation running on it
	cluster2, err := client.CreateCluster(&model.CreateClusterRequest{
		Provider: model.ProviderAWS,
		Zones:    []string{"zone"},
	})
	require.NoError(t, err)
	err = sqlStore.CreateClusterInstallation(&model.ClusterInstallation{
		ClusterID:      cluster2.ID,
		InstallationID: model.NewID(),
		State:          model.ClusterInstallationStateStable,
	})
	require.NoError(t, err)

	t.Run("unknown cluster", func(t *testing.T) {
		err := client.DeleteCluster(model.NewID())
		require.EqualError(t, err, "failed with status code 404")
	})

	t.Run("while locked", func(t *testing.T) {
		cluster1.State = model.ClusterStateStable
		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		lockerID := model.NewID()

		locked, err := sqlStore.LockCluster(cluster1.ID, lockerID)
		require.NoError(t, err)
		require.True(t, locked)
		defer func() {
			unlocked, err := sqlStore.UnlockCluster(cluster1.ID, lockerID, false)
			require.NoError(t, err)
			require.True(t, unlocked)

			cluster1, err = client.GetCluster(cluster1.ID)
			require.NoError(t, err)
			require.Equal(t, int64(0), cluster1.LockAcquiredAt)
		}()

		err = client.DeleteCluster(cluster1.ID)
		require.EqualError(t, err, "failed with status code 409")
	})

	t.Run("while api-security-locked", func(t *testing.T) {
		err = sqlStore.LockClusterAPI(cluster1.ID)
		require.NoError(t, err)

		err := client.DeleteCluster(cluster1.ID)
		require.EqualError(t, err, "failed with status code 403")

		err = sqlStore.UnlockClusterAPI(cluster1.ID)
		require.NoError(t, err)
	})

	// valid unlocked states
	states := []string{
		model.ClusterStateStable,
		model.ClusterStateCreationRequested,
		model.ClusterStateCreationFailed,
		model.ClusterStateProvisioningFailed,
		model.ClusterStateUpgradeRequested,
		model.ClusterStateUpgradeFailed,
		model.ClusterStateDeletionRequested,
		model.ClusterStateDeletionFailed,
	}

	t.Run("from a valid, unlocked state", func(t *testing.T) {
		for _, state := range states {
			t.Run(state, func(t *testing.T) {
				cluster1.State = state
				err = sqlStore.UpdateCluster(cluster1)
				require.NoError(t, err)

				err = client.DeleteCluster(cluster1.ID)
				require.NoError(t, err)

				cluster1, err = client.GetCluster(cluster1.ID)
				require.NoError(t, err)
				require.Equal(t, model.ClusterStateDeletionRequested, cluster1.State)
			})
		}
	})

	t.Run("from a valid, unlocked state, but not empty of cluster installations", func(t *testing.T) {
		for _, state := range states {
			t.Run(state, func(t *testing.T) {
				cluster2.State = state
				err = sqlStore.UpdateCluster(cluster2)
				require.NoError(t, err)

				err = client.DeleteCluster(cluster2.ID)
				require.Error(t, err)

				cluster2, err = client.GetCluster(cluster2.ID)
				require.NoError(t, err)
				require.Equal(t, state, cluster2.State)
			})
		}
	})
}

func TestGetAllUtilityMetadata(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})

	ts := httptest.NewServer(router)
	client := model.NewClient(ts.URL)
	c, err := client.CreateCluster(
		&model.CreateClusterRequest{
			Provider: model.ProviderAWS,
			Zones:    []string{"zone"},
			DesiredUtilityVersions: map[string]string{
				"prometheus":          "10.3.0",
				"prometheus-operator": "9.4.4",
				"thanos":              "2.4.3",
				"nginx":               "stable",
			},
		})

	require.NoError(t, err)
	utilityMetadata, err := client.GetClusterUtilities(c.ID)

	assert.Equal(t, "", utilityMetadata.ActualVersions.Prometheus)
	assert.Equal(t, "", utilityMetadata.ActualVersions.PrometheusOperator)
	assert.Equal(t, "", utilityMetadata.ActualVersions.Thanos)
	assert.Equal(t, "", utilityMetadata.ActualVersions.Nginx)
	assert.Equal(t, "", utilityMetadata.ActualVersions.Fluentbit)

	assert.Equal(t, "", utilityMetadata.DesiredVersions.Nginx)
	assert.Equal(t, "10.3.0", utilityMetadata.DesiredVersions.Prometheus)
	assert.Equal(t, "9.4.4", utilityMetadata.DesiredVersions.PrometheusOperator)
	assert.Equal(t, "2.4.3", utilityMetadata.DesiredVersions.Thanos)
	assert.Equal(t, model.FluentbitDefaultVersion, utilityMetadata.DesiredVersions.Fluentbit)
}
