package store

import (
	"database/sql"

	"github.com/pkg/errors"
)

// getSystemValue queries the System table for the given key
func (sqlStore *SQLStore) getSystemValue(q queryer, key string) (string, error) {
	var value string
	err := sqlStore.get(q, &value, "SELECT Value FROM System WHERE Key = ?", key)
	if err == sql.ErrNoRows {
		return "", nil
	} else if err != nil {
		return "", errors.Wrapf(err, "failed to query system key %s", key)
	}

	return value, nil
}

// setSystemValue updates the System table for the given key.
func (sqlStore *SQLStore) setSystemValue(e execer, key, value string) error {
	result, err := sqlStore.exec(e, "UPDATE System SET Value = ? WHERE Key = ?", value, key)
	if err != nil {
		return errors.Wrapf(err, "failed to update system key %s", key)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		return nil
	}

	result, err = sqlStore.exec(e, "INSERT INTO System (Key, Value) VALUES (?, ?)", key, value)
	if err != nil {
		return errors.Wrapf(err, "failed to insert system key %s", key)
	}

	return nil
}
