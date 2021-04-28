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

// InstallationDBMigrationRequest represent request for installation database migration.
type InstallationDBMigrationRequest struct {
	InstallationID string

	DestinationDatabase string

	DestinationMultiTenant *MultiTenantDBMigrationData `json:"DestinationMultiTenant,omitempty"`
}

// NewInstallationDBMigrationRequestFromReader will create a InstallationDBMigrationRequest from an
// io.Reader with JSON data.
func NewInstallationDBMigrationRequestFromReader(reader io.Reader) (*InstallationDBMigrationRequest, error) {
	var installationDBMigrationRequest InstallationDBMigrationRequest
	err := json.NewDecoder(reader).Decode(&installationDBMigrationRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode InstallationDBMigrationRequest")
	}

	return &installationDBMigrationRequest, nil
}

// GetInstallationDBMigrationOperationsRequest describes the parameters to request
// a list of installation db migration operations.
type GetInstallationDBMigrationOperationsRequest struct {
	Paging
	InstallationID        string
	ClusterInstallationID string
	State                 string
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *GetInstallationDBMigrationOperationsRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	q.Add("installation", request.InstallationID)
	q.Add("cluster_installation", request.ClusterInstallationID)
	q.Add("state", request.State)
	request.Paging.AddToQuery(q)

	u.RawQuery = q.Encode()
}
