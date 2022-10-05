// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model_test

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/require"
)

func TestValidateExternalDatabaseRequest(t *testing.T) {
	for _, testCase := range []struct {
		description string
		request     model.ExternalDatabaseRequest
		valid       bool
	}{
		{
			description: "valid request",
			request:     model.ExternalDatabaseRequest{SecretName: "test-secret"},
			valid:       true,
		},
		{
			description: "empty secret name",
			request:     model.ExternalDatabaseRequest{},
			valid:       false,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			err := testCase.request.Validate()
			if testCase.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestExternalDatabaseRequestToDBConfig(t *testing.T) {
	for _, testCase := range []struct {
		description    string
		database       string
		request        model.ExternalDatabaseRequest
		expectedConfig *model.ExternalDatabaseConfig
	}{
		{
			description:    "not external database",
			database:       model.InstallationDatabaseMultiTenantRDSPostgres,
			request:        model.ExternalDatabaseRequest{SecretName: "test-secret"},
			expectedConfig: nil,
		},
		{
			description: "external database",
			database:    model.InstallationDatabaseExternal,
			request: model.ExternalDatabaseRequest{
				SecretName: "test-secret",
			},
			expectedConfig: &model.ExternalDatabaseConfig{
				SecretName: "test-secret",
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			config := testCase.request.ToDBConfig(testCase.database)
			require.Equal(t, testCase.expectedConfig, config)
		})
	}
}
