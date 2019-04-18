package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func patchClusterTimestamps(cluster *Cluster) {
	// Workaround a Postgres-encoding issue with time.Time: https://github.com/golang/go/issues/11712.
	cluster.CreateAt = cluster.CreateAt.In(time.UTC)
	// Also ignore extra precision on some systems.
	cluster.CreateAt = cluster.CreateAt.Round(time.Microsecond)
}

func assertCluster(t *testing.T, expected *Cluster, actual *Cluster) {
	patchClusterTimestamps(expected)
	patchClusterTimestamps(actual)
	require.Equal(t, expected, actual)
}

func assertClusters(t *testing.T, expected []*Cluster, actual []*Cluster) {
	for _, cluster := range expected {
		patchClusterTimestamps(cluster)
	}
	for _, cluster := range actual {
		patchClusterTimestamps(cluster)
	}
	require.Equal(t, expected, actual)
}

func TestClusters(t *testing.T) {
	t.Run("get unknown cluster", func(t *testing.T) {
		sqlStore := makeSQLStore(t)

		cluster, err := sqlStore.GetCluster("unknown")
		require.NoError(t, err)
		require.Nil(t, cluster)
	})

	t.Run("get clusters", func(t *testing.T) {
		sqlStore := makeSQLStore(t)

		cluster1 := &Cluster{
			Provider:            "aws",
			Provisioner:         "kops",
			ProviderMetadata:    []byte(`{"provider": "test1"}`),
			ProvisionerMetadata: []byte(`{"provisioner": "test1"}`),
			AllowInstallations:  false,
		}

		cluster2 := &Cluster{
			Provider:            "azure",
			Provisioner:         "cluster-api",
			ProviderMetadata:    []byte(`{"provider": "test2"}`),
			ProvisionerMetadata: []byte(`{"provisioner": "test2"}`),
			AllowInstallations:  true,
		}

		err := sqlStore.CreateCluster(cluster1)
		require.NoError(t, err)

		err = sqlStore.CreateCluster(cluster2)
		require.NoError(t, err)

		actualCluster1, err := sqlStore.GetCluster(cluster1.ID)
		require.NoError(t, err)
		assertCluster(t, cluster1, actualCluster1)

		actualCluster2, err := sqlStore.GetCluster(cluster2.ID)
		require.NoError(t, err)
		assertCluster(t, cluster2, actualCluster2)

		actualClusters, err := sqlStore.GetClusters(0, 0, false)
		require.NoError(t, err)
		require.Empty(t, actualClusters)

		actualClusters, err = sqlStore.GetClusters(0, 1, false)
		require.NoError(t, err)
		assertClusters(t, []*Cluster{cluster1}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(0, 10, false)
		require.NoError(t, err)
		assertClusters(t, []*Cluster{cluster1, cluster2}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(0, 1, true)
		require.NoError(t, err)
		assertClusters(t, []*Cluster{cluster1}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(0, 10, true)
		require.NoError(t, err)
		assertClusters(t, []*Cluster{cluster1, cluster2}, actualClusters)
	})

	t.Run("delete cluster", func(t *testing.T) {
		sqlStore := makeSQLStore(t)

		cluster1 := &Cluster{
			Provider:            "aws",
			Provisioner:         "kops",
			ProviderMetadata:    []byte(`{"provider": "test1"}`),
			ProvisionerMetadata: []byte(`{"provisioner": "test1"}`),
			AllowInstallations:  false,
		}

		cluster2 := &Cluster{
			Provider:            "azure",
			Provisioner:         "cluster-api",
			ProviderMetadata:    []byte(`{"provider": "test2"}`),
			ProvisionerMetadata: []byte(`{"provisioner": "test2"}`),
			AllowInstallations:  true,
		}

		err := sqlStore.CreateCluster(cluster1)
		require.NoError(t, err)

		err = sqlStore.CreateCluster(cluster2)
		require.NoError(t, err)

		err = sqlStore.DeleteCluster(cluster1.ID)
		require.NoError(t, err)

		actualCluster1, err := sqlStore.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.NotNil(t, actualCluster1.DeleteAt)
		cluster1.DeleteAt = actualCluster1.DeleteAt
		assertCluster(t, cluster1, actualCluster1)

		actualCluster2, err := sqlStore.GetCluster(cluster2.ID)
		require.NoError(t, err)
		assertCluster(t, cluster2, actualCluster2)

		actualClusters, err := sqlStore.GetClusters(0, 0, false)
		require.NoError(t, err)
		require.Empty(t, actualClusters)

		actualClusters, err = sqlStore.GetClusters(0, 1, false)
		require.NoError(t, err)
		assertClusters(t, []*Cluster{cluster2}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(0, 10, false)
		require.NoError(t, err)
		assertClusters(t, []*Cluster{cluster2}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(0, 1, true)
		require.NoError(t, err)
		assertClusters(t, []*Cluster{cluster1}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(0, 10, true)
		require.NoError(t, err)
		assertClusters(t, []*Cluster{cluster1, cluster2}, actualClusters)
	})
}
