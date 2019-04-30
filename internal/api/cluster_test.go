package api_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/internal/api"
	"github.com/mattermost/mattermost-cloud/internal/model"
	"github.com/mattermost/mattermost-cloud/internal/provisioner"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/stretchr/testify/require"
)

func TestClusters(t *testing.T) {
	clusterRootDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)

	mockTerraformCmd := newMockTerraformCmd()

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		SQLStore: sqlStore,
		Provisioner: provisioner.NewKopsProvisioner(
			clusterRootDir,
			"fake-state-store",
			sqlStore,
			func(logger log.FieldLogger) (provisioner.KopsCmd, error) {
				mockKopsCmd, err := newMockKopsCmd()
				require.NoError(t, err)

				return mockKopsCmd, nil
			},
			func(outputDir string, logger log.FieldLogger) provisioner.TerraformCmd {
				return mockTerraformCmd
			},
			logger,
		),
		Logger: logger,
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

		t.Run("upgrade cluster", func(t *testing.T) {
			err := client.UpgradeCluster("unknown", "latest")
			require.EqualError(t, err, "failed with status code 404")
		})

		t.Run("delete cluster", func(t *testing.T) {
			err := client.DeleteCluster("unknown")
			require.EqualError(t, err, "failed with status code 404")
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

	t.Run("get clusters, invalid page", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/api/clusters?page=invalid&per_page=100", ts.URL))
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("get clusters, invalid perPage", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/api/clusters?page=0&per_page=invalid", ts.URL))
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("create cluster, invalid payload", func(t *testing.T) {
		resp, err := http.Post(fmt.Sprintf("%s/api/clusters", ts.URL), "application/json", bytes.NewReader([]byte("invalid")))
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("clusters", func(t *testing.T) {
		cluster1, err := client.CreateCluster(&api.CreateClusterRequest{
			Provider: "aws",
			Size:     "SizeAlef500",
			Zones:    []string{"zone"},
		})
		require.NoError(t, err)
		require.NotNil(t, cluster1)

		actualCluster1, err := client.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, cluster1, actualCluster1)

		time.Sleep(1 * time.Millisecond)

		cluster2, err := client.CreateCluster(&api.CreateClusterRequest{
			Provider: "aws",
			Size:     "SizeAlef500",
			Zones:    []string{"zone"},
		})
		require.NoError(t, err)
		require.NotNil(t, cluster2)

		actualCluster2, err := client.GetCluster(cluster2.ID)
		require.NoError(t, err)
		require.Equal(t, cluster2, actualCluster2)

		time.Sleep(1 * time.Millisecond)

		cluster3, err := client.CreateCluster(&api.CreateClusterRequest{
			Provider: "aws",
			Size:     "SizeAlef500",
			Zones:    []string{"zone"},
		})
		require.NoError(t, err)
		require.NotNil(t, cluster3)

		actualCluster3, err := client.GetCluster(cluster3.ID)
		require.NoError(t, err)
		require.Equal(t, cluster3, actualCluster3)

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
			mockTerraformCmd.MockOutput("cluster_name", fmt.Sprintf("%s-kops.k8s.local", cluster2.ID))
			err = client.DeleteCluster(cluster2.ID)
			require.NoError(t, err)

			cluster2, err = client.GetCluster(cluster2.ID)
			require.NoError(t, err)
			require.NotEqual(t, 0, cluster2.DeleteAt)
		})

		t.Run("get clusters after deletion", func(t *testing.T) {
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

		t.Run("upgrade cluster", func(t *testing.T) {
			mockTerraformCmd.MockOutput("cluster_name", fmt.Sprintf("%s-kops.k8s.local", cluster1.ID))
			err := client.UpgradeCluster(cluster1.ID, "latest")
			require.NoError(t, err)
		})
	})
}
