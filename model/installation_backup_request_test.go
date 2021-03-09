// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"bytes"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInstallationBackupRequestFromReader(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		backupRequest, err := NewInstallationBackupRequestFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, &InstallationBackupRequest{}, backupRequest)
	})

	t.Run("invalid", func(t *testing.T) {
		backupRequest, err := NewInstallationBackupRequestFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, backupRequest)
	})

	t.Run("valid", func(t *testing.T) {
		backupRequest, err := NewInstallationBackupRequestFromReader(bytes.NewReader([]byte(
			`{"InstallationID":"installation-1"}`,
		)))
		require.NoError(t, err)
		require.Equal(t, &InstallationBackupRequest{InstallationID: "installation-1"}, backupRequest)
	})
}

func TestGetInstallationBackupsRequest_ApplyToURL(t *testing.T) {
	req := &GetInstallationBackupsRequest{
		InstallationID:        "my-installation",
		ClusterInstallationID: "my-ci",
		State:                 "failed",
		Page:                  1,
		PerPage:               5,
		IncludeDeleted:        true,
	}

	u, err := url.Parse("https://provisioner/backups")
	require.NoError(t, err)

	req.ApplyToURL(u)

	assert.Equal(t, req.InstallationID, u.Query().Get("installation"))
	assert.Equal(t, req.ClusterInstallationID, u.Query().Get("cluster_installation"))
	assert.Equal(t, req.State, u.Query().Get("state"))
	assert.Equal(t, "1", u.Query().Get("page"))
	assert.Equal(t, "5", u.Query().Get("per_page"))
	assert.Equal(t, "true", u.Query().Get("include_deleted"))
}
