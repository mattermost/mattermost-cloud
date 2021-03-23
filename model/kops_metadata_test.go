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
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			require.Equal(t, tc.expected, tc.metadata.GetWorkerNodesResizeChanges())
		})
	}
}
