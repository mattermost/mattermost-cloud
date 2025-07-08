// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model_test

import (
	"bytes"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/mattermost/mattermost-operator/apis/mattermost/v1alpha1"

	"github.com/mattermost/mattermost-cloud/internal/util"
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
			"invalid priority mattermost env",
			true,
			&model.CreateInstallationRequest{
				OwnerID: "owner1",
				DNS:     "domain4321.com",
				PriorityEnv: model.EnvVarMap{
					"key1": {Value: ""},
				},
			},
		},
		{
			"invalid size",
			true,
			&model.CreateInstallationRequest{
				OwnerID: "owner1",
				DNS:     "domain4321.com",
				Size:    "some-size",
			},
		},
		{
			"valid Operator size",
			false,
			&model.CreateInstallationRequest{
				OwnerID: "owner1",
				DNS:     "domain4321.com",
				Size:    v1alpha1.Size5000String,
			},
		},
		{
			"valid Provisioner size",
			false,
			&model.CreateInstallationRequest{
				OwnerID: "owner1",
				DNS:     "domain4321.com",
				Size:    fmt.Sprintf("%s-10", model.SizeProvisionerXL),
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
		{
			"new DNS format with name provided",
			false,
			&model.CreateInstallationRequest{
				OwnerID:  "owner1",
				DNSNames: []string{"my-installation.example.com"},
				Name:     "my-installation",
			},
		},
		{
			"validate all DNS",
			true,
			&model.CreateInstallationRequest{
				OwnerID:  "owner1",
				DNSNames: []string{"my-installation.example.com", "invalid dns.example.com"},
				Name:     "my-installation",
			},
		},
		{
			"name does not match DNS",
			true,
			&model.CreateInstallationRequest{
				OwnerID:  "owner1",
				DNSNames: []string{"my-installation.example.com", "some-installation.example.com"},
				Name:     "some-installation",
			},
		},
		{
			"new DNS format without",
			true,
			&model.CreateInstallationRequest{
				OwnerID:  "owner1",
				DNSNames: []string{"my-installation.example.com"},
			},
		},
		{
			"no DNS provided",
			true,
			&model.CreateInstallationRequest{
				OwnerID:  "owner1",
				DNSNames: []string{},
			},
		},
		{
			"validate DNS provided as DNSNames",
			true,
			&model.CreateInstallationRequest{
				OwnerID:  "owner1",
				DNSNames: []string{"my invalid-dns.com"},
			},
		},
		{
			"valid external database",
			false,
			&model.CreateInstallationRequest{
				OwnerID:  "owner1",
				Name:     "my-installation",
				DNSNames: []string{"my-installation.example.com"},
				Database: model.InstallationDatabaseExternal,
				ExternalDatabaseConfig: model.ExternalDatabaseRequest{
					SecretName: "test-secret",
				},
			},
		},
		{
			"invalid external database",
			true,
			&model.CreateInstallationRequest{
				OwnerID:                "owner1",
				Name:                   "my-installation",
				DNSNames:               []string{"my-installation.example.com"},
				Database:               model.InstallationDatabaseExternal,
				ExternalDatabaseConfig: model.ExternalDatabaseRequest{},
			},
		},
		{
			"valid local-ephemeral filestore",
			false,
			&model.CreateInstallationRequest{
				OwnerID:   "owner1",
				DNSNames:  []string{"my-installation.example.com"},
				Name:      "my-installation",
				Filestore: model.InstallationFilestoreLocalEphemeral,
			},
		},
		{
			"local-ephemeral filestore with minimal config",
			false,
			&model.CreateInstallationRequest{
				OwnerID:   "owner1",
				DNS:       "domain4321.com",
				Filestore: model.InstallationFilestoreLocalEphemeral,
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

	t.Run("convert DNS and name to lowercase", func(t *testing.T) {
		request := &model.CreateInstallationRequest{
			OwnerID:  "owner1",
			DNS:      "AWesoMeDomaiN4321.cOM",
			DNSNames: []string{"SOME-dnS123.COM"},
			Name:     "My-INSTALLATION",
		}

		request.SetDefaults()

		assert.Equal(t, "awesomedomain4321.com", request.DNS)
		assert.Equal(t, "some-dns123.com", request.DNSNames[1])
		assert.Equal(t, "my-installation", request.Name)
	})

	t.Run("set name based on dns and convert to DNSNames", func(t *testing.T) {
		request := &model.CreateInstallationRequest{
			OwnerID: "owner1",
			DNS:     "my-installation.mattermost.cloud.com",
		}

		request.SetDefaults()

		assert.Equal(t, "my-installation", request.Name)
		assert.Equal(t, []string{"my-installation.mattermost.cloud.com"}, request.DNSNames)
	})

	t.Run("set default filestore with local-ephemeral override", func(t *testing.T) {
		request := &model.CreateInstallationRequest{
			OwnerID:   "owner1",
			DNS:       "my-installation.mattermost.cloud.com",
			Filestore: model.InstallationFilestoreLocalEphemeral,
		}

		request.SetDefaults()

		assert.Equal(t, model.InstallationFilestoreLocalEphemeral, request.Filestore)
		assert.Equal(t, "my-installation", request.Name)
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
			"Name": "dns4321",
			"DNSNames": ["dns4321.cloud.com","dns4321.io"],
			"License": "this_is_my_license",
			"MattermostEnv": {"key1": {"Value": "value1"}},
			"Affinity":"multitenant"
		}`)))
		require.NoError(t, err)

		expected := &model.CreateInstallationRequest{
			OwnerID:       "owner",
			Version:       "version",
			DNS:           "dns4321.com",
			Name:          "dns4321",
			DNSNames:      []string{"dns4321.cloud.com", "dns4321.io"},
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
				Version: util.SToP("version1"),
			},
		},
		{
			"invalid version only",
			true,
			&model.PatchInstallationRequest{
				Version: util.SToP(""),
			},
		},
		{
			"image only",
			false,
			&model.PatchInstallationRequest{
				Image: util.SToP("image1"),
			},
		},
		{
			"invalid image only",
			true,
			&model.PatchInstallationRequest{
				Image: util.SToP(""),
			},
		},
		{
			"invalid size",
			true,
			&model.PatchInstallationRequest{
				Size: util.SToP("some-size"),
			},
		},
		{
			"valid Operator size",
			false,
			&model.PatchInstallationRequest{
				Size: util.SToP(v1alpha1.Size5000String),
			},
		},
		{
			"valid Provisioner size",
			false,
			&model.PatchInstallationRequest{
				Size: util.SToP(fmt.Sprintf("%s-10", model.SizeProvisionerXL)),
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

func TestMergeNewIngressSourceRangesWithExisting(t *testing.T) {
	// Define some test data
	allowedRanges := model.AllowedIPRanges{
		{CIDRBlock: "192.168.0.0/24", Description: "Test IP range", Enabled: true},
		{CIDRBlock: "10.0.0.0/8", Description: "Another test IP range", Enabled: false},
	}
	patchAllowedRanges := model.AllowedIPRanges{
		{CIDRBlock: "172.16.0.0/12", Description: "New IP range", Enabled: true},
		{CIDRBlock: "192.168.0.0/24", Description: "Updated IP range", Enabled: false},
	}
	patchInstallationRequest := &model.PatchInstallationRequest{
		AllowedIPRanges: &patchAllowedRanges,
	}
	installation := &model.Installation{
		AllowedIPRanges: &allowedRanges,
	}

	// Test merging with valid data
	mergedRanges, err := patchInstallationRequest.MergeNewIngressSourceRangesWithExisting(installation)
	assert.NoError(t, err)
	expectedRanges := model.AllowedIPRanges{
		{CIDRBlock: "192.168.0.0/24", Description: "Updated IP range", Enabled: false},
		{CIDRBlock: "10.0.0.0/8", Description: "Another test IP range", Enabled: false},
		{CIDRBlock: "172.16.0.0/12", Description: "New IP range", Enabled: true},
	}
	sortAllowedIPRanges(&expectedRanges)
	sortAllowedIPRanges(mergedRanges)
	assert.Equal(t, expectedRanges, *mergedRanges)

	t.Run("Test merging with nil patchAllowedRanges", func(t *testing.T) {
		patchInstallationRequest.AllowedIPRanges = nil
		mergedRanges, err = patchInstallationRequest.MergeNewIngressSourceRangesWithExisting(installation)
		assert.NoError(t, err)
		sortAllowedIPRanges(&allowedRanges)
		sortAllowedIPRanges(mergedRanges)
		assert.Equal(t, allowedRanges, *mergedRanges)
	})

	t.Run("Test merging with invalid CIDR block", func(t *testing.T) {
		patchAllowedRanges[0].CIDRBlock = "invalid"
		patchInstallationRequest.AllowedIPRanges = &patchAllowedRanges
		mergedRanges, err = patchInstallationRequest.MergeNewIngressSourceRangesWithExisting(installation)
		assert.Error(t, err)
		assert.Nil(t, mergedRanges)
	})

	t.Run("Test merging with nil installation.AllowedIPRanges", func(t *testing.T) {
		installation.AllowedIPRanges = nil
		mergedRanges, err = patchInstallationRequest.MergeNewIngressSourceRangesWithExisting(installation)
		assert.NoError(t, err)
		sortAllowedIPRanges(&patchAllowedRanges)
		sortAllowedIPRanges(mergedRanges)
		assert.Equal(t, patchAllowedRanges, *mergedRanges)
	})
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
				OwnerID: util.SToP("new-owner"),
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
				Version: util.SToP("version1"),
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
				Image: util.SToP("image1"),
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
				Size: util.SToP("miniSingleton"),
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
				License: util.SToP("license1"),
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
			"ranges only, with override should apply",
			true,
			&model.PatchInstallationRequest{
				AllowedIPRanges: &model.AllowedIPRanges{
					model.AllowedIPRange{CIDRBlock: "127.0.0.1"},
					model.AllowedIPRange{CIDRBlock: "192.168.0.1/24"},
				},
				OverrideIPRanges: util.BToP(true),
			},
			&model.Installation{
				AllowedIPRanges: &model.AllowedIPRanges{
					model.AllowedIPRange{CIDRBlock: "192.168.1.1/24"},
				},
			},
			&model.Installation{
				AllowedIPRanges: &model.AllowedIPRanges{
					model.AllowedIPRange{CIDRBlock: "127.0.0.1"},
					model.AllowedIPRange{CIDRBlock: "192.168.0.1/24"},
				},
			},
		},
		{
			"invalid ranges , without override should fail to apply",
			false,
			&model.PatchInstallationRequest{
				AllowedIPRanges: &model.AllowedIPRanges{
					model.AllowedIPRange{CIDRBlock: "127.0.0.1"},
					model.AllowedIPRange{CIDRBlock: "192.168.0.1/24"},
					model.AllowedIPRange{CIDRBlock: "blahblah"},
					model.AllowedIPRange{CIDRBlock: "1002.980.12.1"},
				},
			},
			&model.Installation{
				AllowedIPRanges: &model.AllowedIPRanges{
					model.AllowedIPRange{CIDRBlock: "192.168.1.1/24"},
				},
			},
			&model.Installation{
				AllowedIPRanges: &model.AllowedIPRanges{
					model.AllowedIPRange{CIDRBlock: "192.168.1.1/24"},
				},
			},
		},
		{
			"complex should apply",
			true,
			&model.PatchInstallationRequest{
				OwnerID: util.SToP("new-owner"),
				Version: util.SToP("patch-version"),
				Size:    util.SToP("miniSingleton"),
				AllowedIPRanges: &model.AllowedIPRanges{
					model.AllowedIPRange{CIDRBlock: "127.0.0.1"},
					model.AllowedIPRange{CIDRBlock: "192.168.0.1/24"},
				},
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
				AllowedIPRanges: &model.AllowedIPRanges{
					model.AllowedIPRange{CIDRBlock: "192.168.1.1/24"},
				},
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
				AllowedIPRanges: &model.AllowedIPRanges{
					model.AllowedIPRange{CIDRBlock: "192.168.0.1/24"},
					model.AllowedIPRange{CIDRBlock: "192.168.1.1/24"},
					model.AllowedIPRange{CIDRBlock: "127.0.0.1"},
				},
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
			sortAllowedIPRanges(tc.expectedInstallation.AllowedIPRanges)
			sortAllowedIPRanges(tc.installation.AllowedIPRanges)
			assert.Equal(t, tc.expectedInstallation, tc.installation)
		})
	}
}

// Sorts the list lexicographically by the CIDR block to allow for easier equality testing.
func sortAllowedIPRanges(allowedIPRanges *model.AllowedIPRanges) {
	if allowedIPRanges == nil {
		return
	}
	sort.Slice(*allowedIPRanges, func(i, j int) bool {
		return (*allowedIPRanges)[i].CIDRBlock < (*allowedIPRanges)[j].CIDRBlock
	})
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
			Version:       util.SToP("version"),
			License:       util.SToP("this_is_my_license"),
			MattermostEnv: model.EnvVarMap{"key1": {Value: "value1"}},
		}
		require.Equal(t, expected, request)
		require.NoError(t, request.Validate())
	})

	t.Run("request with ranges", func(t *testing.T) {
		request, err := model.NewPatchInstallationRequestFromReader(bytes.NewReader([]byte(`{
			"Version":"version",
			"License": "this_is_my_license",
			"AllowedIPRanges": [
				{
					"CIDRBlock": "127.0.0.1"
				},
				{
					"CIDRBlock": "192.168.1.0/24"
				}
			]
		}`)))
		require.NoError(t, err)

		expected := &model.PatchInstallationRequest{
			Version: util.SToP("version"),
			License: util.SToP("this_is_my_license"),
			AllowedIPRanges: &model.AllowedIPRanges{
				model.AllowedIPRange{CIDRBlock: "127.0.0.1"},
				model.AllowedIPRange{CIDRBlock: "192.168.1.0/24"},
			},
		}
		require.Equal(t, expected, request)
		require.NoError(t, request.Validate())
	})
}

func TestPatchInstallationDeletionRequestValid(t *testing.T) {
	var testCases = []struct {
		testName    string
		expectError bool
		request     *model.PatchInstallationDeletionRequest
	}{
		{
			"empty",
			false,
			&model.PatchInstallationDeletionRequest{},
		},
		{
			"deletion expiry only, valid with current time",
			false,
			&model.PatchInstallationDeletionRequest{
				DeletionPendingExpiry: util.IToP(model.GetMillis()),
			},
		},
		{
			"deletion expiry only, valid with future time",
			false,
			&model.PatchInstallationDeletionRequest{
				DeletionPendingExpiry: util.IToP(model.GetMillisAtTime(time.Now().Add(999999 * time.Hour))),
			},
		},
		{
			"deletion expiry only, invalid",
			true,
			&model.PatchInstallationDeletionRequest{
				DeletionPendingExpiry: util.IToP(model.GetMillisAtTime(time.Now().Add(-time.Hour))),
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

func TestPatchInstallationDeletionRequestApply(t *testing.T) {
	var testCases = []struct {
		testName             string
		expectApply          bool
		request              *model.PatchInstallationDeletionRequest
		installation         *model.Installation
		expectedInstallation *model.Installation
	}{
		{
			"empty",
			false,
			&model.PatchInstallationDeletionRequest{},
			&model.Installation{},
			&model.Installation{},
		},
		{
			"deletion expiry only",
			true,
			&model.PatchInstallationDeletionRequest{
				DeletionPendingExpiry: util.IToP(999),
			},
			&model.Installation{},
			&model.Installation{
				DeletionPendingExpiry: 999,
			},
		},
		{
			"deletion expiry only, same value",
			false,
			&model.PatchInstallationDeletionRequest{
				DeletionPendingExpiry: util.IToP(999),
			},
			&model.Installation{
				DeletionPendingExpiry: 999,
			},
			&model.Installation{
				DeletionPendingExpiry: 999,
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

func TestPatchInstallationScheduledDeletionRequestValidate(t *testing.T) {
	var testCases = []struct {
		testName    string
		expectError bool
		request     *model.PatchInstallationScheduledDeletionRequest
	}{
		{
			"nil scheduled deletion time",
			false,
			&model.PatchInstallationScheduledDeletionRequest{
				ScheduledDeletionTime: nil,
			},
		},
		{
			"zero scheduled deletion time",
			false,
			&model.PatchInstallationScheduledDeletionRequest{
				ScheduledDeletionTime: util.IToP(0),
			},
		},
		{
			"negative scheduled deletion time",
			true,
			&model.PatchInstallationScheduledDeletionRequest{
				ScheduledDeletionTime: util.IToP(-100),
			},
		},
		{
			"past scheduled deletion time",
			true,
			&model.PatchInstallationScheduledDeletionRequest{
				ScheduledDeletionTime: util.IToP(model.GetMillisAtTime(time.Now().Add(-1 * time.Hour))),
			},
		},
		{
			"future scheduled deletion time",
			false,
			&model.PatchInstallationScheduledDeletionRequest{
				ScheduledDeletionTime: util.IToP(model.GetMillisAtTime(time.Now().Add(24 * time.Hour))),
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

func TestPatchInstallationScheduledDeletionRequestApply(t *testing.T) {
	var testCases = []struct {
		testName             string
		expectApply          bool
		request              *model.PatchInstallationScheduledDeletionRequest
		installation         *model.Installation
		expectedInstallation *model.Installation
	}{
		{
			"nil scheduled deletion time",
			false,
			&model.PatchInstallationScheduledDeletionRequest{
				ScheduledDeletionTime: nil,
			},
			&model.Installation{
				ScheduledDeletionTime: 0,
			},
			&model.Installation{
				ScheduledDeletionTime: 0,
			},
		},
		{
			"zero scheduled deletion time",
			true,
			&model.PatchInstallationScheduledDeletionRequest{
				ScheduledDeletionTime: util.IToP(0),
			},
			&model.Installation{
				ScheduledDeletionTime: 100000,
			},
			&model.Installation{
				ScheduledDeletionTime: 0,
			},
		},
		{
			"non-zero scheduled deletion time",
			true,
			&model.PatchInstallationScheduledDeletionRequest{
				ScheduledDeletionTime: util.IToP(200000),
			},
			&model.Installation{
				ScheduledDeletionTime: 0,
			},
			&model.Installation{
				ScheduledDeletionTime: 200000,
			},
		},
		{
			"same scheduled deletion time",
			true,
			&model.PatchInstallationScheduledDeletionRequest{
				ScheduledDeletionTime: util.IToP(300000),
			},
			&model.Installation{
				ScheduledDeletionTime: 300000,
			},
			&model.Installation{
				ScheduledDeletionTime: 300000,
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

func TestNewAssignInstallationGroupFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		request, err := model.NewAssignInstallationGroupRequestFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, &model.AssignInstallationGroupRequest{}, request)
	})

	t.Run("invalid request", func(t *testing.T) {
		request, err := model.NewAssignInstallationGroupRequestFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, request)
	})

	t.Run("request", func(t *testing.T) {
		request, err := model.NewAssignInstallationGroupRequestFromReader(bytes.NewReader([]byte(`{
			"GroupSelectionAnnotations": ["test1", "test2"]
		}`)))
		require.NoError(t, err)

		expected := &model.AssignInstallationGroupRequest{
			GroupSelectionAnnotations: []string{"test1", "test2"},
		}
		require.Equal(t, expected, request)
	})
}
