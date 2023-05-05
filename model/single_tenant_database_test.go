// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model_test

import (
	"bytes"
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/require"
)

func TestNewSingleTenantDatabaseConfigurationFromReader(t *testing.T) {
	t.Run("empty data", func(t *testing.T) {
		config, err := model.NewSingleTenantDatabaseConfigurationFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, &model.SingleTenantDatabaseConfig{}, config)
	})

	t.Run("invalid data", func(t *testing.T) {
		config, err := model.NewSingleTenantDatabaseConfigurationFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, config)
	})

	t.Run("valid data", func(t *testing.T) {
		config, err := model.NewSingleTenantDatabaseConfigurationFromReader(bytes.NewReader([]byte(`{
			"PrimaryInstanceType":"db.r5.xlarge",
			"ReplicaInstanceType":"db.r5.large",
			"ReplicasCount":3
		}`)))
		require.NoError(t, err)

		expected := &model.SingleTenantDatabaseConfig{
			PrimaryInstanceType: "db.r5.xlarge",
			ReplicaInstanceType: "db.r5.large",
			ReplicasCount:       3,
		}
		require.Equal(t, expected, config)
	})
}

func TestNewSingleTenantDatabaseRequestFromReader(t *testing.T) {
	t.Run("empty data", func(t *testing.T) {
		request, err := model.NewSingleTenantDatabaseRequestFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)

		expected := &model.SingleTenantDatabaseRequest{
			PrimaryInstanceType: "db.t4g.medium",
			ReplicaInstanceType: "db.t4g.medium",
			ReplicasCount:       0,
		}
		require.Equal(t, expected, request)
	})

	t.Run("invalid data", func(t *testing.T) {
		request, err := model.NewSingleTenantDatabaseRequestFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, request)
	})

	t.Run("valid data", func(t *testing.T) {
		request, err := model.NewSingleTenantDatabaseRequestFromReader(bytes.NewReader([]byte(`{
			"PrimaryInstanceType":"db.r5.xlarge",
			"ReplicaInstanceType":"db.r5.large",
			"ReplicasCount":3
		}`)))
		require.NoError(t, err)

		expected := &model.SingleTenantDatabaseRequest{
			PrimaryInstanceType: "db.r5.xlarge",
			ReplicaInstanceType: "db.r5.large",
			ReplicasCount:       3,
		}
		expected.SetDefaults()
		require.Equal(t, expected, request)
	})
}

func TestValidateSingleTenantDatabaseRequest(t *testing.T) {
	for _, testCase := range []struct {
		description string
		request     model.SingleTenantDatabaseRequest
		valid       bool
	}{
		{
			description: "valid request",
			request:     model.SingleTenantDatabaseRequest{ReplicasCount: 2},
			valid:       true,
		},
		{
			description: "default request",
			request:     model.SingleTenantDatabaseRequest{},
			valid:       true,
		},
		{
			description: "replicas < 0",
			request:     model.SingleTenantDatabaseRequest{ReplicasCount: -1},
			valid:       false,
		},
		{
			description: "replicas > 15",
			request:     model.SingleTenantDatabaseRequest{ReplicasCount: 16},
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

func TestToDBConfig(t *testing.T) {

	for _, testCase := range []struct {
		description    string
		request        model.SingleTenantDatabaseRequest
		database       string
		expectedConfig *model.SingleTenantDatabaseConfig
	}{
		{
			description:    "not single tenant RDS",
			request:        model.SingleTenantDatabaseRequest{PrimaryInstanceType: "db.r5.xlarge"},
			database:       model.InstallationDatabaseMultiTenantRDSPostgres,
			expectedConfig: nil,
		},
		{
			description: "single tenant RDS",
			request: model.SingleTenantDatabaseRequest{
				PrimaryInstanceType: "db.r5.xlarge",
				ReplicaInstanceType: "db.r5.large",
				ReplicasCount:       10,
			},
			database: model.InstallationDatabaseSingleTenantRDSPostgres,
			expectedConfig: &model.SingleTenantDatabaseConfig{
				PrimaryInstanceType: "db.r5.xlarge",
				ReplicaInstanceType: "db.r5.large",
				ReplicasCount:       10,
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			config := testCase.request.ToDBConfig(testCase.database)
			require.Equal(t, testCase.expectedConfig, config)
		})
	}
}
