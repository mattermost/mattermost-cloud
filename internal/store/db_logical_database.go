// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"database/sql"
	"fmt"

	"github.com/mattermost/mattermost-cloud/model"

	sq "github.com/Masterminds/squirrel"
	"github.com/pkg/errors"
)

var logicalDatabaseSelect sq.SelectBuilder

func init() {
	logicalDatabaseSelect = sq.
		Select(
			"ID",
			"MultitenantDatabaseID",
			"Name",
			"CreateAt",
			"DeleteAt",
			"LockAcquiredBy",
			"LockAcquiredAt").
		From("LogicalDatabase")
}

// GetLogicalDatabase fetches the given logical database by id.
func (sqlStore *SQLStore) GetLogicalDatabase(id string) (*model.LogicalDatabase, error) {
	var logicalDatabase model.LogicalDatabase
	err := sqlStore.getBuilder(sqlStore.db, &logicalDatabase, logicalDatabaseSelect.Where("ID = ?", id))
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get logical database by id")
	}

	return &logicalDatabase, nil
}

// GetLogicalDatabases fetches the given page of created logical databases.
// The first page is 0.
func (sqlStore *SQLStore) GetLogicalDatabases(filter *model.LogicalDatabaseFilter) ([]*model.LogicalDatabase, error) {
	builder := logicalDatabaseSelect.OrderBy("CreateAt ASC")
	builder = applyPagingFilter(builder, filter.Paging)

	if len(filter.MultitenantDatabaseID) > 0 {
		builder = builder.Where(sq.Eq{"MultitenantDatabaseID": filter.MultitenantDatabaseID})
	}

	var logicalDatabases []*model.LogicalDatabase
	err := sqlStore.selectBuilder(sqlStore.db, &logicalDatabases, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query logical databases")
	}

	return logicalDatabases, nil
}

// CreateLogicalDatabase records the supplied logical database to the datastore.
func (sqlStore *SQLStore) CreateLogicalDatabase(logicalDatabase *model.LogicalDatabase) error {
	multitenantDatabase, err := sqlStore.GetMultitenantDatabase(logicalDatabase.MultitenantDatabaseID)
	if err != nil {
		return errors.Wrap(err, "failed to get multitenant database")
	}
	if multitenantDatabase == nil {
		return errors.Errorf("multitenant database %s does not exist", logicalDatabase.MultitenantDatabaseID)
	}

	logicalDatabase.ID = model.NewID()
	logicalDatabase.CreateAt = model.GetMillis()
	if len(logicalDatabase.Name) == 0 {
		logicalDatabase.Name = fmt.Sprintf("cloud_%s", logicalDatabase.ID)
	}

	_, err = sqlStore.execBuilder(sqlStore.db, sq.
		Insert("LogicalDatabase").
		SetMap(map[string]interface{}{
			"ID":                    logicalDatabase.ID,
			"MultitenantDatabaseID": logicalDatabase.MultitenantDatabaseID,
			"Name":                  logicalDatabase.Name,
			"LockAcquiredBy":        nil,
			"LockAcquiredAt":        0,
			"CreateAt":              logicalDatabase.CreateAt,
			"DeleteAt":              0,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to store logical database")
	}

	return nil
}

// DeleteLogicalDatabase marks the given logical database as deleted, but does
// not remove the record from the database.
func (sqlStore *SQLStore) DeleteLogicalDatabase(id string) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("LogicalDatabase").
		Set("DeleteAt", model.GetMillis()).
		Where("ID = ?", id).
		Where("DeleteAt = 0"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to mark logical database as deleted")
	}

	return nil
}

// LockLogicalDatabase marks the logical database as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockLogicalDatabase(logicalDatabaseID, lockerID string) (bool, error) {
	return sqlStore.lockRows("LogicalDatabase", []string{logicalDatabaseID}, lockerID)
}

// UnlockLogicalDatabase releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockLogicalDatabase(logicalDatabaseID, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows("LogicalDatabase", []string{logicalDatabaseID}, lockerID, force)
}
