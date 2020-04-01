package store

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	mmv1alpha1 "github.com/mattermost/mattermost-operator/pkg/apis/mattermost/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestGroups(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	someBool := false
	mattermostEnv := model.EnvVarMap{
		"Var1": model.EnvVar{
			Value: "Var1Value",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "1",
					FieldPath:  "some/path/neat",
				},

				// TODO @gigawhitlocks 2/17/2020

				// The ResourceFieldRef member below, for some reason, breaks
				// testify's Equal() comparator, which in turn causes this
				// test to fail. Investigate this later so that this test can
				// include the entirety of the EnvVarSource type.

				// ResourceFieldRef: &corev1.ResourceFieldSelector{
				//	ContainerName: "someContainer",
				//	Resource:      "someResource",
				//	Divisor: k8sResource.Quantity{
				//		Format: "some_format",
				//	},
				// },

				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					Key:      "key_string",
					Optional: &someBool,
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "configMap_localObjectReference",
					},
				},
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "secret_localObjectReference",
					},
					Key:      "key_secret",
					Optional: &someBool,
				},
			},
		},
	}

	group1 := &model.Group{
		Name:          "name1",
		Description:   "description1",
		Version:       "version1",
		MattermostEnv: mattermostEnv,
	}

	err := sqlStore.CreateGroup(group1)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	group2 := &model.Group{
		Name:        "name2",
		Description: "description2",
		Version:     "version2",
	}

	err = sqlStore.CreateGroup(group2)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	group3 := &model.Group{
		Name:        "name3",
		Description: "description3",
		Version:     "version3",
	}

	err = sqlStore.CreateGroup(group3)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	group4 := &model.Group{
		Name:        "name4",
		Description: "description4",
		Version:     "version4",
	}

	err = sqlStore.CreateGroup(group4)
	require.NoError(t, err)
	err = sqlStore.DeleteGroup(group4.ID)
	require.NoError(t, err)
	group4, err = sqlStore.GetGroup(group4.ID)
	require.NoError(t, err)

	t.Run("get unknown group", func(t *testing.T) {
		group, err := sqlStore.GetGroup("unknown")
		require.NoError(t, err)
		require.Nil(t, group)
	})

	t.Run("get group 1", func(t *testing.T) {
		group, err := sqlStore.GetGroup(group1.ID)
		require.NoError(t, err)
		assert.Equal(t, group1, group)
	})

	t.Run("get group 2", func(t *testing.T) {
		group, err := sqlStore.GetGroup(group2.ID)
		require.NoError(t, err)
		assert.Equal(t, group2, group)
	})

	testCases := []struct {
		Description string
		Filter      *model.GroupFilter
		Expected    []*model.Group
	}{
		{
			"page 0, perPage 0",
			&model.GroupFilter{
				Page:           0,
				PerPage:        0,
				IncludeDeleted: false,
			},
			nil,
		},
		{
			"page 0, perPage 1",
			&model.GroupFilter{
				Page:           0,
				PerPage:        1,
				IncludeDeleted: false,
			},
			[]*model.Group{group1},
		},
		{
			"page 0, perPage 10",
			&model.GroupFilter{
				Page:           0,
				PerPage:        10,
				IncludeDeleted: false,
			},
			[]*model.Group{group1, group2, group3},
		},
		{
			"page 0, perPage 10, include deleted",
			&model.GroupFilter{
				Page:           0,
				PerPage:        10,
				IncludeDeleted: true,
			},
			[]*model.Group{group1, group2, group3, group4},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Description, func(t *testing.T) {
			actual, err := sqlStore.GetGroups(testCase.Filter)
			require.NoError(t, err)
			assert.Equal(t, testCase.Expected, actual)
		})
	}
}

func TestLockGroup(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	lockerID1 := model.NewID()
	lockerID2 := model.NewID()

	group1 := &model.Group{
		Name: "group1",
	}
	err := sqlStore.CreateGroup(group1)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	group2 := &model.Group{
		Name: "group2",
	}
	err = sqlStore.CreateGroup(group2)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	t.Run("groups should start unlocked", func(t *testing.T) {
		group1, err = sqlStore.GetGroup(group1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), group1.LockAcquiredAt)
		require.Nil(t, group1.LockAcquiredBy)

		group2, err = sqlStore.GetGroup(group2.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), group2.LockAcquiredAt)
		require.Nil(t, group2.LockAcquiredBy)
	})

	t.Run("lock an unlocked group", func(t *testing.T) {
		locked, err := sqlStore.LockGroup(group1.ID, lockerID1)
		require.NoError(t, err)
		require.True(t, locked)

		group1, err = sqlStore.GetGroup(group1.ID)
		require.NoError(t, err)
		require.NotEqual(t, int64(0), group1.LockAcquiredAt)
		require.Equal(t, lockerID1, *group1.LockAcquiredBy)
	})

	t.Run("lock a previously locked group", func(t *testing.T) {
		t.Run("by the same locker", func(t *testing.T) {
			locked, err := sqlStore.LockGroup(group1.ID, lockerID1)
			require.NoError(t, err)
			require.False(t, locked)
		})

		t.Run("by a different locker", func(t *testing.T) {
			locked, err := sqlStore.LockGroup(group1.ID, lockerID2)
			require.NoError(t, err)
			require.False(t, locked)
		})
	})

	t.Run("lock a second group from a different locker", func(t *testing.T) {
		locked, err := sqlStore.LockGroup(group2.ID, lockerID2)
		require.NoError(t, err)
		require.True(t, locked)

		group2, err = sqlStore.GetGroup(group2.ID)
		require.NoError(t, err)
		require.NotEqual(t, int64(0), group2.LockAcquiredAt)
		require.Equal(t, lockerID2, *group2.LockAcquiredBy)
	})

	t.Run("unlock the first group", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockGroup(group1.ID, lockerID1, false)
		require.NoError(t, err)
		require.True(t, unlocked)

		group1, err = sqlStore.GetGroup(group1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), group1.LockAcquiredAt)
		require.Nil(t, group1.LockAcquiredBy)
	})

	t.Run("unlock the first group again", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockGroup(group1.ID, lockerID1, false)
		require.NoError(t, err)
		require.False(t, unlocked)

		group1, err = sqlStore.GetGroup(group1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), group1.LockAcquiredAt)
		require.Nil(t, group1.LockAcquiredBy)
	})

	t.Run("force unlock the first group again", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockGroup(group1.ID, lockerID1, true)
		require.NoError(t, err)
		require.False(t, unlocked)

		group1, err = sqlStore.GetGroup(group1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), group1.LockAcquiredAt)
		require.Nil(t, group1.LockAcquiredBy)
	})

	t.Run("unlock the second group from the wrong locker", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockGroup(group2.ID, lockerID1, false)
		require.NoError(t, err)
		require.False(t, unlocked)

		group2, err = sqlStore.GetGroup(group2.ID)
		require.NoError(t, err)
		require.NotEqual(t, int64(0), group2.LockAcquiredAt)
		require.Equal(t, lockerID2, *group2.LockAcquiredBy)
	})

	t.Run("force unlock the second group from the wrong locker", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockGroup(group2.ID, lockerID1, true)
		require.NoError(t, err)
		require.True(t, unlocked)

		group2, err = sqlStore.GetGroup(group2.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), group2.LockAcquiredAt)
		require.Nil(t, group2.LockAcquiredBy)
	})
}

// TODO: better group checks than simply checking with Len().
func TestGetUnlockedGroupsPendingWork(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	group1 := &model.Group{
		Name:        "group1",
		Description: "description1",
		Version:     "version1",
	}

	err := sqlStore.CreateGroup(group1)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	group2 := &model.Group{
		Name:        "group2",
		Description: "description2",
		Version:     "version2",
	}

	err = sqlStore.CreateGroup(group2)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	installation1 := &model.Installation{
		OwnerID:   model.NewID(),
		GroupID:   &group1.ID,
		Version:   "version",
		DNS:       "dns.example.com",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Size:      mmv1alpha1.Size100String,
		Affinity:  model.InstallationAffinityIsolated,
		State:     model.InstallationStateCreationRequested,
	}

	err = sqlStore.CreateInstallation(installation1)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	groups, err := sqlStore.GetUnlockedGroupsPendingWork()
	require.NoError(t, err)
	require.Len(t, groups, 1)

	group1.Version = "new-version"
	err = sqlStore.UpdateGroup(group1)
	require.NoError(t, err)

	groups, err = sqlStore.GetUnlockedGroupsPendingWork()
	require.NoError(t, err)
	require.Len(t, groups, 1)

	lockerID := model.NewID()

	locked, err := sqlStore.LockGroup(group1.ID, lockerID)
	require.NoError(t, err)
	require.True(t, locked)

	groups, err = sqlStore.GetUnlockedGroupsPendingWork()
	require.NoError(t, err)
	require.Len(t, groups, 0)

	locked, err = sqlStore.UnlockGroup(group1.ID, lockerID, false)
	require.NoError(t, err)
	require.True(t, locked)

	groups, err = sqlStore.GetUnlockedGroupsPendingWork()
	require.NoError(t, err)
	require.Len(t, groups, 1)
}

func TestGetGroupRollingMetadata(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	group1 := &model.Group{
		Name:        "group1",
		Description: "description1",
		Version:     "version1",
		Sequence:    2,
	}

	err := sqlStore.CreateGroup(group1)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	group2 := &model.Group{
		Name:        "group2",
		Description: "description2",
		Version:     "version2",
	}

	err = sqlStore.CreateGroup(group2)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	installation1 := &model.Installation{
		OwnerID:   model.NewID(),
		GroupID:   &group1.ID,
		Version:   "version",
		DNS:       "dns.example.com",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Size:      mmv1alpha1.Size100String,
		Affinity:  model.InstallationAffinityIsolated,
		State:     model.InstallationStateCreationRequested,
	}

	err = sqlStore.CreateInstallation(installation1)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	t.Run("empty group", func(t *testing.T) {
		expectedMetadata := &GroupRollingMetadata{
			InstallationIDsToBeRolled:  []string{},
			InstallationTotalCount:     0,
			InstallationStableCount:    0,
			InstallationNonStableCount: 0,
		}
		metadata, err := sqlStore.GetGroupRollingMetadata(group2.ID)
		require.NoError(t, err)
		assert.Equal(t, expectedMetadata, metadata)
	})

	t.Run("group with work", func(t *testing.T) {
		groups, err := sqlStore.GetUnlockedGroupsPendingWork()
		require.NoError(t, err)
		require.Len(t, groups, 1)

		expectedMetadata := &GroupRollingMetadata{
			InstallationIDsToBeRolled:  []string{},
			InstallationTotalCount:     1,
			InstallationStableCount:    0,
			InstallationNonStableCount: 1,
		}
		metadata, err := sqlStore.GetGroupRollingMetadata(group1.ID)
		require.NoError(t, err)
		assert.Equal(t, expectedMetadata, metadata)

		installation1.State = model.InstallationStateStable
		err = sqlStore.UpdateInstallation(installation1)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		expectedMetadata = &GroupRollingMetadata{
			InstallationIDsToBeRolled:  []string{installation1.ID},
			InstallationTotalCount:     1,
			InstallationStableCount:    1,
			InstallationNonStableCount: 0,
		}
		metadata, err = sqlStore.GetGroupRollingMetadata(group1.ID)
		require.NoError(t, err)
		assert.Equal(t, expectedMetadata, metadata)
	})
}

func TestUpdateGroup(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	group1 := &model.Group{
		Name:        "name1",
		Description: "description1",
		Version:     "version1",
	}

	err := sqlStore.CreateGroup(group1)
	require.NoError(t, err)

	group2 := &model.Group{
		Name:        "name2",
		Description: "description2",
		Version:     "version2",
	}

	err = sqlStore.CreateGroup(group2)
	require.NoError(t, err)

	group1.Name = "name3"
	group1.Description = "description3"
	group1.Version = "version3"

	oldSequence := group1.Sequence

	err = sqlStore.UpdateGroup(group1)
	require.NoError(t, err)
	assert.Equal(t, oldSequence+1, group1.Sequence)

	oldSequence = group1.Sequence
	group1.Sequence = 9001
	group1.Version = "version4"

	err = sqlStore.UpdateGroup(group1)
	require.NoError(t, err)
	assert.Equal(t, oldSequence+1, group1.Sequence)

	actualGroup1, err := sqlStore.GetGroup(group1.ID)
	require.NoError(t, err)
	assert.Equal(t, group1, actualGroup1)

	actualGroup2, err := sqlStore.GetGroup(group2.ID)
	require.NoError(t, err)
	assert.Equal(t, group2, actualGroup2)
}

func TestDeleteGroup(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	group1 := &model.Group{
		Name:        "name1",
		Description: "description1",
		Version:     "version1",
	}

	err := sqlStore.CreateGroup(group1)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	group2 := &model.Group{
		Name:        "name2",
		Description: "description2",
		Version:     "version2",
	}

	err = sqlStore.CreateGroup(group2)
	require.NoError(t, err)

	err = sqlStore.DeleteGroup(group1.ID)
	require.NoError(t, err)

	actualGroup1, err := sqlStore.GetGroup(group1.ID)
	require.NoError(t, err)
	require.NotEqual(t, 0, actualGroup1.DeleteAt)
	group1.DeleteAt = actualGroup1.DeleteAt
	assert.Equal(t, group1, actualGroup1)

	actualGroup2, err := sqlStore.GetGroup(group2.ID)
	require.NoError(t, err)
	assert.Equal(t, group2, actualGroup2)

	time.Sleep(1 * time.Millisecond)

	// Deleting again shouldn't change timestamp
	err = sqlStore.DeleteGroup(group1.ID)
	require.NoError(t, err)

	actualGroup1, err = sqlStore.GetGroup(group1.ID)
	require.NoError(t, err)
	assert.Equal(t, group1, actualGroup1)
}
