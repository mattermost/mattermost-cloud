package model_test

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
)

func TestIsSupportedAffinity(t *testing.T) {
	var testCases = []struct {
		affinity        string
		expectSupported bool
	}{
		{"", false},
		{"unknown", false},
		{model.InstallationAffinityIsolated, true},
		{model.InstallationAffinityMultiTenant, true},
	}

	for _, tc := range testCases {
		t.Run(tc.affinity, func(t *testing.T) {
			assert.Equal(t, tc.expectSupported, model.IsSupportedAffinity(tc.affinity))
		})
	}
}
