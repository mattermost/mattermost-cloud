package model_test

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestEnvValid(t *testing.T) {
	var testCases = []struct {
		testName     string
		requireError bool
		envVarMap    model.EnvVarMap
	}{
		{
			"empty",
			false,
			model.EnvVarMap{},
		},
		{
			"value",
			false,
			model.EnvVarMap{"key1": {Value: "value1"}},
		},
		{
			"value from",
			false,
			model.EnvVarMap{"key1": {ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: "secretKey1",
				},
			}}},
		},
		{
			"no value or value from",
			true,
			model.EnvVarMap{"key1": {}},
		},
		{
			"valid and invalid",
			true,
			model.EnvVarMap{
				"key1": {Value: "value1"},
				"key2": {},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			if tc.requireError {
				assert.Error(t, tc.envVarMap.Validate())
			} else {
				assert.NoError(t, tc.envVarMap.Validate())
			}
		})
	}
}

func TestEnvClearOrPatch(t *testing.T) {
	var testCases = []struct {
		testName          string
		expectPatch       bool
		old               model.EnvVarMap
		new               model.EnvVarMap
		expectedEnvVarMap model.EnvVarMap
	}{
		{
			"nil old and new EnvVarMap",
			false,
			nil,
			nil,
			nil,
		},
		{
			"nil old EnvVarMap",
			true,
			nil,
			model.EnvVarMap{
				"key1": {Value: "value1"},
			},
			model.EnvVarMap{
				"key1": {Value: "value1"},
			},
		},
		{
			"nil new EnvVarMap",
			true,
			model.EnvVarMap{
				"key1": {Value: "value1"},
			},
			nil,
			nil,
		},
		{
			"empty old and nil new EnvVarMap",
			false,
			model.EnvVarMap{},
			nil,
			nil,
		},
		{
			"empty old EnvVarMap",
			true,
			model.EnvVarMap{},
			model.EnvVarMap{
				"key1": {Value: "patch1"},
			},
			model.EnvVarMap{
				"key1": {Value: "patch1"},
			},
		},
		{
			"complex",
			true,
			model.EnvVarMap{
				"key1": {Value: "value1"},
				"key3": {Value: "value1"},
			},
			model.EnvVarMap{
				"key2": {Value: "patch1"},
				"key3": {Value: "patch1"},
			},
			model.EnvVarMap{
				"key1": {Value: "value1"},
				"key2": {Value: "patch1"},
				"key3": {Value: "patch1"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			patched := tc.old.ClearOrPatch(&tc.new)
			assert.Equal(t, tc.expectPatch, patched)
			assert.Equal(t, tc.expectedEnvVarMap, tc.old)
		})
	}
}

func TestEnvPatch(t *testing.T) {
	var testCases = []struct {
		testName          string
		expectPatch       bool
		old               model.EnvVarMap
		new               model.EnvVarMap
		expectedEnvVarMap model.EnvVarMap
	}{
		{
			"nil new EnvVarMap",
			false,
			model.EnvVarMap{
				"key1": {Value: "value1"},
			},
			nil,
			model.EnvVarMap{
				"key1": {Value: "value1"},
			},
		},
		{
			"no changes",
			false,
			model.EnvVarMap{
				"key1": {Value: "value1"},
			},
			model.EnvVarMap{
				"key1": {Value: "value1"},
			},
			model.EnvVarMap{
				"key1": {Value: "value1"},
			},
		},
		{
			"patch only old keys",
			true,
			model.EnvVarMap{
				"key1": {Value: "value1"},
			},
			model.EnvVarMap{
				"key1": {Value: "patch1"},
			},
			model.EnvVarMap{
				"key1": {Value: "patch1"},
			},
		},
		{
			"patch only new keys",
			true,
			model.EnvVarMap{
				"key1": {Value: "value1"},
			},
			model.EnvVarMap{
				"key2": {Value: "patch2"},
			},
			model.EnvVarMap{
				"key1": {Value: "value1"},
				"key2": {Value: "patch2"},
			},
		},
		{
			"patch new and old keys",
			true,
			model.EnvVarMap{
				"key1": {Value: "value1"},
				"key2": {Value: "value2"},
				"key3": {Value: "value3"},
				"key5": {Value: "value5"},
			},
			model.EnvVarMap{
				"key2": {Value: "patch2"},
				"key3": {Value: "patch3"},
				"key4": {Value: "patch4"},
			},
			model.EnvVarMap{
				"key1": {Value: "value1"},
				"key2": {Value: "patch2"},
				"key3": {Value: "patch3"},
				"key4": {Value: "patch4"},
				"key5": {Value: "value5"},
			},
		},
		{
			"delete keys",
			true,
			model.EnvVarMap{
				"key1": {Value: "value1"},
			},
			model.EnvVarMap{
				"key1": {},
			},
			model.EnvVarMap{},
		},
		{
			"delete nonexistent keys",
			false,
			model.EnvVarMap{
				"key1": {Value: "value1"},
			},
			model.EnvVarMap{
				"key2": {},
				"key3": {},
			},
			model.EnvVarMap{
				"key1": {Value: "value1"},
			},
		},
		{
			"complex",
			true,
			model.EnvVarMap{
				"key1": {Value: "value1"},
				"key2": {Value: "value2"},
				"key5": {Value: "value5"},
				"key6": {Value: "value6"},
				"key7": {Value: "value7"},
			},
			model.EnvVarMap{
				"key1": {Value: "patch1"},
				"key3": {Value: "patch3"},
				"key4": {},
				"key5": {},
				"key7": {Value: "patch7"},
			},
			model.EnvVarMap{
				"key1": {Value: "patch1"},
				"key2": {Value: "value2"},
				"key3": {Value: "patch3"},
				"key6": {Value: "value6"},
				"key7": {Value: "patch7"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			patched := tc.old.Patch(tc.new)
			assert.Equal(t, tc.expectPatch, patched)
			assert.Equal(t, tc.expectedEnvVarMap, tc.old)
		})
	}
}

func TestEnvToEnvList(t *testing.T) {
	var testCases = []struct {
		testName  string
		envVarMap model.EnvVarMap
		expected  []corev1.EnvVar
	}{
		{
			"empty",
			model.EnvVarMap{},
			[]corev1.EnvVar{},
		},
		{
			"value",
			model.EnvVarMap{"key1": {Value: "value1"}},
			[]corev1.EnvVar{
				{Name: "key1", Value: "value1"},
			},
		},
		{
			"value from",
			model.EnvVarMap{"key1": {ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: "secretKey",
				},
			}}},
			[]corev1.EnvVar{
				{Name: "key1", ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						Key: "secretKey",
					},
				}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.envVarMap.ToEnvList())
		})
	}
}
