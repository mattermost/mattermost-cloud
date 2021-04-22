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

// InstallationDBRestorationRequest represents request for installation restoration.
type InstallationDBRestorationRequest struct {
	InstallationID string
	BackupID       string
}

// NewInstallationDBRestorationRequestFromReader will create a InstallationDBRestorationRequest from an
// io.Reader with JSON data.
func NewInstallationDBRestorationRequestFromReader(reader io.Reader) (*InstallationDBRestorationRequest, error) {
	var restoreRequest InstallationDBRestorationRequest
	err := json.NewDecoder(reader).Decode(&restoreRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode installation db restore request")
	}

	return &restoreRequest, nil
}

// GetInstallationDBRestorationOperationsRequest describes the parameters to request
// a list of installation restoration operations.
type GetInstallationDBRestorationOperationsRequest struct {
	Paging
	InstallationID        string
	ClusterInstallationID string
	State                 string
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *GetInstallationDBRestorationOperationsRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	q.Add("installation", request.InstallationID)
	q.Add("cluster_installation", request.ClusterInstallationID)
	q.Add("state", request.State)
	request.Paging.AddToQuery(q)

	u.RawQuery = q.Encode()
}
