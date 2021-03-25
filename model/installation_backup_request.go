// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
	"net/url"

	"github.com/pkg/errors"
)

// InstallationBackupRequest represents request for installation backup.
type InstallationBackupRequest struct {
	InstallationID string
}

// NewInstallationBackupRequestFromReader will create a InstallationBackup from an
// io.Reader with JSON data.
func NewInstallationBackupRequestFromReader(reader io.Reader) (*InstallationBackupRequest, error) {
	var backupRequest InstallationBackupRequest
	err := json.NewDecoder(reader).Decode(&backupRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode backup request")
	}

	return &backupRequest, nil
}

// GetInstallationBackupsRequest describes the parameters to request a list of installation backups.
type GetInstallationBackupsRequest struct {
	Paging
	InstallationID        string
	ClusterInstallationID string
	State                 string
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *GetInstallationBackupsRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	q.Add("installation", request.InstallationID)
	q.Add("cluster_installation", request.ClusterInstallationID)
	q.Add("state", request.State)
	request.Paging.AddToQuery(q)

	u.RawQuery = q.Encode()
}
