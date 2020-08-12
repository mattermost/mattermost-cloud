// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"net/url"
	"strconv"
)

// GetDatabasesRequest describes the parameters to request a list of multitenant databases.
type GetDatabasesRequest struct {
	VpcID        string
	DatabaseType string
	Page         int
	PerPage      int
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *GetDatabasesRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	q.Add("page", strconv.Itoa(request.Page))
	q.Add("per_page", strconv.Itoa(request.PerPage))
	q.Add("vpc_id", request.VpcID)
	q.Add("database_type", request.DatabaseType)

	u.RawQuery = q.Encode()
}
