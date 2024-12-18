// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"
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

type UniqueConstraintError struct {
}

func (e *UniqueConstraintError) Error() string {
	return "unique constraint violation"
}

// isUniqueConstraintViolation checks if the error is a unique constraint violation.
func isUniqueConstraintViolation(err error) bool {
	if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
		return true
	}
	return false
}
