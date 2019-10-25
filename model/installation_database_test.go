package model_test

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
)

func TestInternalDatabase(t *testing.T) {
	var testCases = []struct {
		databaseType   string
		expectInternal bool
	}{
		{"", false},
		{"unknown", false},
		{model.InstallationDatabaseMysqlOperator, true},
		{model.InstallationDatabaseAwsRDS, false},
	}

	for _, tc := range testCases {
		t.Run(tc.databaseType, func(t *testing.T) {
			installation := &model.Installation{
				Database: tc.databaseType,
			}

			assert.Equal(t, tc.expectInternal, installation.InternalDatabase())
		})
	}
}

func TestIsSupportedDatabase(t *testing.T) {
	var testCases = []struct {
		database        string
		expectSupported bool
	}{
		{"", false},
		{"unknown", false},
		{model.InstallationDatabaseMysqlOperator, true},
		{model.InstallationDatabaseAwsRDS, true},
	}

	for _, tc := range testCases {
		t.Run(tc.database, func(t *testing.T) {
			assert.Equal(t, tc.expectSupported, model.IsSupportedDatabase(tc.database))
		})
	}
}
