// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model_test

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
)

func TestCreateClusterRequestValid(t *testing.T) {
	var testCases = []struct {
		testName     string
		request      *model.CreateClusterRequest
		requireError bool
	}{
		{"defaults", &model.CreateClusterRequest{}, false},
		{"invalid provider", &model.CreateClusterRequest{Provider: "blah"}, true},
		{"invalid version", &model.CreateClusterRequest{Version: "blah"}, true},
		{"negative node counts", &model.CreateClusterRequest{NodeMinCount: -1, NodeMaxCount: -1}, true},
		{"negative master count", &model.CreateClusterRequest{MasterCount: -1}, true},
		{"mismatched node count", &model.CreateClusterRequest{NodeMinCount: 2, NodeMaxCount: 3}, true},
		{"max pods too low", &model.CreateClusterRequest{MaxPodsPerNode: 1}, true},
		{"eks no node group", &model.CreateClusterRequest{Provisioner: model.ProvisionerEKS}, true},
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

func TestUpgradeClusterRequestValid(t *testing.T) {
	var testCases = []struct {
		testName     string
		request      *model.PatchUpgradeClusterRequest
		requireError bool
	}{
		{"empty payload", &model.PatchUpgradeClusterRequest{}, false},
		{"valid version", &model.PatchUpgradeClusterRequest{Version: sToP("1.15.2")}, false},
		{"invalid version", &model.PatchUpgradeClusterRequest{Version: sToP("invalid")}, true},
		{"blank version", &model.PatchUpgradeClusterRequest{Version: sToP("")}, true},
		{"max pods too low", &model.PatchUpgradeClusterRequest{MaxPodsPerNode: i64oP(1)}, true},
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

func TestUpgradeClusterRequestApply(t *testing.T) {
	var testCases = []struct {
		testName         string
		expectApply      bool
		request          *model.PatchUpgradeClusterRequest
		metadata         *model.KopsMetadata
		expectedMetadata *model.KopsMetadata
	}{
		{
			"empty",
			false,
			&model.PatchUpgradeClusterRequest{},
			&model.KopsMetadata{
				ChangeRequest: &model.KopsMetadataRequestedState{},
			},
			&model.KopsMetadata{
				ChangeRequest:  &model.KopsMetadataRequestedState{},
				RotatorRequest: &model.RotatorMetadata{},
			},
		},
		{
			"version only",
			true,
			&model.PatchUpgradeClusterRequest{
				Version: sToP("version1"),
			},
			&model.KopsMetadata{
				ChangeRequest: &model.KopsMetadataRequestedState{},
			},
			&model.KopsMetadata{
				ChangeRequest: &model.KopsMetadataRequestedState{
					Version: "version1",
				},
				RotatorRequest: &model.RotatorMetadata{},
			},
		},
		{
			"ami only",
			true,
			&model.PatchUpgradeClusterRequest{
				AMI: sToP("image1"),
			},
			&model.KopsMetadata{
				ChangeRequest: &model.KopsMetadataRequestedState{},
			},
			&model.KopsMetadata{
				ChangeRequest: &model.KopsMetadataRequestedState{
					AMI: "image1",
				},
				RotatorRequest: &model.RotatorMetadata{},
			},
		},
		{
			"max pods only",
			true,
			&model.PatchUpgradeClusterRequest{
				MaxPodsPerNode: i64oP(200),
			},
			&model.KopsMetadata{
				ChangeRequest: &model.KopsMetadataRequestedState{},
			},
			&model.KopsMetadata{
				ChangeRequest: &model.KopsMetadataRequestedState{
					MaxPodsPerNode: 200,
				},
				RotatorRequest: &model.RotatorMetadata{},
			},
		},
		{
			"version and ami",
			true,
			&model.PatchUpgradeClusterRequest{
				Version: sToP("version1"),
				AMI:     sToP("image1"),
			},
			&model.KopsMetadata{
				Version:       "old-version",
				ChangeRequest: &model.KopsMetadataRequestedState{},
			},
			&model.KopsMetadata{
				Version: "old-version",
				ChangeRequest: &model.KopsMetadataRequestedState{
					Version: "version1",
					AMI:     "image1",
				},
				RotatorRequest: &model.RotatorMetadata{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			apply := tc.metadata.ApplyUpgradePatch(tc.request)
			assert.Equal(t, tc.expectApply, apply)
			assert.Equal(t, tc.expectedMetadata, tc.metadata)
		})
	}
}

func TestResizeClusterRequestValid(t *testing.T) {
	var testCases = []struct {
		testName     string
		request      *model.PatchClusterSizeRequest
		requireError bool
	}{
		{"empty payload", &model.PatchClusterSizeRequest{}, false},
		{"valid", &model.PatchClusterSizeRequest{NodeInstanceType: sToP("m5.large")}, false},
		{"blank node type", &model.PatchClusterSizeRequest{NodeInstanceType: sToP("")}, true},
		{"zero nodes", &model.PatchClusterSizeRequest{NodeMinCount: i64oP(0), NodeMaxCount: i64oP(0)}, true},
		{"max lower than min", &model.PatchClusterSizeRequest{NodeMinCount: i64oP(5), NodeMaxCount: i64oP(2)}, true},
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
