// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model_test

import (
	"bytes"
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateInstallationRequestValid(t *testing.T) {
	model.SetDeployOperators(true, true)
	var testCases = []struct {
		testName     string
		requireError bool
		request      *model.CreateInstallationRequest
	}{
		{
			"defaults",
			false,
			&model.CreateInstallationRequest{
				OwnerID: "owner1",
				DNS:     "domain4321.com",
			},
		},
		{
			"no owner ID",
			true,
			&model.CreateInstallationRequest{
				DNS: "domain4321.com",
			},
		},
		{
			"no DNS",
			true,
			&model.CreateInstallationRequest{
				OwnerID: "owner1",
			},
		},
		{
			"DNS too long",
			true,
			&model.CreateInstallationRequest{
				OwnerID: "owner1",
				DNS:     "asupersupersupersupersupersupersupersupersupersupersupersuperlongname.com",
			},
		},
		{
			"DNS too short",
			true,
			&model.CreateInstallationRequest{
				OwnerID: "owner1",
				DNS:     "a.com",
			},
		},
		{
			"DNS starts with a hyphen",
			true,
			&model.CreateInstallationRequest{
				OwnerID: "owner1",
				DNS:     "-domain4321.com",
			},
		},
		{
			"DNS has invalid unicode characters",
			true,
			&model.CreateInstallationRequest{
				OwnerID: "owner1",
				DNS:     "letseat🍕.com",
			},
		},
		{
			"DNS has invalid special characters",
			true,
			&model.CreateInstallationRequest{
				OwnerID: "owner1",
				DNS:     "joram&gabe.com",
			},
		},
		{
			"invalid installation size",
			true,
			&model.CreateInstallationRequest{
				OwnerID: "owner1",
				DNS:     "domain4321.com",
				Size:    "jumbo",
			},
		},
		{
			"invalid affinity size",
			true,
			&model.CreateInstallationRequest{
				OwnerID:  "owner1",
				DNS:      "domain4321.com",
				Affinity: "solo",
			},
		},
		{
			"invalid database",
			true,
			&model.CreateInstallationRequest{
				OwnerID:  "owner1",
				DNS:      "domain4321.com",
				Database: "none",
			},
		},
		{
			"invalid filestore",
			true,
			&model.CreateInstallationRequest{
				OwnerID:   "owner1",
				DNS:       "domain4321.com",
				Filestore: "none",
			},
		},
		{
			"invalid mattermost env",
			true,
			&model.CreateInstallationRequest{
				OwnerID: "owner1",
				DNS:     "domain4321.com",
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: ""},
				},
			},
		},
		{
			"invalid single tenant db replicas",
			true,
			&model.CreateInstallationRequest{
				OwnerID:  "owner1",
				DNS:      "domain4321.com",
				Database: model.InstallationDatabaseSingleTenantRDSPostgres,
				SingleTenantDatabaseConfig: model.SingleTenantDatabaseRequest{
					ReplicasCount: 33,
				},
			},
		},
		{
			"ignore invalid replicas if db not single tenant",
			false,
			&model.CreateInstallationRequest{
				OwnerID:  "owner1",
				DNS:      "domain4321.com",
				Database: model.InstallationDatabaseMultiTenantRDSPostgres,
				SingleTenantDatabaseConfig: model.SingleTenantDatabaseRequest{
					ReplicasCount: 33,
				},
			},
		},
		{
			"dns has space",
			true,
			&model.CreateInstallationRequest{
				OwnerID: "owner1",
				DNS:     "domain4321.com ",
			},
		},
		{
			"Group/Database/filestore is blank should not fail validation",
			false,
			&model.CreateInstallationRequest{
				OwnerID:   "owner1",
				DNS:       "domain4321.com",
				GroupID:   "",
				Filestore: "",
				Database:  "",
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

	t.Run("require annotated installation", func(t *testing.T) {
		request := &model.CreateInstallationRequest{
			OwnerID: "owner1",
			DNS:     "domain4321.com",
		}
		request.SetDefaults()

		assert.NoError(t, request.Validate())

		model.SetRequireAnnotatedInstallations(true)
		assert.Error(t, request.Validate())

		request.Annotations = []string{"my-annotation"}
		assert.NoError(t, request.Validate())
		model.SetRequireAnnotatedInstallations(false)
	})
}

func TestCreateInstallationRequestFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		request, err := model.NewCreateInstallationRequestFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.Error(t, err)
		require.Nil(t, request)
	})

	t.Run("invalid request", func(t *testing.T) {
		installation, err := model.NewCreateInstallationRequestFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, installation)
	})

	t.Run("request", func(t *testing.T) {
		request, err := model.NewCreateInstallationRequestFromReader(bytes.NewReader([]byte(`{
			"OwnerID":"owner",
			"Version":"version",
			"DNS":"dns4321.com",
			"License": "this_is_my_license",
			"MattermostEnv": {"key1": {"Value": "value1"}},
			"Affinity":"multitenant"
		}`)))
		require.NoError(t, err)

		expected := &model.CreateInstallationRequest{
			OwnerID:       "owner",
			Version:       "version",
			DNS:           "dns4321.com",
			License:       "this_is_my_license",
			MattermostEnv: model.EnvVarMap{"key1": {Value: "value1"}},
			Affinity:      "multitenant",
		}
		expected.SetDefaults()
		require.Equal(t, expected, request)
		require.NoError(t, request.Validate())
	})
}

func TestPatchInstallationRequestValid(t *testing.T) {
	var testCases = []struct {
		testName    string
		expectError bool
		request     *model.PatchInstallationRequest
	}{
		{
			"empty",
			false,
			&model.PatchInstallationRequest{},
		},
		{
			"version only",
			false,
			&model.PatchInstallationRequest{
				Version: sToP("version1"),
			},
		},
		{
			"invalid version only",
			true,
			&model.PatchInstallationRequest{
				Version: sToP(""),
			},
		},
		{
			"image only",
			false,
			&model.PatchInstallationRequest{
				Image: sToP("image1"),
			},
		},
		{
			"invalid image only",
			true,
			&model.PatchInstallationRequest{
				Image: sToP(""),
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

func TestPatchInstallationRequestApply(t *testing.T) {
	var testCases = []struct {
		testName             string
		expectApply          bool
		request              *model.PatchInstallationRequest
		installation         *model.Installation
		expectedInstallation *model.Installation
	}{
		{
			"empty",
			false,
			&model.PatchInstallationRequest{},
			&model.Installation{},
			&model.Installation{},
		},
		{
			"ownerID only",
			true,
			&model.PatchInstallationRequest{
				OwnerID: sToP("new-owner"),
			},
			&model.Installation{},
			&model.Installation{
				OwnerID: "new-owner",
			},
		},
		{
			"version only",
			true,
			&model.PatchInstallationRequest{
				Version: sToP("version1"),
			},
			&model.Installation{},
			&model.Installation{
				Version: "version1",
			},
		},
		{
			"image only",
			true,
			&model.PatchInstallationRequest{
				Image: sToP("image1"),
			},
			&model.Installation{},
			&model.Installation{
				Image: "image1",
			},
		},
		{
			"size only",
			true,
			&model.PatchInstallationRequest{
				Size: sToP("miniSingleton"),
			},
			&model.Installation{},
			&model.Installation{
				Size: "miniSingleton",
			},
		},
		{
			"license only",
			true,
			&model.PatchInstallationRequest{
				License: sToP("license1"),
			},
			&model.Installation{},
			&model.Installation{
				License: "license1",
			},
		},
		{
			"mattermost env only, no installation env",
			true,
			&model.PatchInstallationRequest{
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "value1"},
				},
			},
			&model.Installation{},
			&model.Installation{
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "value1"},
				},
			},
		},
		{
			"mattermost env only, patch installation env with no changes",
			false,
			&model.PatchInstallationRequest{
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "value1"},
				},
			},
			&model.Installation{
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "value1"},
				},
			},
			&model.Installation{
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "value1"},
				},
			},
		},
		{
			"mattermost env only, patch installation env with changes",
			true,
			&model.PatchInstallationRequest{
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "value1"},
				},
			},
			&model.Installation{
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "value2"},
				},
			},
			&model.Installation{
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "value1"},
				},
			},
		},
		{
			"mattermost env only, patch installation env with new key",
			true,
			&model.PatchInstallationRequest{
				MattermostEnv: model.EnvVarMap{
					"key2": {Value: "value1"},
				},
			},
			&model.Installation{
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "value1"},
				},
			},
			&model.Installation{
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "value1"},
					"key2": {Value: "value1"},
				},
			},
		},
		{
			"complex",
			true,
			&model.PatchInstallationRequest{
				OwnerID: sToP("new-owner"),
				Version: sToP("patch-version"),
				Size:    sToP("miniSingleton"),
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "patch-value-1"},
					"key3": {Value: "patch-value-3"},
				},
			},
			&model.Installation{
				OwnerID: "owner",
				Version: "version1",
				Image:   "image1",
				License: "license1",
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: "value1"},
					"key2": {Value: "value2"},
				},
			},
			&model.Installation{
				OwnerID: "new-owner",
				Version: "patch-version",
				Image:   "image1",
				License: "license1",
				Size:    "miniSingleton",
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

func TestNewPatchInstallationRequestFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		request, err := model.NewPatchInstallationRequestFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, &model.PatchInstallationRequest{}, request)
	})

	t.Run("invalid request", func(t *testing.T) {
		request, err := model.NewPatchInstallationRequestFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, request)
	})

	t.Run("request", func(t *testing.T) {
		request, err := model.NewPatchInstallationRequestFromReader(bytes.NewReader([]byte(`{
			"Version":"version",
			"License": "this_is_my_license",
			"MattermostEnv": {"key1": {"Value": "value1"}}
		}`)))
		require.NoError(t, err)

		expected := &model.PatchInstallationRequest{
			Version:       sToP("version"),
			License:       sToP("this_is_my_license"),
			MattermostEnv: model.EnvVarMap{"key1": {Value: "value1"}},
		}
		require.Equal(t, expected, request)
		require.NoError(t, request.Validate())
	})
}

func sToP(s string) *string {
	return &s
}
