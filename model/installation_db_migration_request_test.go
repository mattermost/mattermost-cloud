// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewInstallationDBMigrationRequestFromReader(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		installationDBMigrationRequest, err := NewInstallationDBMigrationRequestFromReader(bytes.NewReader([]byte(
			"",
		)))
		require.NoError(t, err)
		require.Equal(t, &InstallationDBMigrationRequest{}, installationDBMigrationRequest)
	})

	t.Run("invalid", func(t *testing.T) {
		installationDBMigrationRequest, err := NewInstallationDBMigrationRequestFromReader(bytes.NewReader([]byte(
			"{test",
		)))
		require.Error(t, err)
		require.Nil(t, installationDBMigrationRequest)
	})

	t.Run("valid", func(t *testing.T) {
		installationDBMigrationRequest, err := NewInstallationDBMigrationRequestFromReader(bytes.NewReader([]byte(
			`{"InstallationID": "installation", "DestinationDatabase":"pg", "DestinationMultiTenant": {"DatabaseID":"db"}}`,
		)))
		require.NoError(t, err)
		require.Equal(t, &InstallationDBMigrationRequest{
			InstallationID:         "installation",
			DestinationDatabase:    "pg",
			DestinationMultiTenant: &MultiTenantDBMigrationData{DatabaseID: "db"},
		}, installationDBMigrationRequest)
	})
}
