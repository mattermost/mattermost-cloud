// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
)

func applyPagingFilter(builder sq.SelectBuilder, paging model.Paging, deleteAtTables ...string) sq.SelectBuilder {
	if paging.PerPage != model.AllPerPage {
		builder = builder.
			Limit(uint64(paging.PerPage)).
			Offset(uint64(paging.Page * paging.PerPage))
	}
	if !paging.IncludeDeleted {
		// Allow optionally specifying tables if the DeleteAt column is ambiguous
		// or should apply to multiple tables when query uses JOIN.
		if len(deleteAtTables) > 0 {
			for _, t := range deleteAtTables {
				if t == "" {
					continue
				}
				deleteAtCol := fmt.Sprintf("%s.DeleteAt", t)
				builder = builder.Where(fmt.Sprintf("%s = 0", deleteAtCol))
			}
		} else {
			builder = builder.Where("DeleteAt = 0")
		}
	}

	return builder
}
