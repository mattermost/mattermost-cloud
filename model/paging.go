// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"net/url"
	"strconv"
)

// Paging represent paging filter.
type Paging struct {
	Page           int
	PerPage        int
	IncludeDeleted bool
}

// AddToQuery adds paging filter to query values.
func (p *Paging) AddToQuery(q url.Values) {
	q.Add("page", strconv.Itoa(p.Page))
	q.Add("per_page", strconv.Itoa(p.PerPage))
	if p.IncludeDeleted {
		q.Add("include_deleted", "true")
	}
}

// AllPagesNotDeleted if paging filter returning all not deleted elements.
func AllPagesNotDeleted() Paging {
	return Paging{
		Page:           0,
		PerPage:        AllPerPage,
		IncludeDeleted: false,
	}
}

// AllPagesWithDeleted if paging filter returning all elements including deleted ones.
func AllPagesWithDeleted() Paging {
	return Paging{
		Page:           0,
		PerPage:        AllPerPage,
		IncludeDeleted: true,
	}
}
