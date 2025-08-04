// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model_test

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/util"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestCreateGroupRequestValid(t *testing.T) {
	var testCases = []struct {
		testName     string
		requireError bool
		request      *model.CreateGroupRequest
	}{
		{
			"defaults",
			false,
			&model.CreateGroupRequest{
				Name:       "group1",
				MaxRolling: 1,
			},
		},
		{
			"no name",
			true,
			&model.CreateGroupRequest{
				MaxRolling: 1,
			},
		},
		{
			"negative max rolling",
			true,
			&model.CreateGroupRequest{
				Name:       "group1",
				MaxRolling: -1,
			},
		},
		{
			"invalid mattermost env",
			true,
			&model.CreateGroupRequest{
				Name:       "group1",
				MaxRolling: 1,
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: ""},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			if tc.requireError {
				assert.Error(t, tc.request.Validate())
			} else {
				assert.NoError(t, tc.request.Validate())
			}
		})
	}
}

func TestPatchGroupRequestValid(t *testing.T) {
	var testCases = []struct {
		testName    string
		expectError bool
		request     *model.PatchGroupRequest
	}{
		{
			"empty",
			false,
			&model.PatchGroupRequest{},
		},
		{
			"version only",
			false,
			&model.PatchGroupRequest{
				Name: util.SToP("group1"),
			},
		},
		{
			"invalid name only",
			true,
			&model.PatchGroupRequest{
				Name: util.SToP(""),
			},
		},
		{
			"max rolling only",
			false,
			&model.PatchGroupRequest{
				MaxRolling: i64oP(1),
			},
		},
		{
			"invalid max rolling only",
			true,
			&model.PatchGroupRequest{
				MaxRolling: i64oP(-1),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			if tc.expectError {
				assert.Error(t, tc.request.Validate())
				return
			}

			assert.NoError(t, tc.request.Validate())
		})
	}
}

func TestPatchGroupRequestApply(t *testing.T) {
	var testCases = []struct {
		testName             string
		expectApply          bool
		request              *model.PatchGroupRequest
		installation         *model.Group
		expectedInstallation *model.Group
	}{
		{
			"empty",
			false,
			&model.PatchGroupRequest{},
			&model.Group{},
			&model.Group{},
		},
		{
			"force sequence bump",
			true,
			&model.PatchGroupRequest{
				ForceSequenceUpdate: true,
			},
			&model.Group{},
			&model.Group{},
		},
		{
			"name only",
			true,
			&model.PatchGroupRequest{
				Name: util.SToP("group1"),
			},
			&model.Group{},
			&model.Group{
				Name: "group1",
			},
		},
		{
			"description only",
			true,
			&model.PatchGroupRequest{
				Description: util.SToP("group1 description"),
			},
			&model.Group{},
			&model.Group{
				Description: "group1 description",
			},
		},
		{
			"version only",
			true,
			&model.PatchGroupRequest{
				Version: util.SToP("version1"),
			},
			&model.Group{},
			&model.Group{
				Version: "version1",
			},
		},
		{
			"image only",
			true,
			&model.PatchGroupRequest{
				Image: util.SToP("image1"),
			},
			&model.Group{},
			&model.Group{
				Image: "image1",
			},
		},
		{
			"max rolling only",
			true,
			&model.PatchGroupRequest{
				MaxRolling: i64oP(5),
			},
			&model.Group{},
			&model.Group{
				MaxRolling: 5,
			},
		},
		{
			"mattermost env only, no group env",
			true,
			&model.PatchGroupRequest{
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "value1"},
				},
			},
			&model.Group{},
			&model.Group{
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "value1"},
				},
			},
		},
		{
			"mattermost env only, patch group env with no changes",
			false,
			&model.PatchGroupRequest{
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "value1"},
				},
			},
			&model.Group{
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "value1"},
				},
			},
			&model.Group{
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "value1"},
				},
			},
		},
		{
			"mattermost env only, patch group env with changes",
			true,
			&model.PatchGroupRequest{
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "value1"},
				},
			},
			&model.Group{
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "value2"},
				},
			},
			&model.Group{
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "value1"},
				},
			},
		},
		{
			"mattermost env only, patch group env with new key",
			true,
			&model.PatchGroupRequest{
				MattermostEnv: model.EnvVarMap{
					"key2": {Value: "value1"},
				},
			},
			&model.Group{
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "value1"},
				},
			},
			&model.Group{
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "value1"},
					"key2": {Value: "value1"},
				},
			},
		},
		{
			"complex",
			true,
			&model.PatchGroupRequest{
				Version: util.SToP("patch-version"),
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "patch-value-1"},
					"key3": {Value: "patch-value-3"},
				},
			},
			&model.Group{
				Version: "version1",
				Image:   "image1",
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "value1"},
					"key2": {Value: "value2"},
				},
			},
			&model.Group{
				Version: "patch-version",
				Image:   "image1",
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "patch-value-1"},
					"key2": {Value: "value2"},
					"key3": {Value: "patch-value-3"},
				},
			},
		},
		{
			"force restart",
			true,
			&model.PatchGroupRequest{
				ForceInstallationsRestart: true,
			},
			&model.Group{
				Image:    "image1",
				Sequence: 1,
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "value1"},
				},
			},
			&model.Group{
				Image:    "image1",
				Sequence: 1,
				MattermostEnv: model.EnvVarMap{
					"key1":                               {Value: "value1"},
					"CLOUD_PROVISIONER_ENFORCED_RESTART": {Value: "force-restart-at-sequence-1"},
				},
			},
		},
		{
			"scheduling only - set for first time",
			true,
			&model.PatchGroupRequest{
				Scheduling: &model.Scheduling{
					NodeSelector: map[string]string{
						"node-type": "worker",
					},
					Tolerations: []corev1.Toleration{
						{
							Key:      "dedicated",
							Operator: corev1.TolerationOpEqual,
							Value:    "mattermost",
							Effect:   corev1.TaintEffectNoSchedule,
						},
					},
				},
			},
			&model.Group{},
			&model.Group{
				Scheduling: &model.Scheduling{
					NodeSelector: map[string]string{
						"node-type": "worker",
					},
					Tolerations: []corev1.Toleration{
						{
							Key:      "dedicated",
							Operator: corev1.TolerationOpEqual,
							Value:    "mattermost",
							Effect:   corev1.TaintEffectNoSchedule,
						},
					},
				},
			},
		},
		{
			"scheduling only - update existing",
			true,
			&model.PatchGroupRequest{
				Scheduling: &model.Scheduling{
					NodeSelector: map[string]string{
						"environment": "production",
					},
				},
			},
			&model.Group{
				Scheduling: &model.Scheduling{
					NodeSelector: map[string]string{
						"node-type": "worker",
					},
				},
			},
			&model.Group{
				Scheduling: &model.Scheduling{
					NodeSelector: map[string]string{
						"environment": "production",
					},
				},
			},
		},
		{
			"scheduling only - no changes",
			false,
			&model.PatchGroupRequest{
				Scheduling: &model.Scheduling{
					NodeSelector: map[string]string{
						"node-type": "worker",
					},
				},
			},
			&model.Group{
				Scheduling: &model.Scheduling{
					NodeSelector: map[string]string{
						"node-type": "worker",
					},
				},
			},
			&model.Group{
				Scheduling: &model.Scheduling{
					NodeSelector: map[string]string{
						"node-type": "worker",
					},
				},
			},
		},
		{
			"scheduling only - clear scheduling",
			true,
			&model.PatchGroupRequest{
				Scheduling: &model.Scheduling{
					NodeSelector: nil,
					Tolerations:  []corev1.Toleration{},
				},
			},
			&model.Group{
				Scheduling: &model.Scheduling{
					NodeSelector: map[string]string{
						"node-type": "worker",
					},
				},
			},
			&model.Group{
				Scheduling: nil,
			},
		},
		{
			"scheduling only - tolerations only",
			true,
			&model.PatchGroupRequest{
				Scheduling: &model.Scheduling{
					Tolerations: []corev1.Toleration{
						{
							Key:      "dedicated",
							Operator: corev1.TolerationOpEqual,
							Value:    "mattermost",
							Effect:   corev1.TaintEffectNoSchedule,
						},
						{
							Key:      "high-priority",
							Operator: corev1.TolerationOpExists,
							Effect:   corev1.TaintEffectNoExecute,
						},
					},
				},
			},
			&model.Group{},
			&model.Group{
				Scheduling: &model.Scheduling{
					Tolerations: []corev1.Toleration{
						{
							Key:      "dedicated",
							Operator: corev1.TolerationOpEqual,
							Value:    "mattermost",
							Effect:   corev1.TaintEffectNoSchedule,
						},
						{
							Key:      "high-priority",
							Operator: corev1.TolerationOpExists,
							Effect:   corev1.TaintEffectNoExecute,
						},
					},
				},
			},
		},
		{
			"scheduling with other changes",
			true,
			&model.PatchGroupRequest{
				Version: util.SToP("new-version"),
				Scheduling: &model.Scheduling{
					NodeSelector: map[string]string{
						"zone": "us-east-1a",
					},
				},
				MattermostEnv: model.EnvVarMap{
					"NEW_KEY": {Value: "new-value"},
				},
			},
			&model.Group{
				Version: "old-version",
				MattermostEnv: model.EnvVarMap{
					"OLD_KEY": {Value: "old-value"},
				},
			},
			&model.Group{
				Version: "new-version",
				Scheduling: &model.Scheduling{
					NodeSelector: map[string]string{
						"zone": "us-east-1a",
					},
				},
				MattermostEnv: model.EnvVarMap{
					"OLD_KEY": {Value: "old-value"},
					"NEW_KEY": {Value: "new-value"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			apply := tc.request.Apply(tc.installation)
			assert.Equal(t, tc.expectApply, apply)
			assert.Equal(t, tc.expectedInstallation, tc.installation)
		})
	}
}

func i64oP(i int64) *int64 {
	return &i
}
