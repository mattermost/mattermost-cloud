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

// GetMultitenantDatabasesRequest describes the parameters to request a list of
// multitenant databases.
type GetMultitenantDatabasesRequest struct {
	Paging
	VpcID        string
	DatabaseType string
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *GetMultitenantDatabasesRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	q.Add("vpc_id", request.VpcID)
	q.Add("database_type", request.DatabaseType)
	request.Paging.AddToQuery(q)

	u.RawQuery = q.Encode()
}

// PatchMultitenantDatabaseRequest specifies the parameters for an updated
// multitenant database.
type PatchMultitenantDatabaseRequest struct {
	MaxInstallationsPerLogicalDatabase *int64
}

// Validate validates the values of a multitenant database patch request.
func (p *PatchMultitenantDatabaseRequest) Validate() error {
	if p.MaxInstallationsPerLogicalDatabase != nil && *p.MaxInstallationsPerLogicalDatabase < int64(1) {
		return errors.New("MaxInstallationsPerLogicalDatabase must be 1 or greater")
	}

	return nil
}

// Apply applies the patch to the given multitenant database.
func (p *PatchMultitenantDatabaseRequest) Apply(database *MultitenantDatabase) bool {
	var applied bool

	if database.DatabaseType == DatabaseEngineTypePostgresProxy {
		if p.MaxInstallationsPerLogicalDatabase != nil && *p.MaxInstallationsPerLogicalDatabase != database.MaxInstallationsPerLogicalDatabase {
			applied = true
			database.MaxInstallationsPerLogicalDatabase = *p.MaxInstallationsPerLogicalDatabase
		}
	}

	return applied
}

// NewPatchMultitenantDatabaseRequestFromReader will create a PatchMultitenantDatabaseRequest
// from an io.Reader with JSON data.
func NewPatchMultitenantDatabaseRequestFromReader(reader io.Reader) (*PatchMultitenantDatabaseRequest, error) {
	var patchMultitenantDatabaseRequest PatchMultitenantDatabaseRequest
	err := json.NewDecoder(reader).Decode(&patchMultitenantDatabaseRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode patch multitenant database request")
	}

	err = patchMultitenantDatabaseRequest.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "invalid patch multitenant database request")
	}

	return &patchMultitenantDatabaseRequest, nil
}

// GetLogicalDatabasesRequest describes the parameters to request a list of
// logical databases.
type GetLogicalDatabasesRequest struct {
	Paging
	MultitenantDatabaseID string
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *GetLogicalDatabasesRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	q.Add("multitenant_database_id", request.MultitenantDatabaseID)
	request.Paging.AddToQuery(q)

	u.RawQuery = q.Encode()
}

// GetDatabaseSchemaRequest describes the parameters to request a list of
// database schemas.
type GetDatabaseSchemaRequest struct {
	Paging
	LogicalDatabaseID string
	InstallationID    string
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *GetDatabaseSchemaRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	q.Add("logical_database_id", request.LogicalDatabaseID)
	q.Add("installation_id", request.InstallationID)
	request.Paging.AddToQuery(q)

	u.RawQuery = q.Encode()
}
