// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model_test

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/require"
)

func TestNewKopsMetadata(t *testing.T) {
	t.Run("nil payload", func(t *testing.T) {
		kopsMetadata, err := model.NewKopsMetadata(nil)
		require.NoError(t, err)
		require.Nil(t, kopsMetadata)
	})

	t.Run("invalid payload", func(t *testing.T) {
		_, err := model.NewKopsMetadata([]byte(`{`))
		require.Error(t, err)
	})

	t.Run("valid payload", func(t *testing.T) {
		kopsMetadata, err := model.NewKopsMetadata([]byte(`{"Name": "name"}`))
		require.NoError(t, err)
		require.Equal(t, "name", kopsMetadata.Name)
	})
}

func TestValidateChangeRequest(t *testing.T) {
	var km model.KopsMetadata

	t.Run("nil ChangeRequest", func(t *testing.T) {
		require.Error(t, km.ValidateChangeRequest())
	})

	t.Run("empty ChangeRequest", func(t *testing.T) {
		km.ChangeRequest = &model.KopsMetadataRequestedState{}
		require.Error(t, km.ValidateChangeRequest())
	})

	t.Run("valid ChangeRequest", func(t *testing.T) {
		km.ChangeRequest.Version = "1.0.0"
		require.NoError(t, km.ValidateChangeRequest())
	})
}

func TestGetWorkerNodesResizeChanges(t *testing.T) {
	var testCases = []struct {
		testName string
		metadata model.KopsMetadata
		expected model.KopsInstanceGroupsMetadata
	}{
		{
			"no change",
			model.KopsMetadata{
				NodeMinCount: 1,
				NodeInstanceGroups: model.KopsInstanceGroupsMetadata{
					"node-ig1": model.KopsInstanceGroupMetadata{
						NodeMinCount: 1,
						NodeMaxCount: 1,
					},
				},
				ChangeRequest: &model.KopsMetadataRequestedState{
					NodeMinCount: 1,
				},
			},
			model.KopsInstanceGroupsMetadata{
				"node-ig1": model.KopsInstanceGroupMetadata{
					NodeMinCount: 1,
					NodeMaxCount: 1,
				},
			},
		},
		{
			"add one, one node group",
			model.KopsMetadata{
				NodeMinCount: 1,
				NodeInstanceGroups: model.KopsInstanceGroupsMetadata{
					"node-ig1": model.KopsInstanceGroupMetadata{
						NodeMinCount: 1,
						NodeMaxCount: 1,
					},
				},
				ChangeRequest: &model.KopsMetadataRequestedState{
					NodeMinCount: 2,
				},
			},
			model.KopsInstanceGroupsMetadata{
				"node-ig1": model.KopsInstanceGroupMetadata{
					NodeMinCount: 2,
					NodeMaxCount: 2,
				},
			},
		},
		{
			"add ten, one node group",
			model.KopsMetadata{
				NodeMinCount: 1,
				NodeInstanceGroups: model.KopsInstanceGroupsMetadata{
					"node-ig1": model.KopsInstanceGroupMetadata{
						NodeMinCount: 1,
						NodeMaxCount: 1,
					},
				},
				ChangeRequest: &model.KopsMetadataRequestedState{
					NodeMinCount: 11,
				},
			},
			model.KopsInstanceGroupsMetadata{
				"node-ig1": model.KopsInstanceGroupMetadata{
					NodeMinCount: 11,
					NodeMaxCount: 11,
				},
			},
		},
		{
			"add one, two node groups",
			model.KopsMetadata{
				NodeMinCount: 1,
				NodeInstanceGroups: model.KopsInstanceGroupsMetadata{
					"node-ig1": model.KopsInstanceGroupMetadata{
						NodeMinCount: 1,
						NodeMaxCount: 1,
					},
					"node-ig2": model.KopsInstanceGroupMetadata{
						NodeMinCount: 0,
						NodeMaxCount: 0,
					},
				},
				ChangeRequest: &model.KopsMetadataRequestedState{
					NodeMinCount: 2,
				},
			},
			model.KopsInstanceGroupsMetadata{
				"node-ig1": model.KopsInstanceGroupMetadata{
					NodeMinCount: 1,
					NodeMaxCount: 1,
				},
				"node-ig2": model.KopsInstanceGroupMetadata{
					NodeMinCount: 1,
					NodeMaxCount: 1,
				},
			},
		},
		{
			"add ten, two node groups",
			model.KopsMetadata{
				NodeMinCount: 1,
				NodeInstanceGroups: model.KopsInstanceGroupsMetadata{
					"node-ig1": model.KopsInstanceGroupMetadata{
						NodeMinCount: 1,
						NodeMaxCount: 1,
					},
					"node-ig2": model.KopsInstanceGroupMetadata{
						NodeMinCount: 0,
						NodeMaxCount: 0,
					},
				},
				ChangeRequest: &model.KopsMetadataRequestedState{
					NodeMinCount: 11,
				},
			},
			model.KopsInstanceGroupsMetadata{
				"node-ig1": model.KopsInstanceGroupMetadata{
					NodeMinCount: 6,
					NodeMaxCount: 6,
				},
				"node-ig2": model.KopsInstanceGroupMetadata{
					NodeMinCount: 5,
					NodeMaxCount: 5,
				},
			},
		},
		{
			"add ten, four node groups",
			model.KopsMetadata{
				NodeMinCount: 3,
				NodeInstanceGroups: model.KopsInstanceGroupsMetadata{
					"node-ig1": model.KopsInstanceGroupMetadata{
						NodeMinCount: 1,
						NodeMaxCount: 1,
					},
					"node-ig2": model.KopsInstanceGroupMetadata{
						NodeMinCount: 1,
						NodeMaxCount: 1,
					},
					"node-ig3": model.KopsInstanceGroupMetadata{
						NodeMinCount: 1,
						NodeMaxCount: 1,
					},
					"node-ig4": model.KopsInstanceGroupMetadata{
						NodeMinCount: 0,
						NodeMaxCount: 0,
					},
				},
				ChangeRequest: &model.KopsMetadataRequestedState{
					NodeMinCount: 11,
				},
			},
			model.KopsInstanceGroupsMetadata{
				"node-ig1": model.KopsInstanceGroupMetadata{
					NodeMinCount: 3,
					NodeMaxCount: 3,
				},
				"node-ig2": model.KopsInstanceGroupMetadata{
					NodeMinCount: 3,
					NodeMaxCount: 3,
				},
				"node-ig3": model.KopsInstanceGroupMetadata{
					NodeMinCount: 3,
					NodeMaxCount: 3,
				},
				"node-ig4": model.KopsInstanceGroupMetadata{
					NodeMinCount: 2,
					NodeMaxCount: 2,
				},
			},
		},
		{
			"remove one, one node group",
			model.KopsMetadata{
				NodeMinCount: 2,
				NodeInstanceGroups: model.KopsInstanceGroupsMetadata{
					"node-ig1": model.KopsInstanceGroupMetadata{
						NodeMinCount: 2,
						NodeMaxCount: 2,
					},
				},
				ChangeRequest: &model.KopsMetadataRequestedState{
					NodeMinCount: 1,
				},
			},
			model.KopsInstanceGroupsMetadata{
				"node-ig1": model.KopsInstanceGroupMetadata{
					NodeMinCount: 1,
					NodeMaxCount: 1,
				},
			},
		},
		{
			"remove ten, one node group",
			model.KopsMetadata{
				NodeMinCount: 11,
				NodeInstanceGroups: model.KopsInstanceGroupsMetadata{
					"node-ig1": model.KopsInstanceGroupMetadata{
						NodeMinCount: 11,
						NodeMaxCount: 11,
					},
				},
				ChangeRequest: &model.KopsMetadataRequestedState{
					NodeMinCount: 1,
				},
			},
			model.KopsInstanceGroupsMetadata{
				"node-ig1": model.KopsInstanceGroupMetadata{
					NodeMinCount: 1,
					NodeMaxCount: 1,
				},
			},
		},
		{
			"remove one, two node groups",
			model.KopsMetadata{
				NodeMinCount: 2,
				NodeInstanceGroups: model.KopsInstanceGroupsMetadata{
					"node-ig1": model.KopsInstanceGroupMetadata{
						NodeMinCount: 1,
						NodeMaxCount: 1,
					},
					"node-ig2": model.KopsInstanceGroupMetadata{
						NodeMinCount: 1,
						NodeMaxCount: 1,
					},
				},
				ChangeRequest: &model.KopsMetadataRequestedState{
					NodeMinCount: 1,
				},
			},
			model.KopsInstanceGroupsMetadata{
				"node-ig1": model.KopsInstanceGroupMetadata{
					NodeMinCount: 1,
					NodeMaxCount: 1,
				},
				"node-ig2": model.KopsInstanceGroupMetadata{
					NodeMinCount: 0,
					NodeMaxCount: 0,
				},
			},
		},
		{
			"remove ten, two node groups",
			model.KopsMetadata{
				NodeMinCount: 21,
				NodeInstanceGroups: model.KopsInstanceGroupsMetadata{
					"node-ig1": model.KopsInstanceGroupMetadata{
						NodeMinCount: 11,
						NodeMaxCount: 11,
					},
					"node-ig2": model.KopsInstanceGroupMetadata{
						NodeMinCount: 10,
						NodeMaxCount: 10,
					},
				},
				ChangeRequest: &model.KopsMetadataRequestedState{
					NodeMinCount: 11,
				},
			},
			model.KopsInstanceGroupsMetadata{
				"node-ig1": model.KopsInstanceGroupMetadata{
					NodeMinCount: 6,
					NodeMaxCount: 6,
				},
				"node-ig2": model.KopsInstanceGroupMetadata{
					NodeMinCount: 5,
					NodeMaxCount: 5,
				},
			},
		},
		{
			"remove ten, four node groups",
			model.KopsMetadata{
				NodeMinCount: 21,
				NodeInstanceGroups: model.KopsInstanceGroupsMetadata{
					"node-ig1": model.KopsInstanceGroupMetadata{
						NodeMinCount: 6,
						NodeMaxCount: 6,
					},
					"node-ig2": model.KopsInstanceGroupMetadata{
						NodeMinCount: 5,
						NodeMaxCount: 5,
					},
					"node-ig3": model.KopsInstanceGroupMetadata{
						NodeMinCount: 5,
						NodeMaxCount: 5,
					},
					"node-ig4": model.KopsInstanceGroupMetadata{
						NodeMinCount: 5,
						NodeMaxCount: 5,
					},
				},
				ChangeRequest: &model.KopsMetadataRequestedState{
					NodeMinCount: 11,
				},
			},
			model.KopsInstanceGroupsMetadata{
				"node-ig1": model.KopsInstanceGroupMetadata{
					NodeMinCount: 3,
					NodeMaxCount: 3,
				},
				"node-ig2": model.KopsInstanceGroupMetadata{
					NodeMinCount: 3,
					NodeMaxCount: 3,
				},
				"node-ig3": model.KopsInstanceGroupMetadata{
					NodeMinCount: 3,
					NodeMaxCount: 3,
				},
				"node-ig4": model.KopsInstanceGroupMetadata{
					NodeMinCount: 2,
					NodeMaxCount: 2,
				},
			},
		},
		{
			"instance type only",
			model.KopsMetadata{
				NodeMinCount: 1,
				NodeInstanceGroups: model.KopsInstanceGroupsMetadata{
					"node-ig1": model.KopsInstanceGroupMetadata{
						NodeInstanceType: "t1",
						NodeMinCount:     1,
						NodeMaxCount:     1,
					},
				},
				ChangeRequest: &model.KopsMetadataRequestedState{
					NodeInstanceType: "t2",
				},
			},
			model.KopsInstanceGroupsMetadata{
				"node-ig1": model.KopsInstanceGroupMetadata{
					NodeInstanceType: "t2",
					NodeMinCount:     1,
					NodeMaxCount:     1,
				},
			},
		},
		{
			"remove five, three node groups, change instance type",
			model.KopsMetadata{
				NodeMinCount: 16,
				NodeInstanceGroups: model.KopsInstanceGroupsMetadata{
					"node-ig1": model.KopsInstanceGroupMetadata{
						NodeInstanceType: "t1",
						NodeMinCount:     6,
						NodeMaxCount:     6,
					},
					"node-ig2": model.KopsInstanceGroupMetadata{
						NodeInstanceType: "t1",
						NodeMinCount:     5,
						NodeMaxCount:     5,
					},
					"node-ig3": model.KopsInstanceGroupMetadata{
						NodeInstanceType: "t1",
						NodeMinCount:     5,
						NodeMaxCount:     5,
					},
				},
				ChangeRequest: &model.KopsMetadataRequestedState{
					NodeInstanceType: "t3",
					NodeMinCount:     11,
				},
			},
			model.KopsInstanceGroupsMetadata{
				"node-ig1": model.KopsInstanceGroupMetadata{
					NodeInstanceType: "t3",
					NodeMinCount:     4,
					NodeMaxCount:     4,
				},
				"node-ig2": model.KopsInstanceGroupMetadata{
					NodeInstanceType: "t3",
					NodeMinCount:     4,
					NodeMaxCount:     4,
				},
				"node-ig3": model.KopsInstanceGroupMetadata{
					NodeInstanceType: "t3",
					NodeMinCount:     3,
					NodeMaxCount:     3,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			require.Equal(t, tc.expected, tc.metadata.GetWorkerNodesResizeChanges())
		})
	}
}

func TestGetKopsResizeSetActionsFromChanges(t *testing.T) {
	var testCases = []struct {
		testName string
		metadata model.KopsMetadata
		changes  model.KopsInstanceGroupMetadata
		expected []string
	}{
		{
			"no change",
			model.KopsMetadata{
				NodeMinCount: 1,
				NodeInstanceGroups: model.KopsInstanceGroupsMetadata{
					"node-ig1": model.KopsInstanceGroupMetadata{
						NodeInstanceType: "t1",
						NodeMinCount:     1,
						NodeMaxCount:     1,
					},
				},
			},
			model.KopsInstanceGroupMetadata{
				NodeInstanceType: "t1",
				NodeMinCount:     1,
				NodeMaxCount:     1,
			},
			[]string{},
		},
		{
			"scale up",
			model.KopsMetadata{
				NodeMinCount: 1,
				NodeInstanceGroups: model.KopsInstanceGroupsMetadata{
					"node-ig1": model.KopsInstanceGroupMetadata{
						NodeInstanceType: "t1",
						NodeMinCount:     1,
						NodeMaxCount:     1,
					},
				},
			},
			model.KopsInstanceGroupMetadata{
				NodeInstanceType: "t1",
				NodeMinCount:     2,
				NodeMaxCount:     2,
			},
			[]string{"spec.maxSize=2", "spec.minSize=2"},
		},
		{
			"scale down",
			model.KopsMetadata{
				NodeMinCount: 1,
				NodeInstanceGroups: model.KopsInstanceGroupsMetadata{
					"node-ig1": model.KopsInstanceGroupMetadata{
						NodeInstanceType: "t1",
						NodeMinCount:     2,
						NodeMaxCount:     2,
					},
				},
			},
			model.KopsInstanceGroupMetadata{
				NodeInstanceType: "t1",
				NodeMinCount:     1,
				NodeMaxCount:     1,
			},
			[]string{"spec.minSize=1", "spec.maxSize=1"},
		},
		{
			"new instance type",
			model.KopsMetadata{
				NodeMinCount: 1,
				NodeInstanceGroups: model.KopsInstanceGroupsMetadata{
					"node-ig1": model.KopsInstanceGroupMetadata{
						NodeInstanceType: "t1",
						NodeMinCount:     1,
						NodeMaxCount:     1,
					},
				},
			},
			model.KopsInstanceGroupMetadata{
				NodeInstanceType: "t2",
				NodeMinCount:     1,
				NodeMaxCount:     1,
			},
			[]string{"spec.machineType=t2"},
		},
		{
			"scale up, new instance type",
			model.KopsMetadata{
				NodeMinCount: 1,
				NodeInstanceGroups: model.KopsInstanceGroupsMetadata{
					"node-ig1": model.KopsInstanceGroupMetadata{
						NodeInstanceType: "t1",
						NodeMinCount:     1,
						NodeMaxCount:     1,
					},
				},
			},
			model.KopsInstanceGroupMetadata{
				NodeInstanceType: "t2",
				NodeMinCount:     5,
				NodeMaxCount:     5,
			},
			[]string{"spec.maxSize=5", "spec.minSize=5", "spec.machineType=t2"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			require.Equal(t, tc.expected, tc.metadata.GetKopsResizeSetActionsFromChanges(tc.changes, "node-ig1"))
		})
	}
}

func TestRotatorConfigValidation(t *testing.T) {
	boolValue := true
	intValue := 10

	t.Run("UseRotator can't be nil", func(t *testing.T) {
		rotatorConfig := model.RotatorConfig{}
		require.Error(t, rotatorConfig.Validate())
	})
	t.Run("EvictGracePeriod must be set", func(t *testing.T) {
		rotatorConfig := model.RotatorConfig{
			UseRotator: &boolValue,
			// EvictGracePeriod: &intValue,
			MaxDrainRetries:         &intValue,
			MaxScaling:              &intValue,
			WaitBetweenRotations:    &intValue,
			WaitBetweenDrains:       &intValue,
			WaitBetweenPodEvictions: &intValue,
		}
		require.Error(t, rotatorConfig.Validate())
	})
	t.Run("MaxDrainRetries must be set", func(t *testing.T) {
		rotatorConfig := model.RotatorConfig{
			UseRotator:       &boolValue,
			EvictGracePeriod: &intValue,
			// MaxDrainRetries:         &intValue,
			MaxScaling:              &intValue,
			WaitBetweenRotations:    &intValue,
			WaitBetweenDrains:       &intValue,
			WaitBetweenPodEvictions: &intValue,
		}
		require.Error(t, rotatorConfig.Validate())
	})
	t.Run("MaxScaling must be set", func(t *testing.T) {
		rotatorConfig := model.RotatorConfig{
			UseRotator:       &boolValue,
			EvictGracePeriod: &intValue,
			MaxDrainRetries:  &intValue,
			// MaxScaling:              &intValue,
			WaitBetweenRotations:    &intValue,
			WaitBetweenDrains:       &intValue,
			WaitBetweenPodEvictions: &intValue,
		}
		require.Error(t, rotatorConfig.Validate())
	})
	t.Run("WaitBetweenRotations must be set", func(t *testing.T) {
		rotatorConfig := model.RotatorConfig{
			UseRotator:       &boolValue,
			EvictGracePeriod: &intValue,
			MaxDrainRetries:  &intValue,
			MaxScaling:       &intValue,
			// WaitBetweenRotations:    &intValue,
			WaitBetweenDrains:       &intValue,
			WaitBetweenPodEvictions: &intValue,
		}
		require.Error(t, rotatorConfig.Validate())
	})
	t.Run("WaitBetweenDrains must be set", func(t *testing.T) {
		rotatorConfig := model.RotatorConfig{
			UseRotator:           &boolValue,
			EvictGracePeriod:     &intValue,
			MaxDrainRetries:      &intValue,
			MaxScaling:           &intValue,
			WaitBetweenRotations: &intValue,
			// WaitBetweenDrains:       &intValue,
			WaitBetweenPodEvictions: &intValue,
		}
		require.Error(t, rotatorConfig.Validate())
	})
	t.Run("WaitBetweenPodEvictions must be set", func(t *testing.T) {
		rotatorConfig := model.RotatorConfig{
			UseRotator:           &boolValue,
			EvictGracePeriod:     &intValue,
			MaxDrainRetries:      &intValue,
			MaxScaling:           &intValue,
			WaitBetweenRotations: &intValue,
			WaitBetweenDrains:    &intValue,
			// WaitBetweenPodEvictions: &intValue,
		}
		require.Error(t, rotatorConfig.Validate())
	})
	t.Run("valid RotatorConfig", func(t *testing.T) {
		rotatorConfig := model.RotatorConfig{
			UseRotator:              &boolValue,
			EvictGracePeriod:        &intValue,
			MaxDrainRetries:         &intValue,
			MaxScaling:              &intValue,
			WaitBetweenRotations:    &intValue,
			WaitBetweenDrains:       &intValue,
			WaitBetweenPodEvictions: &intValue,
		}
		require.NoError(t, rotatorConfig.Validate())
	})
}
