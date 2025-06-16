// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestInstallationClone(t *testing.T) {
	installation := &Installation{
		ID:       "id",
		OwnerID:  "owner",
		Version:  "version",
		Name:     "test",
		License:  "this_is_my_license",
		Affinity: InstallationAffinityIsolated,
		GroupID:  util.SToP("group_id"),
		State:    InstallationStateStable,
	}

	clone := installation.Clone()
	require.Equal(t, installation, clone)

	// Verify changing pointers in the clone doesn't affect the original.
	clone.GroupID = util.SToP("new_group_id")
	require.NotEqual(t, installation, clone)
}

func TestInstallationFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		installation, err := InstallationFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, &Installation{}, installation)
	})

	t.Run("invalid request", func(t *testing.T) {
		installation, err := InstallationFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, installation)
	})

	t.Run("request", func(t *testing.T) {
		installation, err := InstallationFromReader(bytes.NewReader([]byte(`{
			"ID":"id",
			"OwnerID":"owner",
			"GroupID":"group_id",
			"Version":"version",
			"Name":"dns",
			"License": "this_is_my_license",
			"MattermostEnv": {"key1": {"Value": "value1"}},
			"Affinity":"affinity",
			"State":"state",
			"CreateAt":10,
			"DeleteAt":20,
			"LockAcquiredAt":0
		}`)))
		require.NoError(t, err)
		require.Equal(t, &Installation{
			ID:             "id",
			OwnerID:        "owner",
			GroupID:        util.SToP("group_id"),
			Version:        "version",
			Name:           "dns",
			License:        "this_is_my_license",
			MattermostEnv:  EnvVarMap{"key1": {Value: "value1"}},
			Affinity:       "affinity",
			State:          "state",
			CreateAt:       10,
			DeleteAt:       20,
			LockAcquiredBy: nil,
			LockAcquiredAt: int64(0),
		}, installation)
	})
}

func TestInstallationsFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		installations, err := InstallationsFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, []*Installation{}, installations)
	})

	t.Run("invalid request", func(t *testing.T) {
		installations, err := InstallationsFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, installations)
	})

	t.Run("request", func(t *testing.T) {
		installation, err := InstallationsFromReader(bytes.NewReader([]byte(`[
			{
				"ID":"id1",
				"OwnerID":"owner1",
				"GroupID":"group_id1",
				"Version":"version1",
				"Name":"dns1",
				"MattermostEnv": {"key1": {"Value": "value1"}},
				"Affinity":"affinity1",
				"State":"state1",
				"CreateAt":10,
				"DeleteAt":20,
				"LockAcquiredAt":0
			},
			{
				"ID":"id2",
				"OwnerID":"owner2",
				"GroupID":"group_id2",
				"Version":"version2",
				"Name":"dns2",
				"License": "this_is_my_license",
				"MattermostEnv": {"key2": {"Value": "value2"}},
				"Affinity":"affinity2",
				"State":"state2",
				"CreateAt":30,
				"DeleteAt":40,
				"LockAcquiredBy": "tester",
				"LockAcquiredAt":50
			}
		]`)))
		require.NoError(t, err)
		require.Equal(t, []*Installation{
			{
				ID:             "id1",
				OwnerID:        "owner1",
				GroupID:        util.SToP("group_id1"),
				Version:        "version1",
				Name:           "dns1",
				MattermostEnv:  EnvVarMap{"key1": {Value: "value1"}},
				Affinity:       "affinity1",
				State:          "state1",
				CreateAt:       10,
				DeleteAt:       20,
				LockAcquiredBy: nil,
				LockAcquiredAt: 0,
			}, {
				ID:             "id2",
				OwnerID:        "owner2",
				GroupID:        util.SToP("group_id2"),
				Version:        "version2",
				Name:           "dns2",
				License:        "this_is_my_license",
				MattermostEnv:  EnvVarMap{"key2": {Value: "value2"}},
				Affinity:       "affinity2",
				State:          "state2",
				CreateAt:       30,
				DeleteAt:       40,
				LockAcquiredBy: util.SToP("tester"),
				LockAcquiredAt: 50,
			},
		}, installation)
	})
}

func TestMergeWithGroup(t *testing.T) {
	checkMergeValues := func(t *testing.T, installation *Installation, group *Group) {
		t.Helper()

		assert.Equal(t, installation.GroupID != nil, installation.IsInGroup())

		assert.Equal(t, installation.Version, group.Version)
		assert.Equal(t, installation.Image, group.Image)

		// TODO: check normal installation env settings that aren't overridden.
		for key := range group.MattermostEnv {
			assert.Equal(t, installation.MattermostEnv[key].Value, group.MattermostEnv[key].Value)
		}
	}

	t.Run("without overrides", func(t *testing.T) {
		installation := &Installation{
			ID:       NewID(),
			OwnerID:  "owner",
			Version:  "iversion",
			Image:    "iImage",
			Name:     "test",
			License:  "this_is_my_license",
			Affinity: InstallationAffinityIsolated,
			GroupID:  util.SToP("group_id"),
			State:    InstallationStateStable,
		}

		group := &Group{
			ID:      NewID(),
			Version: "gversion",
			Image:   "gImage",
			MattermostEnv: EnvVarMap{
				"key1": EnvVar{
					Value: "value1",
				},
			},
		}

		installation.MergeWithGroup(group, false)
		checkMergeValues(t, installation, group)
	})

	t.Run("with overrides, no env overrides found", func(t *testing.T) {
		installation := &Installation{
			ID:       NewID(),
			OwnerID:  "owner",
			Version:  "iversion",
			Image:    "iImage",
			Name:     "test",
			License:  "this_is_my_license",
			Affinity: InstallationAffinityIsolated,
			GroupID:  util.SToP("group_id"),
			State:    InstallationStateStable,
		}

		group := &Group{
			ID:      NewID(),
			Version: "gversion",
			Image:   "gImage",
			MattermostEnv: EnvVarMap{
				"key1": EnvVar{
					Value: "value1",
				},
			},
		}

		installation.MergeWithGroup(group, true)
		checkMergeValues(t, installation, group)
		assert.NotEmpty(t, installation.GroupOverrides)
	})

	t.Run("with overrides, env overrides found", func(t *testing.T) {
		installation := &Installation{
			ID:       NewID(),
			OwnerID:  "owner",
			Version:  "iversion",
			Image:    "iImage",
			Name:     "test",
			License:  "this_is_my_license",
			Affinity: InstallationAffinityIsolated,
			GroupID:  util.SToP("group_id"),
			State:    InstallationStateStable,
			MattermostEnv: EnvVarMap{
				"key2": EnvVar{
					Value: "ivalue1",
				},
			},
		}

		group := &Group{
			ID:      NewID(),
			Version: "gversion",
			Image:   "gImage",
			MattermostEnv: EnvVarMap{
				"key1": EnvVar{
					Value: "value1",
				},
				"key2": EnvVar{
					Value: "value2",
				},
			},
		}

		installation.MergeWithGroup(group, true)
		checkMergeValues(t, installation, group)
		assert.NotEmpty(t, installation.GroupOverrides)
	})

	t.Run("without overrides, group sequence matches", func(t *testing.T) {
		installation := &Installation{
			ID:            NewID(),
			OwnerID:       "owner",
			Version:       "iversion",
			Image:         "iImage",
			Name:          "test",
			License:       "this_is_my_license",
			Affinity:      InstallationAffinityIsolated,
			GroupID:       util.SToP("group_id"),
			GroupSequence: util.IToP(2),
			State:         InstallationStateStable,
		}

		group := &Group{
			ID:       NewID(),
			Sequence: 2,
			Version:  "gversion",
			Image:    "gImage",
			MattermostEnv: EnvVarMap{
				"key1": EnvVar{
					Value: "value1",
				},
			},
		}

		installation.MergeWithGroup(group, false)
		checkMergeValues(t, installation, group)
		assert.True(t, installation.InstallationSequenceMatchesMergedGroupSequence())
	})

	t.Run("without overrides, group sequence doesn't match", func(t *testing.T) {
		installation := &Installation{
			ID:            NewID(),
			OwnerID:       "owner",
			Version:       "iversion",
			Image:         "iImage",
			Name:          "test",
			License:       "this_is_my_license",
			Affinity:      InstallationAffinityIsolated,
			GroupID:       util.SToP("group_id"),
			GroupSequence: util.IToP(1),
			State:         InstallationStateStable,
		}

		group := &Group{
			ID:       NewID(),
			Sequence: 2,
			Version:  "gversion",
			Image:    "gImage",
			MattermostEnv: EnvVarMap{
				"key1": EnvVar{
					Value: "value1",
				},
			},
		}

		installation.MergeWithGroup(group, false)
		checkMergeValues(t, installation, group)
		assert.False(t, installation.InstallationSequenceMatchesMergedGroupSequence())
	})
}

func TestInstallation_GetEnvVars(t *testing.T) {
	for _, testCase := range []struct {
		description  string
		installation Installation
		expectedEnv  EnvVarMap
	}{
		{
			description:  "no envs",
			installation: Installation{},
			expectedEnv:  EnvVarMap{},
		},
		{
			description: "use regular envs",
			installation: Installation{MattermostEnv: EnvVarMap{
				"MM_TEST":  EnvVar{Value: "test"},
				"MM_TEST2": EnvVar{ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{Key: "test"}}}},
			},
			expectedEnv: EnvVarMap{
				"MM_TEST":  EnvVar{Value: "test"},
				"MM_TEST2": EnvVar{ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{Key: "test"}}},
			},
		},
		{
			description: "prioritize priority envs",
			installation: Installation{
				MattermostEnv: EnvVarMap{
					"MM_TEST":  EnvVar{Value: "test"},
					"MM_TEST2": EnvVar{ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{Key: "test"}}},
				},
				PriorityEnv: EnvVarMap{
					"MM_TEST2": EnvVar{ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{Key: "secret-test"}}},
					"MM_TEST3": EnvVar{Value: "test3"},
				},
			},
			expectedEnv: EnvVarMap{
				"MM_TEST":  EnvVar{Value: "test"},
				"MM_TEST2": EnvVar{ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{Key: "secret-test"}}},
				"MM_TEST3": EnvVar{Value: "test3"},
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			envs := testCase.installation.GetEnvVars()
			assert.Equal(t, testCase.expectedEnv, envs)
		})
	}
}

func TestInstallationGetDatabaseWeight(t *testing.T) {
	for _, testCase := range []struct {
		installation   Installation
		expectedWeight float64
	}{
		{
			Installation{State: InstallationStateStable},
			DefaultDatabaseWeight,
		},
		{
			Installation{State: InstallationStateUpdateInProgress},
			DefaultDatabaseWeight,
		},
		{
			Installation{State: InstallationStateHibernating},
			HibernatingDatabaseWeight,
		},
		{
			Installation{State: InstallationStateDeletionPendingRequested},
			HibernatingDatabaseWeight,
		},
		{
			Installation{State: InstallationStateDeletionPending},
			HibernatingDatabaseWeight,
		},
	} {
		t.Run(testCase.installation.State, func(t *testing.T) {
			assert.Equal(t, testCase.expectedWeight, testCase.installation.GetDatabaseWeight())
		})
	}
}

func TestInstallationRequiresAWSInfrasctructure(t *testing.T) {
	tests := []struct {
		installation          Installation
		expectedRequiresInfra bool
	}{
		{
			Installation{
				Database:  InstallationDatabaseMultiTenantRDSPostgresPGBouncer,
				Filestore: InstallationFilestoreBifrost,
			},
			true,
		}, {
			Installation{
				Database:  InstallationDatabaseMultiTenantRDSMySQL,
				Filestore: InstallationFilestoreBifrost,
			},
			true,
		}, {
			Installation{
				Database:  InstallationDatabaseExternal,
				Filestore: InstallationFilestoreBifrost,
			},
			true,
		}, {
			Installation{
				Database:  InstallationDatabaseExternal,
				Filestore: InstallationFilestoreAwsS3,
			},
			true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s_%s", test.installation.Database, test.installation.Filestore), func(t *testing.T) {
			assert.Equal(t, test.expectedRequiresInfra, test.installation.RequiresAWSInfrasctructure())
		})
	}
}

func TestAllowedIPRangesValue(t *testing.T) {
	// Define some test data
	ipRanges := AllowedIPRanges{
		{CIDRBlock: "192.168.0.0/24", Description: "Test IP range", Enabled: true},
		{CIDRBlock: "10.0.0.0/8", Description: "Another test IP range", Enabled: false},
	}

	// Call the Value() function
	value, err := ipRanges.Value()
	require.NoError(t, err)

	// Unmarshal the resulting value into an AllowedIPRanges slice
	var result AllowedIPRanges
	err = json.Unmarshal(value.([]byte), &result)
	require.NoError(t, err)

	// Check that the resulting slice is equal to the original slice
	require.Equal(t, len(ipRanges), len(result))
	for i, ipRange := range ipRanges {
		assert.Equal(t, ipRange.CIDRBlock, result[i].CIDRBlock)
		assert.Equal(t, ipRange.Description, result[i].Description)
		assert.Equal(t, ipRange.Enabled, result[i].Enabled)
	}
}

func TestAllowedIPRangesScan(t *testing.T) {
	// Define some test data
	ipRanges := AllowedIPRanges{
		{CIDRBlock: "192.168.0.0/24", Description: "Test IP range", Enabled: true},
		{CIDRBlock: "10.0.0.0/8", Description: "Another test IP range", Enabled: false},
	}
	jsonData, err := json.Marshal(ipRanges)
	require.NoError(t, err)

	// Call the Scan() function
	var result AllowedIPRanges
	err = result.Scan(jsonData)
	require.NoError(t, err)

	// Check that the resulting slice is equal to the original slice
	require.Equal(t, len(ipRanges), len(result))
	for i, ipRange := range ipRanges {
		assert.Equal(t, ipRange.CIDRBlock, result[i].CIDRBlock)
		assert.Equal(t, ipRange.Description, result[i].Description)
		assert.Equal(t, ipRange.Enabled, result[i].Enabled)
	}
}

func TestAllowedIPRangesFromJSONString(t *testing.T) {
	// Define some test data
	ipRanges := AllowedIPRanges{
		{CIDRBlock: "192.168.0.0/24", Description: "Test IP range", Enabled: true},
		{CIDRBlock: "10.0.0.0/8", Description: "Another test IP range", Enabled: false},
	}
	jsonData, err := json.Marshal(ipRanges)
	require.NoError(t, err)

	// Call the FromJSONString() function
	result, err := new(AllowedIPRanges).FromJSONString(string(jsonData))
	require.NoError(t, err)

	// Check that the resulting slice is equal to the original slice
	require.Equal(t, len(ipRanges), len(*result))
	for i, ipRange := range ipRanges {
		assert.Equal(t, ipRange.CIDRBlock, (*result)[i].CIDRBlock)
		assert.Equal(t, ipRange.Description, (*result)[i].Description)
		assert.Equal(t, ipRange.Enabled, (*result)[i].Enabled)
	}
}

func TestAllowedIPRangesToString(t *testing.T) {
	// Define some test data
	ipRanges := AllowedIPRanges{
		{CIDRBlock: "192.168.0.0/24", Description: "Test IP range", Enabled: true},
		{CIDRBlock: "10.0.0.0/8", Description: "Another test IP range", Enabled: false},
	}

	// Call the ToString() function
	result := ipRanges.ToString()

	// Unmarshal the resulting string into an AllowedIPRanges slice
	var parsedResult AllowedIPRanges
	err := json.Unmarshal([]byte(result), &parsedResult)
	require.NoError(t, err)

	// Check that the resulting slice is equal to the original slice
	require.Equal(t, len(ipRanges), len(parsedResult))
	for i, ipRange := range ipRanges {
		assert.Equal(t, ipRange.CIDRBlock, parsedResult[i].CIDRBlock)
		assert.Equal(t, ipRange.Description, parsedResult[i].Description)
		assert.Equal(t, ipRange.Enabled, parsedResult[i].Enabled)
	}
}

func TestAllowedIPRangesContains(t *testing.T) {
	// Define some test data
	ipRanges := AllowedIPRanges{
		{CIDRBlock: "192.168.0.0/24", Description: "Test IP range", Enabled: true},
		{CIDRBlock: "10.0.0.0/8", Description: "Another test IP range", Enabled: false},
	}

	// Test that the Contains() function returns true for an IP range that is present
	assert.True(t, ipRanges.Contains("192.168.0.0/24"))

	// Test that the Contains() function returns false for an IP range that is not present
	assert.False(t, ipRanges.Contains("172.16.0.0/12"))

	// Test that the Contains() function returns false for a nil AllowedIPRanges slice
	var nilIPRanges *AllowedIPRanges
	assert.False(t, nilIPRanges.Contains("192.168.0.0/24"))
}

func TestAllowedIPRangesToAnnotationString(t *testing.T) {
	// Define some test data
	ipRanges := AllowedIPRanges{
		{CIDRBlock: "192.168.0.0/24", Description: "Test IP range", Enabled: true},
		{CIDRBlock: "10.0.0.0/8", Description: "Another test IP range", Enabled: false},
	}

	// Call the ToAnnotationString() function
	result := ipRanges.ToAnnotationString()

	// Check that the resulting string is formatted correctly
	expectedResult := "192.168.0.0/24,10.0.0.0/8"
	assert.Equal(t, expectedResult, result)

	// Test that the ToAnnotationString() function returns an empty string for a nil AllowedIPRanges slice
	var nilIPRanges *AllowedIPRanges
	assert.Equal(t, "", nilIPRanges.ToAnnotationString())
}

func TestAllowedIPRangesAreValid(t *testing.T) {
	// Define some test data
	validIPRanges := AllowedIPRanges{
		{CIDRBlock: "192.168.0.0/24", Description: "Test IP range", Enabled: true},
		{CIDRBlock: "10.0.0.0/8", Description: "Another test IP range", Enabled: false},
	}
	invalidIPRanges := AllowedIPRanges{
		{CIDRBlock: "192.168.0.0/24", Description: "Test IP range", Enabled: true},
		{CIDRBlock: "10.0.0.0/33", Description: "Invalid IP range", Enabled: false},
	}

	// Test that the AreValid() function returns true for a valid AllowedIPRanges slice
	assert.True(t, validIPRanges.AreValid())

	// Test that the AreValid() function returns false for an invalid AllowedIPRanges slice
	assert.False(t, invalidIPRanges.AreValid())

	// Test that the AreValid() function returns true for a nil AllowedIPRanges slice
	var nilIPRanges *AllowedIPRanges
	assert.True(t, nilIPRanges.AreValid())
}

func TestAllowedIPRangesAllRulesAreDisabled(t *testing.T) {
	t.Run("returns true when AllowedIPRanges is nil", func(t *testing.T) {
		var a *AllowedIPRanges

		actual := a.AllRulesAreDisabled()

		require.True(t, actual)
	})

	t.Run("returns true when all rules are disabled", func(t *testing.T) {
		a := AllowedIPRanges{
			{Enabled: false},
			{Enabled: false},
			{Enabled: false},
		}

		actual := a.AllRulesAreDisabled()

		require.True(t, actual)
	})

	t.Run("returns false when at least one rule is enabled", func(t *testing.T) {
		a := AllowedIPRanges{
			{Enabled: false},
			{Enabled: true},
			{Enabled: false},
		}

		actual := a.AllRulesAreDisabled()

		require.False(t, actual)
	})
}

func TestPodProbeOverridesValue(t *testing.T) {
	t.Run("nil PodProbeOverrides", func(t *testing.T) {
		var probeOverrides *PodProbeOverrides
		value, err := probeOverrides.Value()
		assert.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("empty PodProbeOverrides", func(t *testing.T) {
		probeOverrides := &PodProbeOverrides{}
		value, err := probeOverrides.Value()
		assert.NoError(t, err)
		assert.NotNil(t, value)

		expected := `{}`
		assert.JSONEq(t, expected, string(value.([]byte)))
	})

	t.Run("PodProbeOverrides with values", func(t *testing.T) {
		probeOverrides := &PodProbeOverrides{
			LivenessProbeOverride: &corev1.Probe{
				FailureThreshold:    5,
				InitialDelaySeconds: 30,
				TimeoutSeconds:      10,
			},
			ReadinessProbeOverride: &corev1.Probe{
				FailureThreshold:    3,
				InitialDelaySeconds: 15,
				TimeoutSeconds:      5,
			},
		}
		value, err := probeOverrides.Value()
		assert.NoError(t, err)
		assert.NotNil(t, value)

		// Unmarshal to verify JSON structure
		var result PodProbeOverrides
		err = json.Unmarshal(value.([]byte), &result)
		assert.NoError(t, err)
		assert.Equal(t, int32(5), result.LivenessProbeOverride.FailureThreshold)
		assert.Equal(t, int32(30), result.LivenessProbeOverride.InitialDelaySeconds)
		assert.Equal(t, int32(10), result.LivenessProbeOverride.TimeoutSeconds)
		assert.Equal(t, int32(3), result.ReadinessProbeOverride.FailureThreshold)
		assert.Equal(t, int32(15), result.ReadinessProbeOverride.InitialDelaySeconds)
		assert.Equal(t, int32(5), result.ReadinessProbeOverride.TimeoutSeconds)
	})
}

func TestPodProbeOverridesScan(t *testing.T) {
	t.Run("nil source", func(t *testing.T) {
		var probeOverrides PodProbeOverrides
		err := probeOverrides.Scan(nil)
		assert.NoError(t, err)
	})

	t.Run("invalid source type", func(t *testing.T) {
		var probeOverrides PodProbeOverrides
		err := probeOverrides.Scan("invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not assert type of PodProbeOverrides")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		var probeOverrides PodProbeOverrides
		err := probeOverrides.Scan([]byte("invalid json"))
		assert.Error(t, err)
	})

	t.Run("empty JSON object", func(t *testing.T) {
		var probeOverrides PodProbeOverrides
		err := probeOverrides.Scan([]byte(`{}`))
		assert.NoError(t, err)
		assert.Nil(t, probeOverrides.LivenessProbeOverride)
		assert.Nil(t, probeOverrides.ReadinessProbeOverride)
	})

	t.Run("valid JSON with probe overrides", func(t *testing.T) {
		jsonData := `{
			"LivenessProbeOverride": {
				"failureThreshold": 8,
				"initialDelaySeconds": 45,
				"timeoutSeconds": 15
			},
			"ReadinessProbeOverride": {
				"failureThreshold": 6,
				"initialDelaySeconds": 20,
				"timeoutSeconds": 8
			}
		}`

		var probeOverrides PodProbeOverrides
		err := probeOverrides.Scan([]byte(jsonData))
		assert.NoError(t, err)

		assert.NotNil(t, probeOverrides.LivenessProbeOverride)
		assert.Equal(t, int32(8), probeOverrides.LivenessProbeOverride.FailureThreshold)
		assert.Equal(t, int32(45), probeOverrides.LivenessProbeOverride.InitialDelaySeconds)
		assert.Equal(t, int32(15), probeOverrides.LivenessProbeOverride.TimeoutSeconds)

		assert.NotNil(t, probeOverrides.ReadinessProbeOverride)
		assert.Equal(t, int32(6), probeOverrides.ReadinessProbeOverride.FailureThreshold)
		assert.Equal(t, int32(20), probeOverrides.ReadinessProbeOverride.InitialDelaySeconds)
		assert.Equal(t, int32(8), probeOverrides.ReadinessProbeOverride.TimeoutSeconds)
	})
}

func TestPodProbeOverridesRoundTrip(t *testing.T) {
	t.Run("round trip serialization", func(t *testing.T) {
		original := &PodProbeOverrides{
			LivenessProbeOverride: &corev1.Probe{
				FailureThreshold:    10,
				SuccessThreshold:    2,
				InitialDelaySeconds: 60,
				PeriodSeconds:       20,
				TimeoutSeconds:      15,
			},
			ReadinessProbeOverride: &corev1.Probe{
				FailureThreshold:    7,
				SuccessThreshold:    3,
				InitialDelaySeconds: 30,
				PeriodSeconds:       10,
				TimeoutSeconds:      12,
			},
		}

		// Serialize using Value()
		value, err := original.Value()
		assert.NoError(t, err)
		assert.NotNil(t, value)

		// Deserialize using Scan()
		var result PodProbeOverrides
		err = result.Scan(value)
		assert.NoError(t, err)

		// Verify the round trip preserved all values
		assert.Equal(t, original.LivenessProbeOverride.FailureThreshold, result.LivenessProbeOverride.FailureThreshold)
		assert.Equal(t, original.LivenessProbeOverride.SuccessThreshold, result.LivenessProbeOverride.SuccessThreshold)
		assert.Equal(t, original.LivenessProbeOverride.InitialDelaySeconds, result.LivenessProbeOverride.InitialDelaySeconds)
		assert.Equal(t, original.LivenessProbeOverride.PeriodSeconds, result.LivenessProbeOverride.PeriodSeconds)
		assert.Equal(t, original.LivenessProbeOverride.TimeoutSeconds, result.LivenessProbeOverride.TimeoutSeconds)

		assert.Equal(t, original.ReadinessProbeOverride.FailureThreshold, result.ReadinessProbeOverride.FailureThreshold)
		assert.Equal(t, original.ReadinessProbeOverride.SuccessThreshold, result.ReadinessProbeOverride.SuccessThreshold)
		assert.Equal(t, original.ReadinessProbeOverride.InitialDelaySeconds, result.ReadinessProbeOverride.InitialDelaySeconds)
		assert.Equal(t, original.ReadinessProbeOverride.PeriodSeconds, result.ReadinessProbeOverride.PeriodSeconds)
		assert.Equal(t, original.ReadinessProbeOverride.TimeoutSeconds, result.ReadinessProbeOverride.TimeoutSeconds)
	})
}

func TestPodProbeOverridesSimulatedDatabaseScenario(t *testing.T) {
	t.Run("simulates the original database error scenario", func(t *testing.T) {
		// This test simulates what happens when trying to store an installation
		// with both liveness and readiness probe overrides in the database.
		// Before our fix, this would fail with:
		// "sql: converting argument $21 type: unsupported type model.PodProbeOverrides, a struct"

		installation := &Installation{
			PodProbeOverrides: &PodProbeOverrides{
				LivenessProbeOverride: &corev1.Probe{
					FailureThreshold:    10,
					InitialDelaySeconds: 60,
					TimeoutSeconds:      15,
				},
				ReadinessProbeOverride: &corev1.Probe{
					FailureThreshold:    5,
					InitialDelaySeconds: 30,
					TimeoutSeconds:      8,
				},
			},
		}

		// Simulate what the SQL driver would do when trying to store this value
		value, err := installation.PodProbeOverrides.Value()
		assert.NoError(t, err, "PodProbeOverrides should be serializable to database value")
		assert.NotNil(t, value, "Database value should not be nil")

		// Verify the value is valid JSON
		var result PodProbeOverrides
		err = json.Unmarshal(value.([]byte), &result)
		assert.NoError(t, err, "Database value should be valid JSON")

		// Verify we can read it back correctly
		var scannedOverrides PodProbeOverrides
		err = scannedOverrides.Scan(value)
		assert.NoError(t, err, "Should be able to scan value back from database")

		// Verify the round trip preserves data
		assert.Equal(t, installation.PodProbeOverrides.LivenessProbeOverride.FailureThreshold, scannedOverrides.LivenessProbeOverride.FailureThreshold)
		assert.Equal(t, installation.PodProbeOverrides.LivenessProbeOverride.InitialDelaySeconds, scannedOverrides.LivenessProbeOverride.InitialDelaySeconds)
		assert.Equal(t, installation.PodProbeOverrides.LivenessProbeOverride.TimeoutSeconds, scannedOverrides.LivenessProbeOverride.TimeoutSeconds)

		assert.Equal(t, installation.PodProbeOverrides.ReadinessProbeOverride.FailureThreshold, scannedOverrides.ReadinessProbeOverride.FailureThreshold)
		assert.Equal(t, installation.PodProbeOverrides.ReadinessProbeOverride.InitialDelaySeconds, scannedOverrides.ReadinessProbeOverride.InitialDelaySeconds)
		assert.Equal(t, installation.PodProbeOverrides.ReadinessProbeOverride.TimeoutSeconds, scannedOverrides.ReadinessProbeOverride.TimeoutSeconds)

		// This test passing means the original error:
		// "sql: converting argument $21 type: unsupported type model.PodProbeOverrides, a struct"
		// should no longer occur because PodProbeOverrides now implements driver.Valuer
	})
}
