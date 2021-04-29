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

func TestNewInstallationDBRestorationOperationFromReader(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		installationDBRestorationOperation, err := NewInstallationDBRestorationOperationFromReader(bytes.NewReader([]byte(
			"",
		)))
		require.NoError(t, err)
		require.Equal(t, &InstallationDBRestorationOperation{}, installationDBRestorationOperation)
	})

	t.Run("invalid", func(t *testing.T) {
		installationDBRestorationOperation, err := NewInstallationDBRestorationOperationFromReader(bytes.NewReader([]byte(
			"{test",
		)))
		require.Error(t, err)
		require.Nil(t, installationDBRestorationOperation)
	})

	t.Run("valid", func(t *testing.T) {
		installationDBRestorationOperation, err := NewInstallationDBRestorationOperationFromReader(bytes.NewReader([]byte(
			`{"ID":"id", "InstallationID":"Installation", "BackupID": "backup", "RequestAt": 10}`,
		)))
		require.NoError(t, err)
		require.Equal(t, &InstallationDBRestorationOperation{
			ID:             "id",
			InstallationID: "Installation",
			BackupID:       "backup",
			RequestAt:      10,
		}, installationDBRestorationOperation)
	})
}

func TestNewInstallationDBRestorationOperationsFromReader(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		installationDBRestorationOperations, err := NewInstallationDBRestorationOperationsFromReader(bytes.NewReader([]byte(
			"",
		)))
		require.NoError(t, err)
		require.Equal(t, []*InstallationDBRestorationOperation{}, installationDBRestorationOperations)
	})

	t.Run("invalid", func(t *testing.T) {
		installationDBRestorationOperations, err := NewInstallationDBRestorationOperationsFromReader(bytes.NewReader([]byte(
			"{test",
		)))
		require.Error(t, err)
		require.Nil(t, installationDBRestorationOperations)
	})

	t.Run("valid", func(t *testing.T) {
		installationDBRestorationOperations, err := NewInstallationDBRestorationOperationsFromReader(bytes.NewReader([]byte(
			`[
	{"ID":"id", "InstallationID":"Installation", "BackupID": "backup", "RequestAt": 10},
	{"ID":"id2", "InstallationID":"Installation2", "BackupID": "backup2", "RequestAt": 20}
]`,
		)))
		require.NoError(t, err)
		require.Equal(t, []*InstallationDBRestorationOperation{
			{
				ID:             "id",
				InstallationID: "Installation",
				BackupID:       "backup",
				RequestAt:      10,
			},
			{
				ID:             "id2",
				InstallationID: "Installation2",
				BackupID:       "backup2",
				RequestAt:      20,
			},
		}, installationDBRestorationOperations)
	})
}

func TestEnsureInstallationReadyForDBRestoration(t *testing.T) {

	for _, testCase := range []struct {
		description   string
		installation  *Installation
		backup        *InstallationBackup
		errorContains string
	}{
		{
			description: "valid installation and backup",
			installation: &Installation{
				ID:        "abcd",
				State:     InstallationStateHibernating,
				Database:  InstallationDatabaseMultiTenantRDSPostgres,
				Filestore: InstallationFilestoreBifrost,
			},
			backup: &InstallationBackup{
				InstallationID: "abcd",
				State:          InstallationBackupStateBackupSucceeded,
			},
		},
		{
			description: "backup failed",
			installation: &Installation{
				ID:        "abcd",
				State:     InstallationStateHibernating,
				Database:  InstallationDatabaseMultiTenantRDSPostgres,
				Filestore: InstallationFilestoreBifrost,
			},
			backup: &InstallationBackup{
				InstallationID: "abcd",
				State:          InstallationBackupStateBackupFailed,
			},
			errorContains: "Only backups in succeeded state can be restored",
		},
		{
			description: "backup not matching installation",
			installation: &Installation{
				ID:        "abcd",
				State:     InstallationStateHibernating,
				Database:  InstallationDatabaseMultiTenantRDSPostgres,
				Filestore: InstallationFilestoreBifrost,
			},
			backup: &InstallationBackup{
				InstallationID: "efgh",
				State:          InstallationBackupStateBackupSucceeded,
			},
			errorContains: "Backup belongs to different installation",
		},
		{
			description: "backup deleted",
			installation: &Installation{
				ID:        "abcd",
				State:     InstallationStateHibernating,
				Database:  InstallationDatabaseMultiTenantRDSPostgres,
				Filestore: InstallationFilestoreBifrost,
			},
			backup: &InstallationBackup{
				InstallationID: "abcd",
				State:          InstallationBackupStateBackupSucceeded,
				DeleteAt:       1,
			},
			errorContains: "Backup files are deleted",
		},
		{
			description: "installation invalid state",
			installation: &Installation{
				ID:        "abcd",
				State:     InstallationStateStable,
				Database:  InstallationDatabaseMultiTenantRDSPostgres,
				Filestore: InstallationFilestoreBifrost,
			},
			backup: &InstallationBackup{
				InstallationID: "abcd",
				State:          InstallationBackupStateBackupSucceeded,
			},
			errorContains: "invalid installation state",
		},
		{
			description: "invalid db",
			installation: &Installation{
				ID:        "abcd",
				State:     InstallationStateHibernating,
				Database:  InstallationDatabaseMultiTenantRDSMySQL,
				Filestore: InstallationFilestoreBifrost,
			},
			backup: &InstallationBackup{
				InstallationID: "abcd",
				State:          InstallationBackupStateBackupSucceeded,
			},
			errorContains: "invalid installation database",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			err := EnsureInstallationReadyForDBRestoration(testCase.installation, testCase.backup)
			if testCase.errorContains == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errorContains)
			}
		})
	}
}

func TestDetermineAfterRestorationState(t *testing.T) {

	for _, testCase := range []struct {
		description string
		state       string
		expected    string
	}{
		{
			description: "hibernating",
			state:       InstallationStateHibernating,
			expected:    InstallationStateHibernating,
		},
		{
			description: "db-migration",
			state:       InstallationStateDBMigrationInProgress,
			expected:    InstallationStateDBMigrationInProgress,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			installation := &Installation{State: testCase.state}
			targetState, err := DetermineAfterRestorationState(installation)
			require.NoError(t, err)
			assert.Equal(t, testCase.expected, targetState)
		})
	}

}
