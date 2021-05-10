// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
)

// GetMillis is a convenience method to get milliseconds since epoch.
func GetMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func applyPagingFilter(builder sq.SelectBuilder, paging model.Paging) sq.SelectBuilder {
	if paging.PerPage != model.AllPerPage {
		builder = builder.
			Limit(uint64(paging.PerPage)).
			Offset(uint64(paging.Page * paging.PerPage))
	}
	if !paging.IncludeDeleted {
		builder = builder.Where("DeleteAt = 0")
	}

	return builder
}
