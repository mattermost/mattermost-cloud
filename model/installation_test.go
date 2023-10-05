// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"bytes"
	"encoding/json"
	"testing"

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
		GroupID:  sToP("group_id"),
		State:    InstallationStateStable,
	}

	clone := installation.Clone()
	require.Equal(t, installation, clone)

	// Verify changing pointers in the clone doesn't affect the original.
	clone.GroupID = sToP("new_group_id")
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
			GroupID:        sToP("group_id"),
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
				GroupID:        sToP("group_id1"),
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
				GroupID:        sToP("group_id2"),
				Version:        "version2",
				Name:           "dns2",
				License:        "this_is_my_license",
				MattermostEnv:  EnvVarMap{"key2": {Value: "value2"}},
				Affinity:       "affinity2",
				State:          "state2",
				CreateAt:       30,
				DeleteAt:       40,
				LockAcquiredBy: sToP("tester"),
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
			GroupID:  sToP("group_id"),
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
			GroupID:  sToP("group_id"),
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
			GroupID:  sToP("group_id"),
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
			GroupID:       sToP("group_id"),
			GroupSequence: iToP(2),
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
			GroupID:       sToP("group_id"),
			GroupSequence: iToP(1),
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
