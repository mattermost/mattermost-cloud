package store

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/require"

	mmv1alpha1 "github.com/mattermost/mattermost-operator/pkg/apis/mattermost/v1alpha1"
)

func TestInstallations(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	ownerID1 := model.NewID()
	ownerID2 := model.NewID()
	groupID1 := model.NewID()
	groupID2 := model.NewID()

	installation1 := &model.Installation{
		OwnerID:   ownerID1,
		Version:   "version",
		DNS:       "dns.example.com",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Size:      mmv1alpha1.Size100String,
		Affinity:  model.InstallationAffinityIsolated,
		GroupID:   &groupID1,
		State:     model.InstallationStateCreationRequested,
	}

	err := sqlStore.CreateInstallation(installation1)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	installation2 := &model.Installation{
		OwnerID:   ownerID1,
		Version:   "version2",
		DNS:       "dns2.example.com",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Size:      mmv1alpha1.Size100String,
		Affinity:  model.InstallationAffinityIsolated,
		GroupID:   &groupID2,
		State:     model.InstallationStateStable,
	}

	err = sqlStore.CreateInstallation(installation2)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	installation3 := &model.Installation{
		OwnerID:   ownerID2,
		Version:   "version",
		DNS:       "dns3.example.com",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Size:      mmv1alpha1.Size100String,
		Affinity:  model.InstallationAffinityIsolated,
		GroupID:   &groupID1,
		State:     model.InstallationStateCreationRequested,
	}

	err = sqlStore.CreateInstallation(installation3)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	installation4 := &model.Installation{
		OwnerID:   ownerID2,
		Version:   "version",
		DNS:       "dns4.example.com",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Size:      mmv1alpha1.Size100String,
		Affinity:  model.InstallationAffinityIsolated,
		GroupID:   &groupID2,
		State:     model.InstallationStateCreationRequested,
	}

	err = sqlStore.CreateInstallation(installation4)
	require.NoError(t, err)
	err = sqlStore.DeleteInstallation(installation4.ID)
	require.NoError(t, err)
	installation4, err = sqlStore.GetInstallation(installation4.ID)
	require.NoError(t, err)

	t.Run("get unknown installation", func(t *testing.T) {
		installation, err := sqlStore.GetInstallation("unknown")
		require.NoError(t, err)
		require.Nil(t, installation)
	})

	t.Run("get installation 1", func(t *testing.T) {
		installation, err := sqlStore.GetInstallation(installation1.ID)
		require.NoError(t, err)
		require.Equal(t, installation1, installation)
	})

	t.Run("get installation 2", func(t *testing.T) {
		installation, err := sqlStore.GetInstallation(installation2.ID)
		require.NoError(t, err)
		require.Equal(t, installation2, installation)
	})

	testCases := []struct {
		Description string
		Filter      *model.InstallationFilter
		Expected    []*model.Installation
	}{
		{
			"page 0, perPage 0",
			&model.InstallationFilter{
				Page:           0,
				PerPage:        0,
				IncludeDeleted: false,
			},
			nil,
		},
		{
			"page 0, perPage 1",
			&model.InstallationFilter{
				Page:           0,
				PerPage:        1,
				IncludeDeleted: false,
			},
			[]*model.Installation{installation1},
		},
		{
			"page 0, perPage 10",
			&model.InstallationFilter{
				Page:           0,
				PerPage:        10,
				IncludeDeleted: false,
			},
			[]*model.Installation{installation1, installation2, installation3},
		},
		{
			"page 0, perPage 10, include deleted",
			&model.InstallationFilter{
				Page:           0,
				PerPage:        10,
				IncludeDeleted: true,
			},
			[]*model.Installation{installation1, installation2, installation3, installation4},
		},
		{
			"owner 1",
			&model.InstallationFilter{
				OwnerID:        ownerID1,
				Page:           0,
				PerPage:        10,
				IncludeDeleted: false,
			},
			[]*model.Installation{installation1, installation2},
		},
		{
			"owner 1, include deleted",
			&model.InstallationFilter{
				OwnerID:        ownerID1,
				Page:           0,
				PerPage:        10,
				IncludeDeleted: true,
			},
			[]*model.Installation{installation1, installation2},
		},
		{
			"owner 2",
			&model.InstallationFilter{
				OwnerID:        ownerID2,
				Page:           0,
				PerPage:        10,
				IncludeDeleted: false,
			},
			[]*model.Installation{installation3},
		},
		{
			"owner 2, include deleted",
			&model.InstallationFilter{
				OwnerID:        ownerID2,
				Page:           0,
				PerPage:        10,
				IncludeDeleted: true,
			},
			[]*model.Installation{installation3, installation4},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Description, func(t *testing.T) {
			actual, err := sqlStore.GetInstallations(testCase.Filter)
			require.NoError(t, err)
			require.Equal(t, testCase.Expected, actual)
		})
	}
}

func TestGetUnlockedInstallationPendingWork(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	ownerID := model.NewID()
	groupID := model.NewID()

	creationRequestedInstallation := &model.Installation{
		OwnerID:   ownerID,
		Version:   "version",
		DNS:       "dns1.example.com",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Affinity:  model.InstallationAffinityIsolated,
		GroupID:   &groupID,
		State:     model.InstallationStateCreationRequested,
	}
	err := sqlStore.CreateInstallation(creationRequestedInstallation)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	upgradeRequestedInstallation := &model.Installation{
		OwnerID:  ownerID,
		Version:  "version",
		DNS:      "dns2.example.com",
		Affinity: model.InstallationAffinityIsolated,
		GroupID:  &groupID,
		State:    model.InstallationStateUpgradeRequested,
	}
	err = sqlStore.CreateInstallation(upgradeRequestedInstallation)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	deletionRequestedInstallation := &model.Installation{
		OwnerID:  ownerID,
		Version:  "version",
		DNS:      "dns3.example.com",
		Affinity: model.InstallationAffinityIsolated,
		GroupID:  &groupID,
		State:    model.InstallationStateDeletionRequested,
	}
	err = sqlStore.CreateInstallation(deletionRequestedInstallation)
	require.NoError(t, err)

	otherStates := []string{
		model.InstallationStateCreationFailed,
		model.InstallationStateDeletionFailed,
		model.InstallationStateDeleted,
		model.InstallationStateUpgradeFailed,
		model.InstallationStateStable,
	}

	otherInstallations := []*model.Installation{}
	for _, otherState := range otherStates {
		otherInstallations = append(otherInstallations, &model.Installation{
			State: otherState,
		})
	}

	installations, err := sqlStore.GetUnlockedInstallationsPendingWork()
	require.NoError(t, err)
	require.Equal(t, []*model.Installation{creationRequestedInstallation, upgradeRequestedInstallation, deletionRequestedInstallation}, installations)

	lockerID := model.NewID()

	locked, err := sqlStore.LockInstallation(creationRequestedInstallation.ID, lockerID)
	require.NoError(t, err)
	require.True(t, locked)

	installations, err = sqlStore.GetUnlockedInstallationsPendingWork()
	require.NoError(t, err)
	require.Equal(t, []*model.Installation{upgradeRequestedInstallation, deletionRequestedInstallation}, installations)

	locked, err = sqlStore.LockInstallation(upgradeRequestedInstallation.ID, lockerID)
	require.NoError(t, err)
	require.True(t, locked)

	installations, err = sqlStore.GetUnlockedInstallationsPendingWork()
	require.NoError(t, err)
	require.Equal(t, []*model.Installation{deletionRequestedInstallation}, installations)

	locked, err = sqlStore.LockInstallation(deletionRequestedInstallation.ID, lockerID)
	require.NoError(t, err)
	require.True(t, locked)

	installations, err = sqlStore.GetUnlockedInstallationsPendingWork()
	require.NoError(t, err)
	require.Empty(t, installations)
}

func TestLockInstallation(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	lockerID1 := model.NewID()
	lockerID2 := model.NewID()

	ownerID := model.NewID()

	installation1 := &model.Installation{
		OwnerID: ownerID,
		DNS:     "dns1.example.com",
	}
	err := sqlStore.CreateInstallation(installation1)
	require.NoError(t, err)

	installation2 := &model.Installation{
		OwnerID: ownerID,
		DNS:     "dns2.example.com",
	}
	err = sqlStore.CreateInstallation(installation2)
	require.NoError(t, err)

	t.Run("installations should start unlocked", func(t *testing.T) {
		installation1, err = sqlStore.GetInstallation(installation1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), installation1.LockAcquiredAt)
		require.Nil(t, installation1.LockAcquiredBy)

		installation2, err = sqlStore.GetInstallation(installation2.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), installation2.LockAcquiredAt)
		require.Nil(t, installation2.LockAcquiredBy)
	})

	t.Run("lock an unlocked installation", func(t *testing.T) {
		locked, err := sqlStore.LockInstallation(installation1.ID, lockerID1)
		require.NoError(t, err)
		require.True(t, locked)

		installation1, err = sqlStore.GetInstallation(installation1.ID)
		require.NoError(t, err)
		require.NotEqual(t, int64(0), installation1.LockAcquiredAt)
		require.Equal(t, lockerID1, *installation1.LockAcquiredBy)
	})

	t.Run("lock a previously locked installation", func(t *testing.T) {
		t.Run("by the same locker", func(t *testing.T) {
			locked, err := sqlStore.LockInstallation(installation1.ID, lockerID1)
			require.NoError(t, err)
			require.False(t, locked)
		})

		t.Run("by a different locker", func(t *testing.T) {
			locked, err := sqlStore.LockInstallation(installation1.ID, lockerID2)
			require.NoError(t, err)
			require.False(t, locked)
		})
	})

	t.Run("lock a second installation from a different locker", func(t *testing.T) {
		locked, err := sqlStore.LockInstallation(installation2.ID, lockerID2)
		require.NoError(t, err)
		require.True(t, locked)

		installation2, err = sqlStore.GetInstallation(installation2.ID)
		require.NoError(t, err)
		require.NotEqual(t, int64(0), installation2.LockAcquiredAt)
		require.Equal(t, lockerID2, *installation2.LockAcquiredBy)
	})

	t.Run("unlock the first installation", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockInstallation(installation1.ID, lockerID1, false)
		require.NoError(t, err)
		require.True(t, unlocked)

		installation1, err = sqlStore.GetInstallation(installation1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), installation1.LockAcquiredAt)
		require.Nil(t, installation1.LockAcquiredBy)
	})

	t.Run("unlock the first installation again", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockInstallation(installation1.ID, lockerID1, false)
		require.NoError(t, err)
		require.False(t, unlocked)

		installation1, err = sqlStore.GetInstallation(installation1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), installation1.LockAcquiredAt)
		require.Nil(t, installation1.LockAcquiredBy)
	})

	t.Run("force unlock the first installation again", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockInstallation(installation1.ID, lockerID1, true)
		require.NoError(t, err)
		require.False(t, unlocked)

		installation1, err = sqlStore.GetInstallation(installation1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), installation1.LockAcquiredAt)
		require.Nil(t, installation1.LockAcquiredBy)
	})

	t.Run("unlock the second installation from the wrong locker", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockInstallation(installation2.ID, lockerID1, false)
		require.NoError(t, err)
		require.False(t, unlocked)

		installation2, err = sqlStore.GetInstallation(installation2.ID)
		require.NoError(t, err)
		require.NotEqual(t, int64(0), installation2.LockAcquiredAt)
		require.Equal(t, lockerID2, *installation2.LockAcquiredBy)
	})

	t.Run("force unlock the second installation from the wrong locker", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockInstallation(installation2.ID, lockerID1, true)
		require.NoError(t, err)
		require.True(t, unlocked)

		installation2, err = sqlStore.GetInstallation(installation2.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), installation2.LockAcquiredAt)
		require.Nil(t, installation2.LockAcquiredBy)
	})
}

func TestUpdateInstallation(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	ownerID1 := model.NewID()
	ownerID2 := model.NewID()
	groupID1 := model.NewID()
	groupID2 := model.NewID()

	installation1 := &model.Installation{
		OwnerID:   ownerID1,
		Version:   "version",
		DNS:       "dns3.example.com",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Size:      mmv1alpha1.Size100String,
		Affinity:  model.InstallationAffinityIsolated,
		GroupID:   &groupID1,
		State:     model.InstallationStateCreationRequested,
	}

	err := sqlStore.CreateInstallation(installation1)
	require.NoError(t, err)

	installation2 := &model.Installation{
		OwnerID:   ownerID1,
		Version:   "version2",
		DNS:       "dns4.example.com",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Size:      mmv1alpha1.Size100String,
		Affinity:  model.InstallationAffinityIsolated,
		GroupID:   &groupID2,
		State:     model.InstallationStateStable,
	}

	err = sqlStore.CreateInstallation(installation2)
	require.NoError(t, err)

	installation1.OwnerID = ownerID2
	installation1.Version = "version3"
	installation1.DNS = "dns5.example.com"
	installation1.Size = mmv1alpha1.Size1000String
	installation1.Affinity = model.InstallationAffinityIsolated
	installation1.GroupID = &groupID2
	installation1.State = model.InstallationStateDeletionRequested

	err = sqlStore.UpdateInstallation(installation1)
	require.NoError(t, err)

	actualInstallation1, err := sqlStore.GetInstallation(installation1.ID)
	require.NoError(t, err)
	require.Equal(t, installation1, actualInstallation1)

	actualInstallation2, err := sqlStore.GetInstallation(installation2.ID)
	require.NoError(t, err)
	require.Equal(t, installation2, actualInstallation2)
}

func TestDeleteInstallation(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	ownerID1 := model.NewID()
	ownerID2 := model.NewID()
	groupID1 := model.NewID()
	groupID2 := model.NewID()

	installation1 := &model.Installation{
		OwnerID:   ownerID1,
		Version:   "version",
		DNS:       "dns6.example.com",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Affinity:  model.InstallationAffinityIsolated,
		GroupID:   &groupID1,
		State:     model.InstallationStateCreationRequested,
	}

	err := sqlStore.CreateInstallation(installation1)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	installation2 := &model.Installation{
		OwnerID:   ownerID2,
		Version:   "version2",
		DNS:       "dns7.example.com",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Affinity:  model.InstallationAffinityIsolated,
		GroupID:   &groupID2,
		State:     model.InstallationStateStable,
	}

	err = sqlStore.CreateInstallation(installation2)
	require.NoError(t, err)

	err = sqlStore.DeleteInstallation(installation1.ID)
	require.NoError(t, err)

	actualInstallation1, err := sqlStore.GetInstallation(installation1.ID)
	require.NoError(t, err)
	require.NotEqual(t, 0, actualInstallation1.DeleteAt)
	installation1.DeleteAt = actualInstallation1.DeleteAt
	require.Equal(t, installation1, actualInstallation1)

	actualInstallation2, err := sqlStore.GetInstallation(installation2.ID)
	require.NoError(t, err)
	require.Equal(t, installation2, actualInstallation2)

	time.Sleep(1 * time.Millisecond)

	// Deleting again shouldn't change timestamp
	err = sqlStore.DeleteInstallation(installation1.ID)
	require.NoError(t, err)

	actualInstallation1, err = sqlStore.GetInstallation(installation1.ID)
	require.NoError(t, err)
	require.Equal(t, installation1, actualInstallation1)
}
