// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mmv1alpha1 "github.com/mattermost/mattermost-operator/pkg/apis/mattermost/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func TestInstallations(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	ownerID1 := model.NewID()
	ownerID2 := model.NewID()
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

	err = sqlStore.CreateInstallation(installation1)
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

	t.Run("get unknown installation", func(t *testing.T) {
		installation, err := sqlStore.GetInstallation("unknown", false, false)
		require.NoError(t, err)
		require.Nil(t, installation)
	})

	t.Run("get installation 1", func(t *testing.T) {
		installation, err := sqlStore.GetInstallation(installation1.ID, false, false)
		require.NoError(t, err)
		require.Equal(t, installation1, installation)
	})

	t.Run("get installation 2", func(t *testing.T) {
		installation, err := sqlStore.GetInstallation(installation2.ID, false, false)
		require.NoError(t, err)
		require.Equal(t, installation2, installation)
	})

	t.Run("get installation 3", func(t *testing.T) {
		installation, err := sqlStore.GetInstallation(installation3.ID, false, false)
		require.NoError(t, err)
		require.Equal(t, installation3, installation)
	})

	t.Run("get installation 3 by name", func(t *testing.T) {
		installations, err := sqlStore.GetInstallations(
			&model.InstallationFilter{
				DNS:     installation3.DNS,
				PerPage: model.AllPerPage,
			}, false, false)

		require.Equal(t, 1, len(installations))
		installation := installations[0]

		require.NoError(t, err)
		require.Equal(t, installation.ID, installation3.ID)
	})

	t.Run("get and delete installation 4", func(t *testing.T) {
		installation, err := sqlStore.GetInstallation(installation4.ID, false, false)
		require.NoError(t, err)
		require.Equal(t, installation4, installation)

		err = sqlStore.DeleteInstallation(installation4.ID)
		require.NoError(t, err)
		installation4, err = sqlStore.GetInstallation(installation4.ID, false, false)
		require.NoError(t, err)
	})

	t.Run("groups", func(t *testing.T) {
		group, err := sqlStore.GetGroup(groupID1)
		require.NoError(t, err)
		require.Equal(t, group1, group)

		t.Run("include group config and overrides", func(t *testing.T) {
			installation, err := sqlStore.GetInstallation(installation1.ID, true, true)
			require.NoError(t, err)
			mergedInstallation := installation1.Clone()
			mergedInstallation.MergeWithGroup(group, true)
			require.Equal(t, mergedInstallation, installation)
		})

		t.Run("include group config, no overrides", func(t *testing.T) {
			installation, err := sqlStore.GetInstallation(installation1.ID, true, false)
			require.NoError(t, err)
			mergedInstallation := installation1.Clone()
			mergedInstallation.MergeWithGroup(group, false)
			require.Equal(t, mergedInstallation, installation)
		})
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
		{
			"group 1",
			&model.InstallationFilter{
				GroupID:        groupID1,
				Page:           0,
				PerPage:        10,
				IncludeDeleted: true,
			},
			[]*model.Installation{installation1, installation3},
		},
		{
			"owner 2, group 2, include deleted",
			&model.InstallationFilter{
				OwnerID:        ownerID2,
				GroupID:        groupID2,
				Page:           0,
				PerPage:        10,
				IncludeDeleted: true,
			},
			[]*model.Installation{installation4},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Description, func(t *testing.T) {
			actual, err := sqlStore.GetInstallations(testCase.Filter, false, false)
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

	updateRequestedInstallation := &model.Installation{
		OwnerID:  ownerID,
		Version:  "version",
		DNS:      "dns2.example.com",
		Affinity: model.InstallationAffinityIsolated,
		GroupID:  &groupID,
		State:    model.InstallationStateUpdateRequested,
	}
	err = sqlStore.CreateInstallation(updateRequestedInstallation)
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
		model.InstallationStateUpdateFailed,
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
	require.Equal(t, []*model.Installation{creationRequestedInstallation, updateRequestedInstallation, deletionRequestedInstallation}, installations)

	lockerID := model.NewID()

	locked, err := sqlStore.LockInstallation(creationRequestedInstallation.ID, lockerID)
	require.NoError(t, err)
	require.True(t, locked)

	installations, err = sqlStore.GetUnlockedInstallationsPendingWork()
	require.NoError(t, err)
	require.Equal(t, []*model.Installation{updateRequestedInstallation, deletionRequestedInstallation}, installations)

	locked, err = sqlStore.LockInstallation(updateRequestedInstallation.ID, lockerID)
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
		installation1, err = sqlStore.GetInstallation(installation1.ID, false, false)
		require.NoError(t, err)
		require.Equal(t, int64(0), installation1.LockAcquiredAt)
		require.Nil(t, installation1.LockAcquiredBy)

		installation2, err = sqlStore.GetInstallation(installation2.ID, false, false)
		require.NoError(t, err)
		require.Equal(t, int64(0), installation2.LockAcquiredAt)
		require.Nil(t, installation2.LockAcquiredBy)
	})

	t.Run("lock an unlocked installation", func(t *testing.T) {
		locked, err := sqlStore.LockInstallation(installation1.ID, lockerID1)
		require.NoError(t, err)
		require.True(t, locked)

		installation1, err = sqlStore.GetInstallation(installation1.ID, false, false)
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

		installation2, err = sqlStore.GetInstallation(installation2.ID, false, false)
		require.NoError(t, err)
		require.NotEqual(t, int64(0), installation2.LockAcquiredAt)
		require.Equal(t, lockerID2, *installation2.LockAcquiredBy)
	})

	t.Run("unlock the first installation", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockInstallation(installation1.ID, lockerID1, false)
		require.NoError(t, err)
		require.True(t, unlocked)

		installation1, err = sqlStore.GetInstallation(installation1.ID, false, false)
		require.NoError(t, err)
		require.Equal(t, int64(0), installation1.LockAcquiredAt)
		require.Nil(t, installation1.LockAcquiredBy)
	})

	t.Run("unlock the first installation again", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockInstallation(installation1.ID, lockerID1, false)
		require.NoError(t, err)
		require.False(t, unlocked)

		installation1, err = sqlStore.GetInstallation(installation1.ID, false, false)
		require.NoError(t, err)
		require.Equal(t, int64(0), installation1.LockAcquiredAt)
		require.Nil(t, installation1.LockAcquiredBy)
	})

	t.Run("force unlock the first installation again", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockInstallation(installation1.ID, lockerID1, true)
		require.NoError(t, err)
		require.False(t, unlocked)

		installation1, err = sqlStore.GetInstallation(installation1.ID, false, false)
		require.NoError(t, err)
		require.Equal(t, int64(0), installation1.LockAcquiredAt)
		require.Nil(t, installation1.LockAcquiredBy)
	})

	t.Run("unlock the second installation from the wrong locker", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockInstallation(installation2.ID, lockerID1, false)
		require.NoError(t, err)
		require.False(t, unlocked)

		installation2, err = sqlStore.GetInstallation(installation2.ID, false, false)
		require.NoError(t, err)
		require.NotEqual(t, int64(0), installation2.LockAcquiredAt)
		require.Equal(t, lockerID2, *installation2.LockAcquiredBy)
	})

	t.Run("force unlock the second installation from the wrong locker", func(t *testing.T) {
		unlocked, err := sqlStore.UnlockInstallation(installation2.ID, lockerID1, true)
		require.NoError(t, err)
		require.True(t, unlocked)

		installation2, err = sqlStore.GetInstallation(installation2.ID, false, false)
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

	someBool := false

	installation1 := &model.Installation{
		OwnerID:   ownerID1,
		Version:   "version",
		DNS:       "dns3.example.com",
		License:   "this-is-a-license",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		MattermostEnv: model.EnvVarMap{
			"Var1": model.EnvVar{
				Value: "Var1Value",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "1",
						FieldPath:  "some/path/neat",
					},
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
		},
		Size:     mmv1alpha1.Size100String,
		Affinity: model.InstallationAffinityIsolated,
		GroupID:  &groupID1,
		State:    model.InstallationStateCreationRequested,
	}

	err = sqlStore.CreateInstallation(installation1)
	require.NoError(t, err)

	installation2 := &model.Installation{
		OwnerID:   ownerID1,
		Version:   "version2",
		DNS:       "dns4.example.com",
		Image:     "custom/image",
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
	installation1.Version = "custom/image"
	installation1.DNS = "dns5.example.com"
	installation1.Size = mmv1alpha1.Size1000String
	installation1.Affinity = model.InstallationAffinityIsolated
	installation1.GroupID = &groupID2
	installation1.State = model.InstallationStateDeletionRequested

	err = sqlStore.UpdateInstallation(installation1)
	require.NoError(t, err)

	installation1.GroupID = &groupID1
	err = sqlStore.UpdateInstallation(installation1)
	require.NoError(t, err)

	actualInstallation1, err := sqlStore.GetInstallation(installation1.ID, false, false)
	require.NoError(t, err)
	require.Equal(t, installation1, actualInstallation1)

	actualInstallation2, err := sqlStore.GetInstallation(installation2.ID, false, false)
	require.NoError(t, err)
	require.Equal(t, installation2, actualInstallation2)

	t.Run("groups", func(t *testing.T) {
		group, err := sqlStore.GetGroup(groupID1)
		require.NoError(t, err)
		require.Equal(t, group1, group)

		t.Run("prevent saving merged installation", func(t *testing.T) {
			installation, err := sqlStore.GetInstallation(installation1.ID, true, true)
			require.NoError(t, err)
			mergedInstallation := installation1.Clone()
			mergedInstallation.MergeWithGroup(group, true)
			require.Equal(t, mergedInstallation, installation)

			err = sqlStore.UpdateInstallation(installation)
			require.Error(t, err)
		})
	})
}

func TestUpdateInstallationSequence(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	group1 := &model.Group{
		Version: "group1-version",
		MattermostEnv: model.EnvVarMap{
			"Key1": model.EnvVar{Value: "Value1"},
		},
	}
	err := sqlStore.CreateGroup(group1)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	installation1 := &model.Installation{
		OwnerID:   model.NewID(),
		Version:   "version",
		DNS:       "dns3.example.com",
		License:   "this-is-a-license",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Size:      mmv1alpha1.Size100String,
		Affinity:  model.InstallationAffinityIsolated,
		GroupID:   &group1.ID,
		State:     model.InstallationStateCreationRequested,
	}

	err = sqlStore.CreateInstallation(installation1)
	require.NoError(t, err)

	t.Run("group config not merged", func(t *testing.T) {
		installation, err := sqlStore.GetInstallation(installation1.ID, false, false)
		require.NoError(t, err)

		err = sqlStore.UpdateInstallationGroupSequence(installation)
		require.Error(t, err)
	})

	t.Run("group config merged", func(t *testing.T) {
		installation, err := sqlStore.GetInstallation(installation1.ID, true, false)
		require.NoError(t, err)

		oldSequence := installation.GroupSequence
		installation.SyncGroupAndInstallationSequence()
		err = sqlStore.UpdateInstallationGroupSequence(installation)
		require.NoError(t, err)

		installation, err = sqlStore.GetInstallation(installation1.ID, true, false)
		require.NoError(t, err)
		assert.NotEqual(t, oldSequence, installation.GroupSequence)
	})
}

func TestUpdateInstallationState(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	installation1 := &model.Installation{
		OwnerID:   model.NewID(),
		Version:   "version",
		DNS:       "dns3.example.com",
		License:   "this-is-a-license",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Size:      mmv1alpha1.Size100String,
		Affinity:  model.InstallationAffinityIsolated,
		State:     model.InstallationStateCreationRequested,
	}

	err := sqlStore.CreateInstallation(installation1)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	installation1.State = model.InstallationStateStable
	installation1.Version = "new-version-that-should-not-be-saved"

	err = sqlStore.UpdateInstallationState(installation1)
	require.NoError(t, err)

	storedInstallation, err := sqlStore.GetInstallation(installation1.ID, false, false)
	require.NoError(t, err)
	assert.Equal(t, storedInstallation.State, installation1.State)
	assert.NotEqual(t, storedInstallation.Version, installation1.Version)
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

	actualInstallation1, err := sqlStore.GetInstallation(installation1.ID, false, false)
	require.NoError(t, err)
	require.NotEqual(t, 0, actualInstallation1.DeleteAt)
	installation1.DeleteAt = actualInstallation1.DeleteAt
	require.Equal(t, installation1, actualInstallation1)

	actualInstallation2, err := sqlStore.GetInstallation(installation2.ID, false, false)
	require.NoError(t, err)
	require.Equal(t, installation2, actualInstallation2)

	time.Sleep(1 * time.Millisecond)

	// Deleting again shouldn't change timestamp
	err = sqlStore.DeleteInstallation(installation1.ID)
	require.NoError(t, err)

	actualInstallation1, err = sqlStore.GetInstallation(installation1.ID, false, false)
	require.NoError(t, err)
	require.Equal(t, installation1, actualInstallation1)
}
