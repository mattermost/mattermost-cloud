// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestNewDBMigrationOperationFromReader(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		dBMigrationOperation, err := NewDBMigrationOperationFromReader(bytes.NewReader([]byte(
			"",
		)))
		require.NoError(t, err)
		require.Equal(t, &InstallationDBMigrationOperation{}, dBMigrationOperation)
	})

	t.Run("invalid", func(t *testing.T) {
		dBMigrationOperation, err := NewDBMigrationOperationFromReader(bytes.NewReader([]byte(
			"{test",
		)))
		require.Error(t, err)
		require.Nil(t, dBMigrationOperation)
	})

	t.Run("valid", func(t *testing.T) {
		dBMigrationOperation, err := NewDBMigrationOperationFromReader(bytes.NewReader([]byte(
			`{"ID":"id", "InstallationID": "installation", "RequestAt": 10, "State": "installation-db-migration-requested"}`,
		)))
		require.NoError(t, err)
		require.Equal(t, &InstallationDBMigrationOperation{
			ID:             "id",
			InstallationID: "installation",
			RequestAt:      10,
			State:          InstallationDBMigrationStateRequested,
		}, dBMigrationOperation)
	})
}

func TestNewDBMigrationOperationsFromReader(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		dBMigrationOperations, err := NewDBMigrationOperationsFromReader(bytes.NewReader([]byte(
			"",
		)))
		require.NoError(t, err)
		require.Equal(t, []*InstallationDBMigrationOperation{}, dBMigrationOperations)
	})

	t.Run("invalid", func(t *testing.T) {
		dBMigrationOperations, err := NewDBMigrationOperationsFromReader(bytes.NewReader([]byte(
			"{test",
		)))
		require.Error(t, err)
		require.Nil(t, dBMigrationOperations)
	})

	t.Run("valid", func(t *testing.T) {
		dBMigrationOperations, err := NewDBMigrationOperationsFromReader(bytes.NewReader([]byte(
			`[
	{"ID":"id", "InstallationID": "installation", "RequestAt": 10, "State": "installation-db-migration-requested"},
	{"ID":"id2", "InstallationID": "installation2", "RequestAt": 20, "State": "installation-db-migration-requested"}
]`,
		)))
		require.NoError(t, err)
		require.Equal(t, []*InstallationDBMigrationOperation{
			{
				ID:             "id",
				InstallationID: "installation",
				RequestAt:      10,
				State:          InstallationDBMigrationStateRequested,
			},
			{
				ID:             "id2",
				InstallationID: "installation2",
				RequestAt:      20,
				State:          InstallationDBMigrationStateRequested,
			},
		}, dBMigrationOperations)
	})
}

func TestInstallationDBMigrationOperation_ValidTransitionState(t *testing.T) {
	// Couple of tests to verify mechanism is working - we can add more for specific cases
	for _, testCase := range []struct {
		oldState InstallationDBMigrationOperationState
		newState InstallationDBMigrationOperationState
		isValid  bool
	}{
		{
			oldState: InstallationDBMigrationStateSucceeded,
			newState: InstallationDBMigrationStateRollbackRequested,
			isValid:  true,
		},
		{
			oldState: InstallationDBMigrationStateRefreshSecrets,
			newState: InstallationDBMigrationStateRollbackRequested,
			isValid:  false,
		},
	} {
		t.Run(string(testCase.oldState)+" to "+string(testCase.newState), func(t *testing.T) {
			dbMigration := &InstallationDBMigrationOperation{State: testCase.oldState}

			isValid := dbMigration.ValidTransitionState(testCase.newState)
			assert.Equal(t, testCase.isValid, isValid)
		})
	}
}
