package store

import (
	"github.com/pkg/errors"
)

// tableExists determines if the given table name exists in the database.
func (sqlStore *SQLStore) tableExists(tableName string) (bool, error) {
	var tableExists bool

	switch sqlStore.db.DriverName() {
	case "sqlite3":
		err := sqlStore.get(sqlStore.db, &tableExists,
			"SELECT COUNT(*) == 1 FROM sqlite_master WHERE type='table' AND name='System'",
		)
		if err != nil {
			return false, errors.Wrapf(err, "failed to check if %s table exists", tableName)
		}

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
