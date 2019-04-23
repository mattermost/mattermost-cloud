package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSetProviderMetadata(t *testing.T) {
	t.Run("set nil", func(t *testing.T) {
		cluster := Cluster{}
		err := cluster.SetProviderMetadata(nil)
		require.NoError(t, err)
		require.Nil(t, cluster.ProviderMetadata)
	})

	t.Run("set data", func(t *testing.T) {
		cluster := Cluster{}
		err := cluster.SetProviderMetadata(struct{ Test string }{"test"})
		require.NoError(t, err)
		require.Equal(t, `{"Test":"test"}`, string(cluster.ProviderMetadata))
	})
}

func TestSetProvisionerMetadata(t *testing.T) {
	t.Run("set nil", func(t *testing.T) {
		cluster := Cluster{}
		err := cluster.SetProvisionerMetadata(nil)
		require.NoError(t, err)
		require.Nil(t, cluster.ProvisionerMetadata)
	})

	t.Run("set data", func(t *testing.T) {
		cluster := Cluster{}
		err := cluster.SetProvisionerMetadata(struct{ Test string }{"test"})
		require.NoError(t, err)
		require.Equal(t, `{"Test":"test"}`, string(cluster.ProvisionerMetadata))
	})
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

		time.Sleep(1 * time.Millisecond)

		err = sqlStore.CreateCluster(cluster2)
		require.NoError(t, err)

		actualCluster1, err := sqlStore.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, cluster1, actualCluster1)

		actualCluster2, err := sqlStore.GetCluster(cluster2.ID)
		require.NoError(t, err)
		require.Equal(t, cluster2, actualCluster2)

		actualClusters, err := sqlStore.GetClusters(0, 0, false)
		require.NoError(t, err)
		require.Empty(t, actualClusters)

		actualClusters, err = sqlStore.GetClusters(0, 1, false)
		require.NoError(t, err)
		require.Equal(t, []*Cluster{cluster1}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(0, 10, false)
		require.NoError(t, err)
		require.Equal(t, []*Cluster{cluster1, cluster2}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(0, 1, true)
		require.NoError(t, err)
		require.Equal(t, []*Cluster{cluster1}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(0, 10, true)
		require.NoError(t, err)
		require.Equal(t, []*Cluster{cluster1, cluster2}, actualClusters)
	})

	t.Run("update clusters", func(t *testing.T) {
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

		cluster1.Provider = "azure"
		cluster1.Provisioner = "cluster-api"
		cluster1.ProviderMetadata = []byte(`{"provider": "updated-test1"}`)
		cluster1.ProvisionerMetadata = []byte(`{"provisioner": "updated-test1"}`)
		cluster1.AllowInstallations = true

		err = sqlStore.UpdateCluster(cluster1)
		require.NoError(t, err)

		actualCluster1, err := sqlStore.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, cluster1, actualCluster1)

		actualCluster2, err := sqlStore.GetCluster(cluster2.ID)
		require.NoError(t, err)
		require.Equal(t, cluster2, actualCluster2)
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

		time.Sleep(1 * time.Millisecond)

		err = sqlStore.CreateCluster(cluster2)
		require.NoError(t, err)

		err = sqlStore.DeleteCluster(cluster1.ID)
		require.NoError(t, err)

		actualCluster1, err := sqlStore.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.NotEqual(t, 0, actualCluster1.DeleteAt)
		cluster1.DeleteAt = actualCluster1.DeleteAt
		require.Equal(t, cluster1, actualCluster1)

		actualCluster2, err := sqlStore.GetCluster(cluster2.ID)
		require.NoError(t, err)
		require.Equal(t, cluster2, actualCluster2)

		actualClusters, err := sqlStore.GetClusters(0, 0, false)
		require.NoError(t, err)
		require.Empty(t, actualClusters)

		actualClusters, err = sqlStore.GetClusters(0, 1, false)
		require.NoError(t, err)
		require.Equal(t, []*Cluster{cluster2}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(0, 10, false)
		require.NoError(t, err)
		require.Equal(t, []*Cluster{cluster2}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(0, 1, true)
		require.NoError(t, err)
		require.Equal(t, []*Cluster{cluster1}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(0, 10, true)
		require.NoError(t, err)
		require.Equal(t, []*Cluster{cluster1, cluster2}, actualClusters)

		time.Sleep(1 * time.Millisecond)

		// Deleting again shouldn't change timestamp
		err = sqlStore.DeleteCluster(cluster1.ID)
		require.NoError(t, err)

		actualCluster1, err = sqlStore.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, cluster1, actualCluster1)

	})
}
