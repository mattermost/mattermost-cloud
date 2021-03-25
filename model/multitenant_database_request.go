// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"net/url"
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
