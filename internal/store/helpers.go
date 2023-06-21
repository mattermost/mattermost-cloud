// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"github.com/pkg/errors"
)

// tableExists determines if the given table name exists in the database.
func (sqlStore *SQLStore) tableExists(tableName string) (bool, error) {
	var tableExists bool

	switch sqlStore.db.DriverName() {
	case "postgres":
		err := sqlStore.get(sqlStore.db, &tableExists,
			"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = 'system')",
		)
		if err != nil {
			return false, errors.Wrapf(err, "failed to check if %s table exists", tableName)
		}

	default:
		return false, errors.Errorf("unsupported driver %s", sqlStore.db.DriverName())
	}

	return tableExists, nil
}

// countResult handles differences in how count queries can be returned.
type countResult []struct {
	Count         int64 `db:"count"`
	CountWildcard int64 `db:"Count (*)"`
}

// value checks the countResult and returns the correct count value.
func (c countResult) value() (int64, error) {
	if len(c) == 0 {
		return 0, errors.New("no count result returned")
	}
	if c[0].Count != 0 {
		return c[0].Count, nil
	}
	if c[0].CountWildcard != 0 {
		return c[0].CountWildcard, nil
	}

	return 0, nil
}
