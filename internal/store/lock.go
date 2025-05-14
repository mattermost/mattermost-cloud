// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

const (
	standardLockByName = "LockAcquiredBy"
	standardLockAtName = "LockAcquiredAt"
)

// lockRow marks the row in the given table as locked for exclusive use by the caller.
func (sqlStore *SQLStore) lockRows(table string, ids []string, lockerID string) (bool, error) {
	return sqlStore.lockRowsTx(sqlStore.db, table, standardLockByName, standardLockAtName, ids, lockerID)
}

// lockCustomRows marks the custom lock rows in the given table as locked for
// exclusive use by the caller.
func (sqlStore *SQLStore) lockCustomRows(table, lockByName, lockAtName string, ids []string, lockerID string) (bool, error) {
	return sqlStore.lockRowsTx(sqlStore.db, table, lockByName, lockAtName, ids, lockerID)
}

// lockRowsTx performs the resource locking transaction.
func (sqlStore *SQLStore) lockRowsTx(db execer, table, lockByName, lockAtName string, ids []string, lockerID string) (bool, error) {
	result, err := sqlStore.execBuilder(db, sq.
		Update(table).
		SetMap(map[string]interface{}{
			lockByName: lockerID,
			lockAtName: model.GetMillis(),
		}).
		Where(sq.Eq{
			"ID":       ids,
			lockAtName: 0,
		}),
	)
	if err != nil {
		return false, errors.Wrapf(err, "failed to lock %d rows in %s", len(ids), table)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, errors.Wrap(err, "failed to count rows affected")
	}

	locked := count > 0

	if locked && int(count) < len(ids) {
		sqlStore.logger.Warnf("Locked only %d of %d rows in %s", count, len(ids), table)
	}

	return locked, nil
}

// unlockRows releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) unlockRows(table string, ids []string, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRowsTx(table, standardLockByName, standardLockAtName, ids, lockerID, force)
}

// unlockCustomRows releases a cusotm lock previously acquired against a caller.
func (sqlStore *SQLStore) unlockCustomRows(table, lockByName, lockAtName string, ids []string, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRowsTx(table, lockByName, lockAtName, ids, lockerID, force)
}

// unlockRowsTx performs the resource unlocking transaction.
func (sqlStore *SQLStore) unlockRowsTx(table, lockByName, lockAtName string, ids []string, lockerID string, force bool) (bool, error) {
	builder := sq.Update(table).
		SetMap(map[string]interface{}{
			lockByName: nil,
			lockAtName: 0,
		}).
		Where(sq.Eq{
			"ID": ids,
		})

	if force {
		// If forcing the unlock, only require that a lock was held by someone.
		builder = builder.Where(fmt.Sprintf("%s <> 0", lockAtName))
	} else {
		// If not forcing the unlock, require that the current instance held the lock.
		builder = builder.Where(sq.Eq{
			lockByName: lockerID,
		})
	}

	result, err := sqlStore.execBuilder(sqlStore.db, builder)
	if err != nil {
		return false, errors.Wrapf(err, "failed to unlock %d rows in %s", len(ids), table)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return false, errors.Wrap(err, "failed to count rows affected")
	}

	unlocked := count > 0

	if int(count) < len(ids) {
		sqlStore.logger.Warnf("Unlocked only %d of %d rows in %s", count, len(ids), table)
	}

	return unlocked, nil
}
