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

var databaseSchemaSelect sq.SelectBuilder

func init() {
	databaseSchemaSelect = sq.
		Select(
			"ID",
			"LogicalDatabaseID",
			"InstallationID",
			"Name",
			"CreateAt",
			"DeleteAt",
			"LockAcquiredBy",
			"LockAcquiredAt").
		From("DatabaseSchema")
}

// GetDatabaseSchema fetches the given database schema by id.
func (sqlStore *SQLStore) GetDatabaseSchema(id string) (*model.DatabaseSchema, error) {
	var databaseSchema model.DatabaseSchema
	err := sqlStore.getBuilder(sqlStore.db, &databaseSchema, databaseSchemaSelect.Where("ID = ?", id))
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get database schema by id")
	}

	return &databaseSchema, nil
}

// GetDatabaseSchemas fetches the given page of created database schemas. The
// first page is 0.
func (sqlStore *SQLStore) GetDatabaseSchemas(filter *model.DatabaseSchemaFilter) ([]*model.DatabaseSchema, error) {
	builder := databaseSchemaSelect.OrderBy("CreateAt ASC")
	builder = applyPagingFilter(builder, filter.Paging)

	if len(filter.LogicalDatabaseID) > 0 {
		builder = builder.Where(sq.Eq{"LogicalDatabaseID": filter.LogicalDatabaseID})
	}
	if len(filter.InstallationID) > 0 {
		builder = builder.Where(sq.Eq{"InstallationID": filter.InstallationID})
	}

	var databaseSchemas []*model.DatabaseSchema
	err := sqlStore.selectBuilder(sqlStore.db, &databaseSchemas, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query database schemas")
	}

	return databaseSchemas, nil
}

// GetDatabaseSchemaForInstallationID fetches the given database schema by
// installation id.
func (sqlStore *SQLStore) GetDatabaseSchemaForInstallationID(installationID string) (*model.DatabaseSchema, error) {
	databaseSchemas, err := sqlStore.GetDatabaseSchemas(&model.DatabaseSchemaFilter{
		InstallationID: installationID,
		Paging:         model.AllPagesNotDeleted(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database schemas")
	}
	if len(databaseSchemas) == 0 {
		return nil, nil
	}
	if len(databaseSchemas) == 1 {
		return databaseSchemas[0], nil
	}

	return nil, errors.Errorf("expected no more than one database schema, but got %d", len(databaseSchemas))
}

// CreateDatabaseSchema records the supplied database schema to the datastore.
func (sqlStore *SQLStore) CreateDatabaseSchema(databaseSchema *model.DatabaseSchema) error {
	return sqlStore.createDatabaseSchema(sqlStore.db, databaseSchema)
}

// createDatabaseSchema records the supplied database schema to the datastore.
func (sqlStore *SQLStore) createDatabaseSchema(db execer, databaseSchema *model.DatabaseSchema) error {
	databaseSchema.ID = model.NewID()
	databaseSchema.CreateAt = model.GetMillis()
	if len(databaseSchema.Name) == 0 {
		databaseSchema.Name = fmt.Sprintf("id_%s", databaseSchema.InstallationID)
	}

	_, err := sqlStore.execBuilder(db, sq.
		Insert("DatabaseSchema").
		SetMap(map[string]interface{}{
			"ID":                databaseSchema.ID,
			"LogicalDatabaseID": databaseSchema.LogicalDatabaseID,
			"InstallationID":    databaseSchema.InstallationID,
			"Name":              databaseSchema.Name,
			"LockAcquiredBy":    nil,
			"LockAcquiredAt":    0,
			"CreateAt":          databaseSchema.CreateAt,
			"DeleteAt":          0,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to store database schema")
	}

	return nil
}

// deleteDatabaseSchema marks the given database schema as deleted, but does not
// remove the record from the database.
func (sqlStore *SQLStore) deleteDatabaseSchema(db execer, id string) error {
	_, err := sqlStore.execBuilder(db, sq.
		Update("DatabaseSchema").
		Set("DeleteAt", model.GetMillis()).
		Where("ID = ?", id).
		Where("DeleteAt = 0"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to mark database schema as deleted")
	}

	return nil
}

// LockDatabaseSchema marks the database schema as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockDatabaseSchema(databaseSchemaID, lockerID string) (bool, error) {
	return sqlStore.lockRows("DatabaseSchema", []string{databaseSchemaID}, lockerID)
}

// UnlockDatabaseSchema releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockDatabaseSchema(databaseSchemaID, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows("DatabaseSchema", []string{databaseSchemaID}, lockerID, force)
}
