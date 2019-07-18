package model_test

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
)

func TestCheckProvider(t *testing.T) {
	var sizeTests = []struct {
		provider               string
		expectedProviderString string
		expectError            bool
	}{
		{"aws", "aws", false},
		{"AWS", "aws", false},
		{"Aws", "aws", false},
		{"gce", "gce", true},
		{"GCE", "gce", true},
		{"Gce", "gce", true},
		{"azure", "azure", true},
		{"AZURE", "azure", true},
		{"Azure", "azure", true},
	}

	for _, tt := range sizeTests {
		t.Run(tt.provider, func(t *testing.T) {
			provider, err := model.CheckProvider(tt.provider)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, provider, tt.expectedProviderString)
		})
	}
}
