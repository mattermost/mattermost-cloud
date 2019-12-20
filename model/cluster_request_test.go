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
