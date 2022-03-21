// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"fmt"
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mmv1alpha1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func TestInstallations(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

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
		PriorityEnv: model.EnvVarMap{
			"V1": model.EnvVar{
				Value: "test",
			},
		},
	}

	err = sqlStore.CreateInstallation(installation1, annotations)
	require.NoError(t, err)

	t.Run("get installation", func(t *testing.T) {
		fetched, err := sqlStore.GetInstallation(installation1.ID, false, false)
		require.NoError(t, err)
		assert.Equal(t, installation1, fetched)
	})

	t.Run("fail on not unique DNS", func(t *testing.T) {
		if sqlStore.db.DriverName() != "postgres" {
			t.Skip()
		}
		err := sqlStore.CreateInstallation(&model.Installation{DNS: "dns.example.com"}, nil)
		require.Error(t, err)
	})

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

	err = sqlStore.CreateInstallation(installation3, nil)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	dbConfig := model.SingleTenantDatabaseConfig{
		PrimaryInstanceType: "db.r5.large",
		ReplicaInstanceType: "db.r5.medium",
		ReplicasCount:       4,
	}
	installation4 := &model.Installation{
		OwnerID:                    ownerID2,
		Version:                    "version",
		DNS:                        "dns4.example.com",
		Database:                   model.InstallationDatabaseMysqlOperator,
		Filestore:                  model.InstallationFilestoreMinioOperator,
		Size:                       mmv1alpha1.Size100String,
		Affinity:                   model.InstallationAffinityIsolated,
		GroupID:                    &groupID2,
		State:                      model.InstallationStateCreationRequested,
		SingleTenantDatabaseConfig: &dbConfig,
	}

	err = sqlStore.CreateInstallation(installation4, nil)
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
			&model.InstallationFilter{
				Paging: model.Paging{
					Page:           0,
					PerPage:        1,
					IncludeDeleted: false,
				},
			},
			[]*model.Installation{installation1},
		},
		{
			"page 0, perPage 10",
			&model.InstallationFilter{
				Paging: model.Paging{
					Page:           0,
					PerPage:        10,
					IncludeDeleted: false,
				},
			},
			[]*model.Installation{installation1, installation2, installation3},
		},
		{
			"page 0, perPage 10, include deleted",
			&model.InstallationFilter{
				Paging: model.Paging{
					Page:           0,
					PerPage:        10,
					IncludeDeleted: true,
				},
			},
			[]*model.Installation{installation1, installation2, installation3, installation4},
		},
		{
			"owner 1",
			&model.InstallationFilter{
				OwnerID: ownerID1,
				Paging:  model.AllPagesNotDeleted(),
			},
			[]*model.Installation{installation1, installation2},
		},
		{
			"owner 1, include deleted",
			&model.InstallationFilter{
				OwnerID: ownerID1,
				Paging:  model.AllPagesWithDeleted(),
			},
			[]*model.Installation{installation1, installation2},
		},
		{
			"owner 2",
			&model.InstallationFilter{
				OwnerID: ownerID2,
				Paging:  model.AllPagesNotDeleted(),
			},
			[]*model.Installation{installation3},
		},
		{
			"owner 2, include deleted",
			&model.InstallationFilter{
				OwnerID: ownerID2,
				Paging:  model.AllPagesWithDeleted(),
			},
			[]*model.Installation{installation3, installation4},
		},
		{
			"group 1",
			&model.InstallationFilter{
				GroupID: groupID1,
				Paging:  model.AllPagesWithDeleted(),
			},
			[]*model.Installation{installation1, installation3},
		},
		{
			"owner 2, group 2, include deleted",
			&model.InstallationFilter{
				OwnerID: ownerID2,
				GroupID: groupID2,
				Paging:  model.AllPagesWithDeleted(),
			},
			[]*model.Installation{installation4},
		},
		{
			"dns 3",
			&model.InstallationFilter{
				DNS:    installation3.DNS,
				Paging: model.AllPagesNotDeleted(),
			},
			[]*model.Installation{installation3},
		},
		{
			"state stable",
			&model.InstallationFilter{
				State:  model.InstallationStateStable,
				Paging: model.AllPagesNotDeleted(),
			},
			[]*model.Installation{installation2},
		},
		{
			"state creation-requested",
			&model.InstallationFilter{
				State:  model.InstallationStateCreationRequested,
				Paging: model.AllPagesNotDeleted(),
			},
			[]*model.Installation{installation1, installation3},
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
	defer CloseConnection(t, sqlStore)

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
	err := sqlStore.CreateInstallation(creationRequestedInstallation, nil)
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
	err = sqlStore.CreateInstallation(updateRequestedInstallation, nil)
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
	err = sqlStore.CreateInstallation(deletionRequestedInstallation, nil)
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

func TestGetSingleTenantDatabaseConfigForInstallation(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	dbConfig := &model.SingleTenantDatabaseConfig{
		PrimaryInstanceType: "db.r5.large",
		ReplicaInstanceType: "db.r5.xlarge",
		ReplicasCount:       11,
	}

	installation1 := model.Installation{
		DNS:                        "dns1.com",
		SingleTenantDatabaseConfig: dbConfig,
	}
	err := sqlStore.CreateInstallation(&installation1, nil)
	require.NoError(t, err)

	fetchedDBConfig, err := sqlStore.GetSingleTenantDatabaseConfigForInstallation(installation1.ID)
	require.NoError(t, err)
	assert.Equal(t, dbConfig, fetchedDBConfig)

	t.Run("no db config for installation", func(t *testing.T) {
		installation := model.Installation{}
		err := sqlStore.CreateInstallation(&installation, nil)
		require.NoError(t, err)

		_, err = sqlStore.GetSingleTenantDatabaseConfigForInstallation(installation.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})
}

func TestLockInstallation(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	lockerID1 := model.NewID()
	lockerID2 := model.NewID()

	ownerID := model.NewID()

	installation1 := &model.Installation{
		OwnerID: ownerID,
		DNS:     "dns1.example.com",
	}
	err := sqlStore.CreateInstallation(installation1, nil)
	require.NoError(t, err)

	installation2 := &model.Installation{
		OwnerID: ownerID,
		DNS:     "dns2.example.com",
	}
	err = sqlStore.CreateInstallation(installation2, nil)
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
	defer CloseConnection(t, sqlStore)

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
		Size:      mmv1alpha1.Size100String,
		Affinity:  model.InstallationAffinityIsolated,
		GroupID:   &groupID1,
		CRVersion: model.DefaultCRVersion,
		State:     model.InstallationStateCreationRequested,
	}

	err = sqlStore.CreateInstallation(installation1, nil)
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

	err = sqlStore.CreateInstallation(installation2, nil)
	require.NoError(t, err)

	installation1.OwnerID = ownerID2
	installation1.Version = "version3"
	installation1.Version = "custom/image"
	installation1.DNS = "dns5.example.com"
	installation1.Size = mmv1alpha1.Size1000String
	installation1.Affinity = model.InstallationAffinityIsolated
	installation1.GroupID = &groupID2
	installation1.CRVersion = model.V1betaCRVersion
	installation1.State = model.InstallationStateDeletionRequested
	installation1.PriorityEnv = model.EnvVarMap{
		"V1": model.EnvVar{
			Value: "test",
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
	}

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
	defer CloseConnection(t, sqlStore)

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

	err = sqlStore.CreateInstallation(installation1, nil)
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
	defer CloseConnection(t, sqlStore)

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

	err := sqlStore.CreateInstallation(installation1, nil)
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

func TestGetInstallationsStatus(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation1 := &model.Installation{
		OwnerID:   model.NewID(),
		Version:   "version",
		DNS:       "dns1.example.com",
		License:   "this-is-a-license",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Size:      mmv1alpha1.Size100String,
		Affinity:  model.InstallationAffinityIsolated,
		State:     model.InstallationStateCreationRequested,
	}

	err := sqlStore.CreateInstallation(installation1, nil)
	require.NoError(t, err)

	status, err := sqlStore.GetInstallationsStatus()
	require.NoError(t, err)
	assert.Equal(t, int64(1), status.InstallationsTotal)
	assert.Equal(t, int64(0), status.InstallationsStable)
	assert.Equal(t, int64(0), status.InstallationsHibernating)
	assert.Equal(t, int64(1), status.InstallationsUpdating)

	time.Sleep(1 * time.Millisecond)

	installation2 := &model.Installation{
		OwnerID:   model.NewID(),
		Version:   "version",
		DNS:       "dns2.example.com",
		License:   "this-is-a-license",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Size:      mmv1alpha1.Size100String,
		Affinity:  model.InstallationAffinityIsolated,
		State:     model.ClusterInstallationStateStable,
	}

	err = sqlStore.CreateInstallation(installation2, nil)
	require.NoError(t, err)

	status, err = sqlStore.GetInstallationsStatus()
	require.NoError(t, err)
	assert.Equal(t, int64(2), status.InstallationsTotal)
	assert.Equal(t, int64(1), status.InstallationsStable)
	assert.Equal(t, int64(0), status.InstallationsHibernating)
	assert.Equal(t, int64(1), status.InstallationsUpdating)

	time.Sleep(1 * time.Millisecond)

	installation3 := &model.Installation{
		OwnerID:   model.NewID(),
		Version:   "version",
		DNS:       "dns3.example.com",
		License:   "this-is-a-license",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Size:      mmv1alpha1.Size100String,
		Affinity:  model.InstallationAffinityIsolated,
		State:     model.InstallationStateHibernating,
	}

	err = sqlStore.CreateInstallation(installation3, nil)
	require.NoError(t, err)

	status, err = sqlStore.GetInstallationsStatus()
	require.NoError(t, err)
	assert.Equal(t, int64(3), status.InstallationsTotal)
	assert.Equal(t, int64(1), status.InstallationsStable)
	assert.Equal(t, int64(1), status.InstallationsHibernating)
	assert.Equal(t, int64(1), status.InstallationsUpdating)

	time.Sleep(1 * time.Millisecond)

	err = sqlStore.DeleteInstallation(installation1.ID)
	require.NoError(t, err)

	status, err = sqlStore.GetInstallationsStatus()
	require.NoError(t, err)
	assert.Equal(t, int64(2), status.InstallationsTotal)
	assert.Equal(t, int64(1), status.InstallationsStable)
	assert.Equal(t, int64(1), status.InstallationsHibernating)
	assert.Equal(t, int64(0), status.InstallationsUpdating)
}

func TestUpdateInstallationCRVersion(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

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
		CRVersion: model.V1betaCRVersion,
	}

	err := sqlStore.CreateInstallation(installation1, nil)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	err = sqlStore.UpdateInstallationCRVersion(installation1.ID, model.V1betaCRVersion)
	require.NoError(t, err)

	storedInstallation, err := sqlStore.GetInstallation(installation1.ID, false, false)
	require.NoError(t, err)
	assert.Equal(t, storedInstallation.CRVersion, model.V1betaCRVersion)
}

func TestGetInstallationsTotalDatabaseWeight(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation1 := &model.Installation{
		OwnerID:   model.NewID(),
		Version:   "version",
		DNS:       "dns1.example.com",
		License:   "this-is-a-license",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Size:      mmv1alpha1.Size100String,
		Affinity:  model.InstallationAffinityIsolated,
		State:     model.InstallationStateStable,
		CRVersion: model.V1betaCRVersion,
	}

	err := sqlStore.CreateInstallation(installation1, nil)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	installation2 := &model.Installation{
		OwnerID:   model.NewID(),
		Version:   "version",
		DNS:       "dns2.example.com",
		License:   "this-is-a-license",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Size:      mmv1alpha1.Size100String,
		Affinity:  model.InstallationAffinityIsolated,
		State:     model.InstallationStateStable,
		CRVersion: model.V1betaCRVersion,
	}

	err = sqlStore.CreateInstallation(installation2, nil)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	installation3 := &model.Installation{
		OwnerID:   model.NewID(),
		Version:   "version",
		DNS:       "dns3.example.com",
		License:   "this-is-a-license",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Size:      mmv1alpha1.Size100String,
		Affinity:  model.InstallationAffinityIsolated,
		State:     model.InstallationStateHibernating,
		CRVersion: model.V1betaCRVersion,
	}

	err = sqlStore.CreateInstallation(installation3, nil)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	t.Run("no installations in filter", func(t *testing.T) {
		totalWeight, err := sqlStore.GetInstallationsTotalDatabaseWeight([]string{})
		require.NoError(t, err)
		assert.Equal(t, float64(0), totalWeight)
	})

	t.Run("stable installation", func(t *testing.T) {
		totalWeight, err := sqlStore.GetInstallationsTotalDatabaseWeight([]string{installation1.ID})
		require.NoError(t, err)
		assert.Equal(t, installation1.GetDatabaseWeight(), totalWeight)
		assert.Equal(t, model.DefaultDatabaseWeight, totalWeight)
	})

	t.Run("hibernating installation", func(t *testing.T) {
		totalWeight, err := sqlStore.GetInstallationsTotalDatabaseWeight([]string{installation3.ID})
		require.NoError(t, err)
		assert.Equal(t, installation3.GetDatabaseWeight(), totalWeight)
		assert.Equal(t, model.HibernatingDatabaseWeight, totalWeight)
	})

	t.Run("three installations", func(t *testing.T) {
		totalWeight, err := sqlStore.GetInstallationsTotalDatabaseWeight([]string{
			installation1.ID,
			installation2.ID,
			installation3.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, installation1.GetDatabaseWeight()+installation2.GetDatabaseWeight()+installation3.GetDatabaseWeight(), totalWeight)
	})
}

func TestDeleteInstallation(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

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

	err := sqlStore.CreateInstallation(installation1, nil)
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

	err = sqlStore.CreateInstallation(installation2, nil)
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

// Helpers

func createAndCheckDummyInstallation(t *testing.T, store *SQLStore) *model.Installation {
	installation := &model.Installation{
		OwnerID: model.NewID(),
		DNS:     fmt.Sprintf("dns-%s.domain.com", model.NewID()),
	}
	createAndCheckInstallation(t, store, installation)

	return installation
}

func createAndCheckInstallation(t *testing.T, store *SQLStore, installation *model.Installation) {
	err := store.CreateInstallation(installation, nil)
	require.NoError(t, err)
	require.NotEmpty(t, installation.ID)
}
