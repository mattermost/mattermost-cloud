// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/pkg/errors"
)

// getSystemValue queries the System table for the given key
func (sqlStore *SQLStore) getSystemValue(q queryer, key string) (string, error) {
	var value string

	err := sqlStore.getBuilder(q, &value,
		sq.Select("Value").From("System").Where("Key = ?", key),
	)
	if err == sql.ErrNoRows {
		return "", nil
	} else if err != nil {
		return "", errors.Wrapf(err, "failed to query system key %s", key)
	}

	return value, nil
}

// setSystemValue updates the System table for the given key.
func (sqlStore *SQLStore) setSystemValue(e execer, key, value string) error {
	result, err := sqlStore.execBuilder(e,
		sq.Update("System").Set("Value", value).Where("Key = ?", key),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to update system key %s", key)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		return nil
	}

	result, err = sqlStore.execBuilder(e,
		sq.Insert("System").Columns("Key", "Value").Values(key, value),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to insert system key %s", key)
	}

	return nil
}
