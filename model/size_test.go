package model_test

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
)

func TestCheckSize(t *testing.T) {
	var testCases = []struct {
		size            string
		expectSupported bool
	}{
		{"", false},
		{"unknown", false},
		{model.SizeAlef500, true},
		{model.SizeAlef1000, true},
	}

	for _, tc := range testCases {
		t.Run(tc.size, func(t *testing.T) {
			assert.Equal(t, tc.expectSupported, model.IsSupportedClusterSize(tc.size))
		})
	}
}
