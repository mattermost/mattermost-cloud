// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model_test

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
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
			tc.request.SetDefaults()

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
				Name: sToP("group1"),
			},
		},
		{
			"invalid name only",
			true,
			&model.PatchGroupRequest{
				Name: sToP(""),
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
			"name only",
			true,
			&model.PatchGroupRequest{
				Name: sToP("group1"),
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
				Description: sToP("group1 description"),
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
				Version: sToP("version1"),
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
				Image: sToP("image1"),
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
				Version: sToP("patch-version"),
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
