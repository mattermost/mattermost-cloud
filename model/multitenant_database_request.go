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

// GetDatabasesRequest describes the parameters to request a list of multitenant databases.
type GetDatabasesRequest struct {
	Paging
	VpcID        string
	DatabaseType string
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *GetDatabasesRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	q.Add("vpc_id", request.VpcID)
	q.Add("database_type", request.DatabaseType)
	request.Paging.AddToQuery(q)

	u.RawQuery = q.Encode()
}

// PatchDatabaseRequest specifies the parameters for an updated multitenant
// database.
type PatchDatabaseRequest struct {
	MaxInstallationsPerLogicalDatabase *int64
}

// Validate validates the values of a multitenant database patch request.
func (p *PatchDatabaseRequest) Validate() error {
	if p.MaxInstallationsPerLogicalDatabase != nil && *p.MaxInstallationsPerLogicalDatabase < int64(1) {
		return errors.New("MaxInstallationsPerLogicalDatabase must be 1 or greater")
	}

	return nil
}

// Apply applies the patch to the given multitenant database.
func (p *PatchDatabaseRequest) Apply(database *MultitenantDatabase) bool {
	var applied bool

	if database.DatabaseType == DatabaseEngineTypePostgresProxy {
		if p.MaxInstallationsPerLogicalDatabase != nil && *p.MaxInstallationsPerLogicalDatabase != database.MaxInstallationsPerLogicalDatabase {
			applied = true
			database.MaxInstallationsPerLogicalDatabase = *p.MaxInstallationsPerLogicalDatabase
		}
	}

	return applied
}

// NewPatchDatabaseRequestFromReader will create a PatchDatabaseRequest from an
// io.Reader with JSON data.
func NewPatchDatabaseRequestFromReader(reader io.Reader) (*PatchDatabaseRequest, error) {
	var patchDatabaseRequest PatchDatabaseRequest
	err := json.NewDecoder(reader).Decode(&patchDatabaseRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode patch database request")
	}

	err = patchDatabaseRequest.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "invalid patch database request")
	}

	return &patchDatabaseRequest, nil
}
