package store

import (
	corev1 "k8s.io/api/core/v1"

	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	err = sqlStore.UpdateGroup(group1)
	require.NoError(t, err)

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
