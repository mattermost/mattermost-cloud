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
