package store

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
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
			Provider:                "aws",
			Provisioner:             "kops",
			ProviderMetadataAWS:     &model.AWSMetadata{Zones: []string{"zone1"}},
			ProvisionerMetadataKops: &model.KopsMetadata{Version: "version1"},
			UtilityMetadata:         &model.UtilityMetadata{},
			State:                   model.ClusterStateCreationRequested,
			AllowInstallations:      false,
		}

		cluster2 := &model.Cluster{
			Provider:                "azure",
			Provisioner:             "cluster-api",
			ProviderMetadataAWS:     &model.AWSMetadata{Zones: []string{"zone1"}},
			ProvisionerMetadataKops: &model.KopsMetadata{Version: "version1"},
			UtilityMetadata:         &model.UtilityMetadata{},
			State:                   model.ClusterStateStable,
			AllowInstallations:      true,
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

		actualClusters, err := sqlStore.GetClusters(&model.ClusterFilter{Page: 0, PerPage: 0, IncludeDeleted: false})
		require.NoError(t, err)
		require.Empty(t, actualClusters)

		actualClusters, err = sqlStore.GetClusters(&model.ClusterFilter{Page: 0, PerPage: 1, IncludeDeleted: false})
		require.NoError(t, err)
		require.Equal(t, []*model.Cluster{cluster1}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(&model.ClusterFilter{Page: 0, PerPage: 10, IncludeDeleted: false})
		require.NoError(t, err)
		require.Equal(t, []*model.Cluster{cluster1, cluster2}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(&model.ClusterFilter{Page: 0, PerPage: 1, IncludeDeleted: true})
		require.NoError(t, err)
		require.Equal(t, []*model.Cluster{cluster1}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(&model.ClusterFilter{Page: 0, PerPage: 10, IncludeDeleted: true})
		require.NoError(t, err)
		require.Equal(t, []*model.Cluster{cluster1, cluster2}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(&model.ClusterFilter{PerPage: model.AllPerPage, IncludeDeleted: true})
		require.NoError(t, err)
		require.Equal(t, []*model.Cluster{cluster1, cluster2}, actualClusters)
	})

	t.Run("update clusters", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := MakeTestSQLStore(t, logger)

		cluster1 := &model.Cluster{
			Provider:                "aws",
			Provisioner:             "kops",
			ProviderMetadataAWS:     &model.AWSMetadata{Zones: []string{"zone1"}},
			ProvisionerMetadataKops: &model.KopsMetadata{Version: "version1"},
			UtilityMetadata:         &model.UtilityMetadata{},
			State:                   model.ClusterStateCreationRequested,
			AllowInstallations:      false,
		}

		cluster2 := &model.Cluster{
			Provider:                "azure",
			Provisioner:             "cluster-api",
			ProviderMetadataAWS:     &model.AWSMetadata{Zones: []string{"zone1"}},
			ProvisionerMetadataKops: &model.KopsMetadata{Version: "version1"},
			UtilityMetadata:         &model.UtilityMetadata{},
			State:                   model.ClusterStateStable,
			AllowInstallations:      true,
		}

		err := sqlStore.CreateCluster(cluster1)
		require.NoError(t, err)

		err = sqlStore.CreateCluster(cluster2)
		require.NoError(t, err)

		cluster1.Provider = "azure"
		cluster1.Provisioner = "cluster-api"
		cluster1.ProviderMetadataAWS = &model.AWSMetadata{Zones: []string{"zone2"}}
		cluster1.ProvisionerMetadataKops = &model.KopsMetadata{Version: "version2"}
		cluster1.State = model.ClusterStateDeletionRequested
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
			Provider:                "aws",
			Provisioner:             "kops",
			ProviderMetadataAWS:     &model.AWSMetadata{Zones: []string{"zone1"}},
			ProvisionerMetadataKops: &model.KopsMetadata{Version: "version1"},
			UtilityMetadata:         &model.UtilityMetadata{},
			AllowInstallations:      false,
		}

		cluster2 := &model.Cluster{
			Provider:                "azure",
			Provisioner:             "cluster-api",
			ProviderMetadataAWS:     &model.AWSMetadata{Zones: []string{"zone1"}},
			ProvisionerMetadataKops: &model.KopsMetadata{Version: "version1"},
			UtilityMetadata:         &model.UtilityMetadata{},
			AllowInstallations:      true,
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

		actualClusters, err := sqlStore.GetClusters(&model.ClusterFilter{Page: 0, PerPage: 0, IncludeDeleted: false})
		require.NoError(t, err)
		require.Empty(t, actualClusters)

		actualClusters, err = sqlStore.GetClusters(&model.ClusterFilter{Page: 0, PerPage: 1, IncludeDeleted: false})
		require.NoError(t, err)
		require.Equal(t, []*model.Cluster{cluster2}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(&model.ClusterFilter{Page: 0, PerPage: 10, IncludeDeleted: false})
		require.NoError(t, err)
		require.Equal(t, []*model.Cluster{cluster2}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(&model.ClusterFilter{Page: 0, PerPage: 1, IncludeDeleted: true})
		require.NoError(t, err)
		require.Equal(t, []*model.Cluster{cluster1}, actualClusters)

		actualClusters, err = sqlStore.GetClusters(&model.ClusterFilter{Page: 0, PerPage: 10, IncludeDeleted: true})
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

func TestGetUnlockedClustersPendingWork(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	creationRequestedCluster := &model.Cluster{
		State: model.ClusterStateCreationRequested,
	}
	err := sqlStore.CreateCluster(creationRequestedCluster)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	upgradeRequestedCluster := &model.Cluster{
		State: model.ClusterStateUpgradeRequested,
	}
	err = sqlStore.CreateCluster(upgradeRequestedCluster)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	deletionRequestedCluster := &model.Cluster{
		State: model.ClusterStateDeletionRequested,
	}
	err = sqlStore.CreateCluster(deletionRequestedCluster)
	require.NoError(t, err)

	// Store clusters with states that should be ignored by GetUnlockedClustersPendingWork()
	otherStates := []string{
		model.ClusterStateCreationFailed,
		model.ClusterStateProvisioningFailed,
		model.ClusterStateDeletionFailed,
		model.ClusterStateDeleted,
		model.ClusterStateUpgradeFailed,
		model.ClusterStateStable,
	}
	for _, otherState := range otherStates {
		err = sqlStore.CreateCluster(&model.Cluster{State: otherState})
		require.NoError(t, err)
	}

	clusters, err := sqlStore.GetUnlockedClustersPendingWork()
	require.NoError(t, err)
	require.Equal(t, []*model.Cluster{creationRequestedCluster, upgradeRequestedCluster, deletionRequestedCluster}, clusters)

	lockerID := model.NewID()

	locked, err := sqlStore.LockCluster(creationRequestedCluster.ID, lockerID)
	require.NoError(t, err)
	require.True(t, locked)

	clusters, err = sqlStore.GetUnlockedClustersPendingWork()
	require.NoError(t, err)
	require.Equal(t, []*model.Cluster{upgradeRequestedCluster, deletionRequestedCluster}, clusters)

	locked, err = sqlStore.LockCluster(upgradeRequestedCluster.ID, lockerID)
	require.NoError(t, err)
	require.True(t, locked)

	clusters, err = sqlStore.GetUnlockedClustersPendingWork()
	require.NoError(t, err)
	require.Equal(t, []*model.Cluster{deletionRequestedCluster}, clusters)

	locked, err = sqlStore.LockCluster(deletionRequestedCluster.ID, lockerID)
	require.NoError(t, err)
	require.True(t, locked)

	clusters, err = sqlStore.GetUnlockedClustersPendingWork()
	require.NoError(t, err)
	require.Empty(t, clusters)
}

func TestLockCluster(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	lockerID1 := model.NewID()
	lockerID2 := model.NewID()

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
		locked, err := sqlStore.LockCluster(cluster1.ID, lockerID1)
		require.NoError(t, err)
		require.True(t, locked)

		cluster1, err = sqlStore.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.NotEqual(t, int64(0), cluster1.LockAcquiredAt)
		require.Equal(t, lockerID1, *cluster1.LockAcquiredBy)
	})

	t.Run("lock a previously locked cluster", func(t *testing.T) {
		t.Run("by the same locker", func(t *testing.T) {
			locked, err := sqlStore.LockCluster(cluster1.ID, lockerID1)
			require.NoError(t, err)
			require.False(t, locked)
		})

		t.Run("by a different locker", func(t *testing.T) {
			locked, err := sqlStore.LockCluster(cluster1.ID, lockerID2)
			require.NoError(t, err)
			require.False(t, locked)
		})
	})

	t.Run("lock a second cluster from a different locker", func(t *testing.T) {
		locked, err := sqlStore.LockCluster(cluster2.ID, lockerID2)
		require.NoError(t, err)
		require.True(t, locked)

		cluster2, err = sqlStore.GetCluster(cluster2.ID)
		require.NoError(t, err)
		require.NotEqual(t, int64(0), cluster2.LockAcquiredAt)
		require.Equal(t, lockerID2, *cluster2.LockAcquiredBy)
	})

	t.Run("unlock the first cluster", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockCluster(cluster1.ID, lockerID1, false)
		require.NoError(t, err)
		require.True(t, unlocked)

		cluster1, err = sqlStore.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), cluster1.LockAcquiredAt)
		require.Nil(t, cluster1.LockAcquiredBy)
	})

	t.Run("unlock the first cluster again", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockCluster(cluster1.ID, lockerID1, false)
		require.NoError(t, err)
		require.False(t, unlocked)

		cluster1, err = sqlStore.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), cluster1.LockAcquiredAt)
		require.Nil(t, cluster1.LockAcquiredBy)
	})

	t.Run("force unlock the first cluster again", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockCluster(cluster1.ID, lockerID1, true)
		require.NoError(t, err)
		require.False(t, unlocked)

		cluster1, err = sqlStore.GetCluster(cluster1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), cluster1.LockAcquiredAt)
		require.Nil(t, cluster1.LockAcquiredBy)
	})

	t.Run("unlock the second cluster from the wrong locker", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockCluster(cluster2.ID, lockerID1, false)
		require.NoError(t, err)
		require.False(t, unlocked)

		cluster2, err = sqlStore.GetCluster(cluster2.ID)
		require.NoError(t, err)
		require.NotEqual(t, int64(0), cluster2.LockAcquiredAt)
		require.Equal(t, lockerID2, *cluster2.LockAcquiredBy)
	})

	t.Run("force unlock the second cluster from the wrong locker", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockCluster(cluster2.ID, lockerID1, true)
		require.NoError(t, err)
		require.True(t, unlocked)

		cluster2, err = sqlStore.GetCluster(cluster2.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), cluster2.LockAcquiredAt)
		require.Nil(t, cluster2.LockAcquiredBy)
	})
}
