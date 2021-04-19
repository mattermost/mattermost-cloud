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

func TestNewInstallationBackupFromReader(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		backupMetadata, err := NewInstallationBackupFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, &InstallationBackup{}, backupMetadata)
	})

	t.Run("invalid", func(t *testing.T) {
		backupMetadata, err := NewInstallationBackupFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, backupMetadata)
	})

	t.Run("valid", func(t *testing.T) {
		backupMetadata, err := NewInstallationBackupFromReader(bytes.NewReader([]byte(
			`{"ID":"metadata-1", "StartAt": 100, "InstallationID":"installation-1"}`,
		)))
		require.NoError(t, err)
		require.Equal(t, &InstallationBackup{ID: "metadata-1", StartAt: 100, InstallationID: "installation-1"}, backupMetadata)
	})
}

func TestNewInstallationBackupsFromReader(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		backupsMetadata, err := NewInstallationBackupsFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, []*InstallationBackup{}, backupsMetadata)
	})

	t.Run("invalid", func(t *testing.T) {
		backupsMetadata, err := NewInstallationBackupsFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, backupsMetadata)
	})

	t.Run("valid", func(t *testing.T) {
		backupsMetadata, err := NewInstallationBackupsFromReader(bytes.NewReader([]byte(
			`[
  {"ID":"metadata-1", "StartAt": 100, "InstallationID":"installation-1"},
  {"ID":"metadata-2", "RequestAt": 101, "InstallationID":"installation-2"}
]`,
		)))
		require.NoError(t, err)
		require.Equal(t, []*InstallationBackup{
			{ID: "metadata-1", StartAt: 100, InstallationID: "installation-1"},
			{ID: "metadata-2", RequestAt: 101, InstallationID: "installation-2"},
		}, backupsMetadata)
	})
}

func TestEnsureBackupRestoreCompatible(t *testing.T) {

	for _, testCase := range []struct {
		description   string
		installation  *Installation
		errorContains string
	}{
		{
			description: "valid installation",
			installation: &Installation{
				Database:  InstallationDatabaseMultiTenantRDSPostgres,
				Filestore: InstallationFilestoreBifrost,
			},
		},
		{
			description: "invalid db",
			installation: &Installation{
				Database:  InstallationDatabaseMultiTenantRDSMySQL,
				Filestore: InstallationFilestoreBifrost,
			},
			errorContains: "invalid installation database",
		},
		{
			description: "invalid file store",
			installation: &Installation{
				Database:  InstallationDatabaseMultiTenantRDSPostgres,
				Filestore: InstallationFilestoreMinioOperator,
			},
			errorContains: "invalid installation file store",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			err := EnsureBackupRestoreCompatible(testCase.installation)
			if testCase.errorContains == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errorContains)
			}
		})
	}
}

func TestEnsureInstallationReadyForBackup(t *testing.T) {

	for _, testCase := range []struct {
		description   string
		installation  *Installation
		errorContains string
	}{
		{
			description: "valid installation",
			installation: &Installation{
				State:     InstallationStateHibernating,
				Database:  InstallationDatabaseMultiTenantRDSPostgres,
				Filestore: InstallationFilestoreBifrost,
			},
		},
		{
			description: "not hibernating",
			installation: &Installation{
				State:     InstallationStateStable,
				Database:  InstallationDatabaseMultiTenantRDSPostgres,
				Filestore: InstallationFilestoreBifrost,
			},
			errorContains: "invalid installation state",
		},
		{
			description: "invalid db",
			installation: &Installation{
				State:     InstallationStateHibernating,
				Database:  InstallationDatabaseMultiTenantRDSMySQL,
				Filestore: InstallationFilestoreBifrost,
			},
			errorContains: "invalid installation database",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			err := EnsureInstallationReadyForBackup(testCase.installation)
			if testCase.errorContains == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errorContains)
			}
		})
	}
}
