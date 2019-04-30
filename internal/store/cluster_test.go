package store

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/model"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/stretchr/testify/require"
)

func TestClusters(t *testing.T) {
	t.Run("get unknown cluster", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := MakeTestSQLStore(t, logger)

		cluster, err := sqlStore.GetCluster("unknown")
		require.NoError(t, err)
		require.Nil(t, cluster)
	})

	t.Run("get clusters", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := MakeTestSQLStore(t, logger)

		cluster1 := &model.Cluster{
			Provider:            "aws",
			Provisioner:         "kops",
			ProviderMetadata:    []byte(`{"provider": "test1"}`),
			ProvisionerMetadata: []byte(`{"provisioner": "test1"}`),
			AllowInstallations:  false,
		}

		cluster2 := &model.Cluster{
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
		require.Equal(t, []*model.Cluster{cluster1}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(0, 10, false)
		require.NoError(t, err)
		require.Equal(t, []*model.Cluster{cluster1, cluster2}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(0, 1, true)
		require.NoError(t, err)
		require.Equal(t, []*model.Cluster{cluster1}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(0, 10, true)
		require.NoError(t, err)
		require.Equal(t, []*model.Cluster{cluster1, cluster2}, actualClusters)
	})

	t.Run("update clusters", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := MakeTestSQLStore(t, logger)

		cluster1 := &model.Cluster{
			Provider:            "aws",
			Provisioner:         "kops",
			ProviderMetadata:    []byte(`{"provider": "test1"}`),
			ProvisionerMetadata: []byte(`{"provisioner": "test1"}`),
			AllowInstallations:  false,
		}

		cluster2 := &model.Cluster{
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
		logger := testlib.MakeLogger(t)
		sqlStore := MakeTestSQLStore(t, logger)

		cluster1 := &model.Cluster{
			Provider:            "aws",
			Provisioner:         "kops",
			ProviderMetadata:    []byte(`{"provider": "test1"}`),
			ProvisionerMetadata: []byte(`{"provisioner": "test1"}`),
			AllowInstallations:  false,
		}

		cluster2 := &model.Cluster{
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
		require.Equal(t, []*model.Cluster{cluster2}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(0, 10, false)
		require.NoError(t, err)
		require.Equal(t, []*model.Cluster{cluster2}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(0, 1, true)
		require.NoError(t, err)
		require.Equal(t, []*model.Cluster{cluster1}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(0, 10, true)
		require.NoError(t, err)
		require.Equal(t, []*model.Cluster{cluster1, cluster2}, actualClusters)

		time.Sleep(1 * time.Millisecond)

		// Deleting again shouldn't change timestamp
		err = sqlStore.DeleteCluster(cluster1.ID)
		require.NoError(t, err)

		actualCluster1, err = sqlStore.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, cluster1, actualCluster1)

	})
}

func TestLockCluster(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	cluster1 := &model.Cluster{}
	err := sqlStore.CreateCluster(cluster1)
	require.NoError(t, err)

	cluster2 := &model.Cluster{}
	err = sqlStore.CreateCluster(cluster2)
	require.NoError(t, err)

	t.Run("clusters should start unlocked", func(t *testing.T) {
		cluster1, err = sqlStore.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), cluster1.LockAcquiredAt)
		require.Nil(t, cluster1.LockAcquiredBy)

		cluster2, err = sqlStore.GetCluster(cluster2.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), cluster2.LockAcquiredAt)
		require.Nil(t, cluster2.LockAcquiredBy)
	})

	t.Run("lock an unlocked cluster", func(t *testing.T) {
		locked, err := sqlStore.LockCluster(cluster1.ID)
		require.NoError(t, err)
		require.True(t, locked)

		cluster1, err = sqlStore.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.NotEqual(t, int64(0), cluster1.LockAcquiredAt)
		require.Equal(t, sqlStore.instanceID, *cluster1.LockAcquiredBy)
	})

	t.Run("lock a previously locked cluster", func(t *testing.T) {
		locked, err := sqlStore.LockCluster(cluster1.ID)
		require.NoError(t, err)
		require.False(t, locked)
	})

	t.Run("lock a second cluster", func(t *testing.T) {
		locked, err := sqlStore.LockCluster(cluster2.ID)
		require.NoError(t, err)
		require.True(t, locked)

		cluster2, err = sqlStore.GetCluster(cluster2.ID)
		require.NoError(t, err)
		require.NotEqual(t, int64(0), cluster2.LockAcquiredAt)
		require.Equal(t, sqlStore.instanceID, *cluster2.LockAcquiredBy)
	})

	t.Run("unlock the first cluster", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockCluster(cluster1.ID, false)
		require.NoError(t, err)
		require.True(t, unlocked)

		cluster1, err = sqlStore.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), cluster1.LockAcquiredAt)
		require.Nil(t, cluster1.LockAcquiredBy)
	})

	t.Run("unlock the first cluster again", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockCluster(cluster1.ID, false)
		require.NoError(t, err)
		require.False(t, unlocked)

		cluster1, err = sqlStore.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), cluster1.LockAcquiredAt)
		require.Nil(t, cluster1.LockAcquiredBy)
	})

	t.Run("force unlock the first cluster again", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockCluster(cluster1.ID, true)
		require.NoError(t, err)
		require.False(t, unlocked)

		cluster1, err = sqlStore.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), cluster1.LockAcquiredAt)
		require.Nil(t, cluster1.LockAcquiredBy)
	})

	t.Run("unlock the second cluster from a difference instance", func(t *testing.T) {
		originalInstanceID := sqlStore.instanceID
		defer func() {
			sqlStore.instanceID = originalInstanceID
		}()
		sqlStore.instanceID = "different"

		unlocked, err := sqlStore.UnlockCluster(cluster2.ID, false)
		require.NoError(t, err)
		require.False(t, unlocked)

		cluster2, err = sqlStore.GetCluster(cluster2.ID)
		require.NoError(t, err)
		require.NotEqual(t, int64(0), cluster2.LockAcquiredAt)
		require.Equal(t, originalInstanceID, *cluster2.LockAcquiredBy)
	})

	t.Run("force unlock the second cluster from a difference instance", func(t *testing.T) {
		originalInstanceID := sqlStore.instanceID
		defer func() {
			sqlStore.instanceID = originalInstanceID
		}()
		sqlStore.instanceID = "different"

		unlocked, err := sqlStore.UnlockCluster(cluster2.ID, true)
		require.NoError(t, err)
		require.True(t, unlocked)

		cluster1, err = sqlStore.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), cluster1.LockAcquiredAt)
		require.Nil(t, cluster1.LockAcquiredBy)
	})
}
