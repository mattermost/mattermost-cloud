package store

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/pkg/errors"
)

// lockRow marks the row in the given table as locked for exclusive use by the caller.
func (sqlStore *SQLStore) lockRow(table, id, lockerID string) (bool, error) {
	result, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update(table).
		SetMap(map[string]interface{}{
			"LockAcquiredBy": lockerID,
			"LockAcquiredAt": GetMillis(),
		}).
		Where(sq.Eq{
			"ID":             id,
			"LockAcquiredAt": 0,
		}),
	)
	if err != nil {
		return false, errors.Wrapf(err, "failed to lock row %s in %s", id, table)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, errors.Wrap(err, "failed to count rows affected")
	}

	locked := false
	if count > 0 {
		locked = true
	}

	return locked, nil
}

// unlockRow releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) unlockRow(table, id, lockerID string, force bool) (bool, error) {
	builder := sq.Update(table).
		SetMap(map[string]interface{}{
			"LockAcquiredBy": nil,
			"LockAcquiredAt": 0,
		}).
		Where(sq.Eq{
			"ID": id,
		})

	if force {
		// If forcing the unlock, only require that a lock was held by someone.
		builder = builder.Where("LockAcquiredAt <> 0")
	} else {
		// If not forcing the unlock, require that the current instance held the lock.
		builder = builder.Where(sq.Eq{
			"LockAcquiredBy": lockerID,
		})
	}

	result, err := sqlStore.execBuilder(sqlStore.db, builder)
	if err != nil {
		return false, errors.Wrapf(err, "failed to unlock row %s in %s", id, table)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return false, errors.Wrap(err, "failed to count rows affected")
	}

	unlocked := false
	if count > 0 {
		unlocked = true
	}

	return unlocked, nil
}
