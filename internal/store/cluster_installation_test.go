// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	mmv1alpha1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1alpha1"
	"github.com/stretchr/testify/require"
)

func TestGetClusterInstallations(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	clusterID1 := model.NewID()
	clusterID2 := model.NewID()
	installationID1 := model.NewID()
	installationID2 := model.NewID()

	clusterInstallation1 := &model.ClusterInstallation{
		ClusterID:      clusterID1,
		InstallationID: installationID1,
		Namespace:      "namespace",
		State:          model.ClusterInstallationStateCreationRequested,
	}

	err := sqlStore.CreateClusterInstallation(clusterInstallation1)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	clusterInstallation2 := &model.ClusterInstallation{
		ClusterID:      clusterID1,
		InstallationID: installationID2,
		Namespace:      "namespace_2",
		State:          model.ClusterInstallationStateStable,
	}

	err = sqlStore.CreateClusterInstallation(clusterInstallation2)
	require.NoError(t, err)

	clusterInstallation3 := &model.ClusterInstallation{
		ClusterID:      clusterID2,
		InstallationID: installationID1,
		Namespace:      "namespace_3",
		State:          model.ClusterInstallationStateStable,
	}

	err = sqlStore.CreateClusterInstallation(clusterInstallation3)
	require.NoError(t, err)

	clusterInstallation4 := &model.ClusterInstallation{
		ClusterID:      clusterID2,
		InstallationID: installationID2,
		Namespace:      "namespace_4",
		State:          model.ClusterInstallationStateStable,
	}

	err = sqlStore.CreateClusterInstallation(clusterInstallation4)
	require.NoError(t, err)
	err = sqlStore.DeleteClusterInstallation(clusterInstallation4.ID)
	require.NoError(t, err)
	clusterInstallation4, err = sqlStore.GetClusterInstallation(clusterInstallation4.ID)
	require.NoError(t, err)

	t.Run("get unknown cluster installation", func(t *testing.T) {
		clusterInstallation, err := sqlStore.GetClusterInstallation("unknown")
		require.NoError(t, err)
		require.Nil(t, clusterInstallation)
	})

	t.Run("get cluster installation 1", func(t *testing.T) {
		clusterInstallation, err := sqlStore.GetClusterInstallation(clusterInstallation1.ID)
		require.NoError(t, err)
		require.Equal(t, clusterInstallation1, clusterInstallation)
	})

	t.Run("get cluster installation 2", func(t *testing.T) {
		clusterInstallation, err := sqlStore.GetClusterInstallation(clusterInstallation2.ID)
		require.NoError(t, err)
		require.Equal(t, clusterInstallation2, clusterInstallation)
	})

	testCases := []struct {
		Description string
		Filter      *model.ClusterInstallationFilter
		Expected    []*model.ClusterInstallation
	}{
		{
			"page 0, perPage 0",
			&model.ClusterInstallationFilter{
				Paging: model.Paging{
					Page:           0,
					PerPage:        0,
					IncludeDeleted: false,
				},
			},
			nil,
		},
		{
			"page 0, perPage 1",
			&model.ClusterInstallationFilter{
				Paging: model.Paging{
					Page:           0,
					PerPage:        1,
					IncludeDeleted: false,
				},
			},
			[]*model.ClusterInstallation{clusterInstallation1},
		},
		{
			"page 0, perPage 10",
			&model.ClusterInstallationFilter{
				Paging: model.Paging{
					Page:           0,
					PerPage:        10,
					IncludeDeleted: false,
				},
			},
			[]*model.ClusterInstallation{clusterInstallation1, clusterInstallation2, clusterInstallation3},
		},
		{
			"page 0, perPage 10, include deleted",
			&model.ClusterInstallationFilter{
				Paging: model.Paging{
					Page:           0,
					PerPage:        10,
					IncludeDeleted: true,
				},
			},
			[]*model.ClusterInstallation{clusterInstallation1, clusterInstallation2, clusterInstallation3, clusterInstallation4},
		},
		{
			"cluster 1",
			&model.ClusterInstallationFilter{
				ClusterID: clusterID1,
				Paging:    model.AllPagesNotDeleted(),
			},
			[]*model.ClusterInstallation{clusterInstallation1, clusterInstallation2},
		},
		{
			"cluster 1, include deleted",
			&model.ClusterInstallationFilter{
				ClusterID: clusterID1,
				Paging:    model.AllPagesWithDeleted(),
			},
			[]*model.ClusterInstallation{clusterInstallation1, clusterInstallation2},
		},
		{
			"cluster 2",
			&model.ClusterInstallationFilter{
				ClusterID: clusterID2,
				Paging:    model.AllPagesNotDeleted(),
			},
			[]*model.ClusterInstallation{clusterInstallation3},
		},
		{
			"cluster 2, include deleted",
			&model.ClusterInstallationFilter{
				ClusterID: clusterID2,
				Paging:    model.AllPagesWithDeleted(),
			},
			[]*model.ClusterInstallation{clusterInstallation3, clusterInstallation4},
		},
		{
			"installation 1",
			&model.ClusterInstallationFilter{
				InstallationID: installationID1,
				Paging:         model.AllPagesNotDeleted(),
			},
			[]*model.ClusterInstallation{clusterInstallation1, clusterInstallation3},
		},
		{
			"installation 1, include deleted",
			&model.ClusterInstallationFilter{
				InstallationID: installationID1,
				Paging:         model.AllPagesWithDeleted(),
			},
			[]*model.ClusterInstallation{clusterInstallation1, clusterInstallation3},
		},
		{
			"installation 2",
			&model.ClusterInstallationFilter{
				InstallationID: installationID2,
				Paging:         model.AllPagesNotDeleted(),
			},
			[]*model.ClusterInstallation{clusterInstallation2},
		},
		{
			"installation 2, include deleted",
			&model.ClusterInstallationFilter{
				InstallationID: installationID2,
				Paging:         model.AllPagesWithDeleted(),
			},
			[]*model.ClusterInstallation{clusterInstallation2, clusterInstallation4},
		},
		{
			"cluster 1 + installation 2",
			&model.ClusterInstallationFilter{
				ClusterID:      clusterID1,
				InstallationID: installationID2,
				Paging:         model.AllPagesNotDeleted(),
			},
			[]*model.ClusterInstallation{clusterInstallation2},
		},
		{
			"cluster installation ids",
			&model.ClusterInstallationFilter{
				IDs:    []string{clusterInstallation1.ID, clusterInstallation4.ID},
				Paging: model.AllPagesNotDeleted(),
			},
			[]*model.ClusterInstallation{clusterInstallation1},
		},
		{
			"cluster installation ids, include deleted",
			&model.ClusterInstallationFilter{
				IDs:    []string{clusterInstallation1.ID, clusterInstallation4.ID},
				Paging: model.AllPagesWithDeleted(),
			},
			[]*model.ClusterInstallation{clusterInstallation1, clusterInstallation4},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Description, func(t *testing.T) {
			actual, err := sqlStore.GetClusterInstallations(testCase.Filter)
			require.NoError(t, err)
			require.Equal(t, testCase.Expected, actual)
		})
	}
}

func TestGetUnlockedClusterInstallationPendingWork(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	clusterID := model.NewID()
	installationID := model.NewID()

	creationRequestedInstallation := &model.ClusterInstallation{
		ClusterID:      clusterID,
		InstallationID: installationID,
		Namespace:      "namespace_1",
		State:          model.ClusterInstallationStateCreationRequested,
	}
	err := sqlStore.CreateClusterInstallation(creationRequestedInstallation)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	deletionRequestedInstallation := &model.ClusterInstallation{
		ClusterID:      clusterID,
		InstallationID: installationID,
		Namespace:      "namespace_3",
		State:          model.ClusterInstallationStateDeletionRequested,
	}
	err = sqlStore.CreateClusterInstallation(deletionRequestedInstallation)
	require.NoError(t, err)

	otherStates := []string{
		model.ClusterInstallationStateCreationFailed,
		model.ClusterInstallationStateDeletionFailed,
		model.ClusterInstallationStateDeleted,
		model.ClusterInstallationStateStable,
	}

	otherClusterInstallations := []*model.ClusterInstallation{}
	for _, otherState := range otherStates {
		otherClusterInstallations = append(otherClusterInstallations, &model.ClusterInstallation{
			State: otherState,
		})
	}

	clusterInstallations, err := sqlStore.GetUnlockedClusterInstallationsPendingWork()
	require.NoError(t, err)
	require.Equal(t, []*model.ClusterInstallation{creationRequestedInstallation, deletionRequestedInstallation}, clusterInstallations)

	lockerID := model.NewID()

	locked, err := sqlStore.LockClusterInstallations([]string{creationRequestedInstallation.ID}, lockerID)
	require.NoError(t, err)
	require.True(t, locked)

	clusterInstallations, err = sqlStore.GetUnlockedClusterInstallationsPendingWork()
	require.NoError(t, err)
	require.Equal(t, []*model.ClusterInstallation{deletionRequestedInstallation}, clusterInstallations)

	locked, err = sqlStore.LockClusterInstallations([]string{deletionRequestedInstallation.ID}, lockerID)
	require.NoError(t, err)
	require.True(t, locked)

	clusterInstallations, err = sqlStore.GetUnlockedClusterInstallationsPendingWork()
	require.NoError(t, err)
	require.Empty(t, clusterInstallations)
}

func TestLockClusterInstallations(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	lockerID1 := model.NewID()
	lockerID2 := model.NewID()

	clusterID := model.NewID()
	installationID := model.NewID()

	clusterInstallation1 := &model.ClusterInstallation{
		ClusterID:      clusterID,
		InstallationID: installationID,
		Namespace:      "namespace_1",
	}
	err := sqlStore.CreateClusterInstallation(clusterInstallation1)
	require.NoError(t, err)

	clusterInstallation2 := &model.ClusterInstallation{
		ClusterID:      clusterID,
		InstallationID: installationID,
		Namespace:      "namespace_2",
	}
	err = sqlStore.CreateClusterInstallation(clusterInstallation2)
	require.NoError(t, err)

	t.Run("installations should start unlocked", func(t *testing.T) {
		clusterInstallation1, err = sqlStore.GetClusterInstallation(clusterInstallation1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), clusterInstallation1.LockAcquiredAt)
		require.Nil(t, clusterInstallation1.LockAcquiredBy)

		clusterInstallation2, err = sqlStore.GetClusterInstallation(clusterInstallation2.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), clusterInstallation2.LockAcquiredAt)
		require.Nil(t, clusterInstallation2.LockAcquiredBy)
	})

	t.Run("lock an unlocked cluster installation", func(t *testing.T) {
		locked, err := sqlStore.LockClusterInstallations([]string{clusterInstallation1.ID}, lockerID1)
		require.NoError(t, err)
		require.True(t, locked)

		clusterInstallation1, err = sqlStore.GetClusterInstallation(clusterInstallation1.ID)
		require.NoError(t, err)
		require.NotEqual(t, int64(0), clusterInstallation1.LockAcquiredAt)
		require.Equal(t, lockerID1, *clusterInstallation1.LockAcquiredBy)
	})

	t.Run("lock a previously locked cluster installation", func(t *testing.T) {
		t.Run("by the same locker", func(t *testing.T) {
			locked, err := sqlStore.LockClusterInstallations([]string{clusterInstallation1.ID}, lockerID1)
			require.NoError(t, err)
			require.False(t, locked)
		})

		t.Run("by a different locker", func(t *testing.T) {
			locked, err := sqlStore.LockClusterInstallations([]string{clusterInstallation1.ID}, lockerID2)
			require.NoError(t, err)
			require.False(t, locked)
		})
	})

	t.Run("lock a second cluster installation from a different locker", func(t *testing.T) {
		locked, err := sqlStore.LockClusterInstallations([]string{clusterInstallation2.ID}, lockerID2)
		require.NoError(t, err)
		require.True(t, locked)

		clusterInstallation2, err = sqlStore.GetClusterInstallation(clusterInstallation2.ID)
		require.NoError(t, err)
		require.NotEqual(t, int64(0), clusterInstallation2.LockAcquiredAt)
		require.Equal(t, lockerID2, *clusterInstallation2.LockAcquiredBy)
	})

	t.Run("unlock the first cluster installation", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockClusterInstallations([]string{clusterInstallation1.ID}, lockerID1, false)
		require.NoError(t, err)
		require.True(t, unlocked)

		clusterInstallation1, err = sqlStore.GetClusterInstallation(clusterInstallation1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), clusterInstallation1.LockAcquiredAt)
		require.Nil(t, clusterInstallation1.LockAcquiredBy)
	})

	t.Run("unlock the first cluster installation again", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockClusterInstallations([]string{clusterInstallation1.ID}, lockerID1, false)
		require.NoError(t, err)
		require.False(t, unlocked)

		clusterInstallation1, err = sqlStore.GetClusterInstallation(clusterInstallation1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), clusterInstallation1.LockAcquiredAt)
		require.Nil(t, clusterInstallation1.LockAcquiredBy)
	})

	t.Run("force unlock the first cluster installation again", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockClusterInstallations([]string{clusterInstallation1.ID}, lockerID1, true)
		require.NoError(t, err)
		require.False(t, unlocked)

		clusterInstallation1, err = sqlStore.GetClusterInstallation(clusterInstallation1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), clusterInstallation1.LockAcquiredAt)
		require.Nil(t, clusterInstallation1.LockAcquiredBy)
	})

	t.Run("unlock the second cluster installation from the wrong locker", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockClusterInstallations([]string{clusterInstallation2.ID}, lockerID1, false)
		require.NoError(t, err)
		require.False(t, unlocked)

		clusterInstallation2, err = sqlStore.GetClusterInstallation(clusterInstallation2.ID)
		require.NoError(t, err)
		require.NotEqual(t, int64(0), clusterInstallation2.LockAcquiredAt)
		require.Equal(t, lockerID2, *clusterInstallation2.LockAcquiredBy)
	})

	t.Run("force unlock the second cluster installation from the wrong locker", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockClusterInstallations([]string{clusterInstallation2.ID}, lockerID1, true)
		require.NoError(t, err)
		require.True(t, unlocked)

		clusterInstallation2, err = sqlStore.GetClusterInstallation(clusterInstallation2.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), clusterInstallation2.LockAcquiredAt)
		require.Nil(t, clusterInstallation2.LockAcquiredBy)
	})

	t.Run("lock multiple rows", func(t *testing.T) {
		locked, err := sqlStore.LockClusterInstallations([]string{clusterInstallation1.ID, clusterInstallation2.ID}, lockerID1)
		require.NoError(t, err)
		require.True(t, locked)

		clusterInstallation1, err = sqlStore.GetClusterInstallation(clusterInstallation1.ID)
		require.NoError(t, err)
		require.NotEqual(t, int64(0), clusterInstallation1.LockAcquiredAt)
		require.Equal(t, lockerID1, *clusterInstallation1.LockAcquiredBy)

		clusterInstallation2, err = sqlStore.GetClusterInstallation(clusterInstallation2.ID)
		require.NoError(t, err)
		require.NotEqual(t, int64(0), clusterInstallation2.LockAcquiredAt)
		require.Equal(t, lockerID1, *clusterInstallation2.LockAcquiredBy)

		require.Equal(t, clusterInstallation1.LockAcquiredAt, clusterInstallation2.LockAcquiredAt)
	})

	t.Run("unlock multiple rows", func(t *testing.T) {
		locked, err := sqlStore.UnlockClusterInstallations([]string{clusterInstallation1.ID, clusterInstallation2.ID}, lockerID1, false)
		require.NoError(t, err)
		require.True(t, locked)

		clusterInstallation1, err = sqlStore.GetClusterInstallation(clusterInstallation1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), clusterInstallation1.LockAcquiredAt)
		require.Nil(t, clusterInstallation1.LockAcquiredBy)

		clusterInstallation2, err = sqlStore.GetClusterInstallation(clusterInstallation2.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), clusterInstallation2.LockAcquiredAt)
		require.Nil(t, clusterInstallation2.LockAcquiredBy)
	})
}

func TestUpdateClusterInstallations(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	clusterID1 := model.NewID()
	installationID1 := model.NewID()
	clusterInstallation1 := &model.ClusterInstallation{
		ClusterID:      clusterID1,
		InstallationID: installationID1,
		Namespace:      "namespace_3",
		State:          model.ClusterInstallationStateCreationRequested,
	}

	time.Sleep(1 * time.Millisecond)

	clusterID2 := model.NewID()
	installationID2 := model.NewID()
	clusterInstallation2 := &model.ClusterInstallation{
		ClusterID:      clusterID1,
		InstallationID: installationID2,
		Namespace:      "namespace_4",
		State:          model.ClusterInstallationStateStable,
	}

	err := sqlStore.CreateClusterInstallation(clusterInstallation1)
	require.NoError(t, err)

	err = sqlStore.CreateClusterInstallation(clusterInstallation2)
	require.NoError(t, err)

	clusterInstallation1.ClusterID = clusterID2
	clusterInstallation1.InstallationID = installationID2
	clusterInstallation1.Namespace = "namespace_5"
	clusterInstallation1.State = model.ClusterInstallationStateDeletionRequested

	err = sqlStore.UpdateClusterInstallation(clusterInstallation1)
	require.NoError(t, err)

	actualClusterInstallation1, err := sqlStore.GetClusterInstallation(clusterInstallation1.ID)
	require.NoError(t, err)
	require.Equal(t, clusterInstallation1, actualClusterInstallation1)

	actualClusterInstallation2, err := sqlStore.GetClusterInstallation(clusterInstallation2.ID)
	require.NoError(t, err)
	require.Equal(t, clusterInstallation2, actualClusterInstallation2)
}

func TestDeleteClusterInstallations(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	clusterID1 := model.NewID()
	clusterID2 := model.NewID()
	installationID1 := model.NewID()
	installationID2 := model.NewID()

	clusterInstallation1 := &model.ClusterInstallation{
		ClusterID:      clusterID1,
		InstallationID: installationID1,
		Namespace:      "namespace_6",
		State:          model.ClusterInstallationStateCreationRequested,
	}

	err := sqlStore.CreateClusterInstallation(clusterInstallation1)
	require.NoError(t, err)

	clusterInstallation2 := &model.ClusterInstallation{
		ClusterID:      clusterID2,
		InstallationID: installationID2,
		Namespace:      "namespace_7",
		State:          model.ClusterInstallationStateStable,
	}

	time.Sleep(1 * time.Millisecond)

	err = sqlStore.CreateClusterInstallation(clusterInstallation2)
	require.NoError(t, err)

	err = sqlStore.DeleteClusterInstallation(clusterInstallation1.ID)
	require.NoError(t, err)

	actualClusterInstallation1, err := sqlStore.GetClusterInstallation(clusterInstallation1.ID)
	require.NoError(t, err)
	require.NotEqual(t, 0, actualClusterInstallation1.DeleteAt)
	clusterInstallation1.DeleteAt = actualClusterInstallation1.DeleteAt
	require.Equal(t, clusterInstallation1, actualClusterInstallation1)

	actualClusterInstallation2, err := sqlStore.GetClusterInstallation(clusterInstallation2.ID)
	require.NoError(t, err)
	require.Equal(t, clusterInstallation2, actualClusterInstallation2)

	time.Sleep(1 * time.Millisecond)

	// Deleting again shouldn't change timestamp
	err = sqlStore.DeleteClusterInstallation(clusterInstallation1.ID)
	require.NoError(t, err)

	actualClusterInstallation1, err = sqlStore.GetClusterInstallation(clusterInstallation1.ID)
	require.NoError(t, err)
	require.Equal(t, clusterInstallation1, actualClusterInstallation1)
}

func TestMigrateSingleClusterInstallation(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	sourceClusterID := model.NewID()
	installationID1 := model.NewID()
	clusterInstallation1 := &model.ClusterInstallation{
		ClusterID:      sourceClusterID,
		InstallationID: installationID1,
		Namespace:      "namespace_8",
		State:          model.ClusterInstallationStateCreationRequested,
	}

	time.Sleep(1 * time.Millisecond)

	targetClusterID := model.NewID()

	err := sqlStore.CreateClusterInstallation(clusterInstallation1)
	require.NoError(t, err)

	clusterInstallations := []*model.ClusterInstallation{clusterInstallation1}
	err = sqlStore.MigrateClusterInstallations(clusterInstallations, targetClusterID)
	require.NoError(t, err)

	migratedClusterInstallations, err := sqlStore.GetClusterInstallations(&model.ClusterInstallationFilter{Paging: model.AllPagesNotDeleted(), ClusterID: targetClusterID})
	require.NoError(t, err)
	require.Equal(t, clusterInstallation1.InstallationID, migratedClusterInstallations[0].InstallationID)
}

func TestMigrateMultipleClusterInstallations(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	sourceClusterID := model.NewID()
	installationID1 := model.NewID()
	clusterInstallation1 := &model.ClusterInstallation{
		ClusterID:      sourceClusterID,
		InstallationID: installationID1,
		Namespace:      "namespace_8",
		State:          model.ClusterInstallationStateCreationRequested,
	}

	time.Sleep(1 * time.Millisecond)

	targetClusterID := model.NewID()
	installationID2 := model.NewID()
	clusterInstallation2 := &model.ClusterInstallation{
		ClusterID:      sourceClusterID,
		InstallationID: installationID2,
		Namespace:      "namespace_9",
		State:          model.ClusterInstallationStateCreationRequested,
	}

	err := sqlStore.CreateClusterInstallation(clusterInstallation1)
	require.NoError(t, err)

	err = sqlStore.CreateClusterInstallation(clusterInstallation2)
	require.NoError(t, err)

	clusterInstallations, err := sqlStore.GetClusterInstallations(&model.ClusterInstallationFilter{Paging: model.AllPagesNotDeleted(), ClusterID: sourceClusterID})
	require.NoError(t, err)

	err = sqlStore.MigrateClusterInstallations(clusterInstallations, targetClusterID)
	require.NoError(t, err)

	migratedClusterInstallations, err := sqlStore.GetClusterInstallations(&model.ClusterInstallationFilter{Paging: model.AllPagesNotDeleted(), ClusterID: targetClusterID})
	require.NoError(t, err)

	for ind, mci := range migratedClusterInstallations {

		require.Equal(t, clusterInstallations[ind].InstallationID, mci.InstallationID)
	}
}

func TestSwitchDNS(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	ownerID1 := model.NewID()
	groupID2 := model.NewID()
	group1 := &model.Group{
		Version: "group1-version",
		Image:   "custom/image",
		MattermostEnv: model.EnvVarMap{
			"Key1": model.EnvVar{Value: "Value1"},
		},
	}
	err := sqlStore.CreateGroup(group1)
	require.NoError(t, err)
	groupID1 := group1.ID

	time.Sleep(1 * time.Millisecond)

	annotations := []*model.Annotation{{Name: "annotation1"}, {Name: "annotation2"}}

	installation1 := &model.Installation{
		OwnerID:   ownerID1,
		Version:   "version",
		DNS:       "dns.example.com",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Size:      mmv1alpha1.Size100String,
		Affinity:  model.InstallationAffinityIsolated,
		GroupID:   &groupID1,
		CRVersion: model.V1betaCRVersion,
		State:     model.InstallationStateCreationRequested,
	}

	err = sqlStore.CreateInstallation(installation1, annotations)
	require.NoError(t, err)
	time.Sleep(1 * time.Millisecond)

	installation2 := &model.Installation{
		OwnerID:   ownerID1,
		Version:   "version2",
		Image:     "custom-image",
		DNS:       "dns2.example.com",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Size:      mmv1alpha1.Size100String,
		Affinity:  model.InstallationAffinityIsolated,
		GroupID:   &groupID2,
		CRVersion: model.DefaultCRVersion,
		State:     model.InstallationStateStable,
	}

	err = sqlStore.CreateInstallation(installation2, nil)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	sourceClusterID := model.NewID()
	clusterInstallation1 := &model.ClusterInstallation{
		ClusterID:      sourceClusterID,
		InstallationID: installation1.ID,
		Namespace:      "namespace_10",
		State:          model.ClusterInstallationStateCreationRequested,
	}

	time.Sleep(1 * time.Millisecond)

	targetClusterID := model.NewID()
	clusterInstallation2 := &model.ClusterInstallation{
		ClusterID:      sourceClusterID,
		InstallationID: installation2.ID,
		Namespace:      "namespace_11",
		State:          model.ClusterInstallationStateCreationRequested,
	}

	err = sqlStore.CreateClusterInstallation(clusterInstallation1)
	require.NoError(t, err)

	err = sqlStore.CreateClusterInstallation(clusterInstallation2)
	require.NoError(t, err)

	clusterInstallations, err := sqlStore.GetClusterInstallations(&model.ClusterInstallationFilter{Paging: model.AllPagesNotDeleted(), ClusterID: sourceClusterID})
	require.NoError(t, err)

	oldCIsIDs := getClusterInstallationIDs(clusterInstallations)
	err = sqlStore.MigrateClusterInstallations(clusterInstallations, targetClusterID)
	require.NoError(t, err)

	var installations []string
	installations = append(installations, installation1.ID)
	installations = append(installations, installation2.ID)
	newCIsIDs := getClusterInstallationIDs(clusterInstallations)
	var installationIDs []string
	for _, ci := range clusterInstallations {
		installationIDs = append(installationIDs, ci.InstallationID)
	}
	err = sqlStore.SwitchDNS(oldCIsIDs, newCIsIDs, installationIDs)
	require.NoError(t, err)
}

func TestDeleteInActiveClusterInstallationsByCluster(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	sourceClusterID := model.NewID()
	installationID1 := model.NewID()
	clusterInstallation1 := &model.ClusterInstallation{
		ClusterID:      sourceClusterID,
		InstallationID: installationID1,
		Namespace:      "namespace_12",
		State:          model.ClusterInstallationStateCreationRequested,
		IsActive:       false,
	}

	time.Sleep(1 * time.Millisecond)

	installationID2 := model.NewID()
	clusterInstallation2 := &model.ClusterInstallation{
		ClusterID:      sourceClusterID,
		InstallationID: installationID2,
		Namespace:      "namespace_13",
		State:          model.ClusterInstallationStateCreationRequested,
		IsActive:       false,
	}

	err := sqlStore.CreateClusterInstallation(clusterInstallation1)
	require.NoError(t, err)

	err = sqlStore.CreateClusterInstallation(clusterInstallation2)
	require.NoError(t, err)

	isActiveClusterInstallations := false
	inActiveClusterInstallations, err := sqlStore.GetClusterInstallations(&model.ClusterInstallationFilter{Paging: model.AllPagesNotDeleted(), ClusterID: sourceClusterID, IsActive: &isActiveClusterInstallations})
	require.NoError(t, err)
	require.NotEmpty(t, inActiveClusterInstallations)

	err = sqlStore.DeleteInActiveClusterInstallationByClusterID(sourceClusterID)
	require.NoError(t, err)

	isActiveClusterInstallations = false
	inActiveClusterInstallations, err = sqlStore.GetClusterInstallations(&model.ClusterInstallationFilter{Paging: model.AllPagesNotDeleted(), ClusterID: sourceClusterID, IsActive: &isActiveClusterInstallations})
	require.NoError(t, err)
	require.Empty(t, inActiveClusterInstallations)
}

func TestDeleteInActiveClusterInstallationsByID(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	sourceClusterID := model.NewID()
	installationID1 := model.NewID()
	clusterInstallation1 := &model.ClusterInstallation{
		ClusterID:      sourceClusterID,
		InstallationID: installationID1,
		Namespace:      "namespace_14",
		State:          model.ClusterInstallationStateCreationRequested,
		IsActive:       false,
	}

	time.Sleep(1 * time.Millisecond)

	err := sqlStore.CreateClusterInstallation(clusterInstallation1)
	require.NoError(t, err)

	isActiveClusterInstallations := false
	filter := &model.ClusterInstallationFilter{
		ClusterID:      sourceClusterID,
		InstallationID: clusterInstallation1.InstallationID,
		Paging:         model.AllPagesNotDeleted(),
		IsActive:       &isActiveClusterInstallations,
	}
	inActiveClusterInstallations, err := sqlStore.GetClusterInstallations(filter)
	require.NoError(t, err)
	require.NotEmpty(t, inActiveClusterInstallations)

	err = sqlStore.DeleteClusterInstallation(clusterInstallation1.ID)
	require.NoError(t, err)

	inActiveClusterInstallations, err = sqlStore.GetClusterInstallations(&model.ClusterInstallationFilter{Paging: model.AllPagesNotDeleted(), ClusterID: sourceClusterID, IsActive: &isActiveClusterInstallations})
	require.NoError(t, err)
	require.Empty(t, inActiveClusterInstallations)
}

func getClusterInstallationIDs(clusterInstallations []*model.ClusterInstallation) []string {
	clusterInstallationIDs := make([]string, 0, len(clusterInstallations))
	for _, clusterInstallation := range clusterInstallations {
		clusterInstallationIDs = append(clusterInstallationIDs, clusterInstallation.ID)
	}
	return clusterInstallationIDs
}
