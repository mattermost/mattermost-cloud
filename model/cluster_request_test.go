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
		{"invalid size", &model.CreateClusterRequest{Size: "blah"}, true},
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
			&model.KopsMetadata{},
			&model.KopsMetadata{},
		},
		{
			"version only",
			true,
			&model.PatchUpgradeClusterRequest{
				Version: sToP("version1"),
			},
			&model.KopsMetadata{},
			&model.KopsMetadata{
				Version: "version1",
			},
		},
		{
			"ami only",
			true,
			&model.PatchUpgradeClusterRequest{
				KopsAMI: sToP("image1"),
			},
			&model.KopsMetadata{},
			&model.KopsMetadata{
				AMI: "image1",
			},
		},
		{
			"version and ami",
			true,
			&model.PatchUpgradeClusterRequest{
				Version: sToP("version1"),
				KopsAMI: sToP("image1"),
			},
			&model.KopsMetadata{
				Version: "old-version",
			},
			&model.KopsMetadata{
				Version: "version1",
				AMI:     "image1",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			apply := tc.request.Apply(tc.metadata)
			assert.Equal(t, tc.expectApply, apply)
			assert.Equal(t, tc.expectedMetadata, tc.metadata)
		})
	}
}
