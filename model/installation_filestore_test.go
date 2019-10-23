package model_test

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
)

func TestInternalFilestore(t *testing.T) {
	var testCases = []struct {
		filestoreType  string
		expectInternal bool
	}{
		{"", false},
		{"unknown", false},
		{model.InstallationFilestoreMinioOperator, true},
		{model.InstallationFilestoreAwsS3, false},
	}

	for _, tc := range testCases {
		t.Run(tc.filestoreType, func(t *testing.T) {
			installation := &model.Installation{
				Filestore: tc.filestoreType,
			}

			assert.Equal(t, tc.expectInternal, installation.InternalFilestore())
		})
	}
}

func TestIsSupportedFilestore(t *testing.T) {
	var testCases = []struct {
		filestore       string
		expectSupported bool
	}{
		{"", false},
		{"unknown", false},
		{model.InstallationFilestoreMinioOperator, true},
		{model.InstallationFilestoreAwsS3, true},
	}

	for _, tc := range testCases {
		t.Run(tc.filestore, func(t *testing.T) {
			assert.Equal(t, tc.expectSupported, model.IsSupportedFilestore(tc.filestore))
		})
	}
}
