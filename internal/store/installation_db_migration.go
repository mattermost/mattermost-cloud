// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"database/sql"
	"encoding/json"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

const (
	installationDBMigrationTable = "InstallationDBMigrationOperation"
)

var installationDBMigrationSelect sq.SelectBuilder

func init() {
	installationDBMigrationSelect = sq.
		Select("ID",
			"InstallationID",
			"RequestAt",
			"State",
			"SourceDatabase",
			"DestinationDatabase",
			"SourceMultiTenantRaw",
			"DestinationMultiTenantRaw",
			"BackupID",
			"InstallationDBRestorationOperationID",
			"CompleteAt",
			"DeleteAt",
			"LockAcquiredBy",
			"LockAcquiredAt",
		).
		From(installationDBMigrationTable)
}

type rawDBMigrationOperation struct {
	*model.InstallationDBMigrationOperation
	SourceMultiTenantRaw      []byte
	DestinationMultiTenantRaw []byte
}

type rawDBMigrationOperations []*rawDBMigrationOperation

func (r *rawDBMigrationOperation) toDBMigrationOperation() (*model.InstallationDBMigrationOperation, error) {
	// We only need to set values that are converted from a raw database format.
	var err error
	if len(r.SourceMultiTenantRaw) > 0 {
		data := model.MultiTenantDBMigrationData{}
		err = json.Unmarshal(r.SourceMultiTenantRaw, &data)
		if err != nil {
			return nil, err
		}
		r.InstallationDBMigrationOperation.SourceMultiTenant = &data
	}
	if len(r.DestinationMultiTenantRaw) > 0 {
		data := model.MultiTenantDBMigrationData{}
		err = json.Unmarshal(r.DestinationMultiTenantRaw, &data)
		if err != nil {
			return nil, err
		}
		r.InstallationDBMigrationOperation.DestinationMultiTenant = &data
	}

	return r.InstallationDBMigrationOperation, nil
}

func (r *rawDBMigrationOperations) toDBMigrationOperations() ([]*model.InstallationDBMigrationOperation, error) {
	if r == nil {
		return []*model.InstallationDBMigrationOperation{}, nil
	}
	migrationOperations := make([]*model.InstallationDBMigrationOperation, 0, len(*r))

	for _, raw := range *r {
		operation, err := raw.toDBMigrationOperation()
		if err != nil {
			return nil, errors.Wrap(err, "failed to create migration operation from raw")
		}
		migrationOperations = append(migrationOperations, operation)
	}
	return migrationOperations, nil
}

// TODO: we should probably create some intermediary layer to not keep this logic in store.
// For now tho transactions are not accessible outside the store, therefore it is implemented this way.

// TriggerInstallationDBMigration creates new InstallationDBMigrationOperation in Requested state
// and changes installation state to InstallationStateDBMigrationInProgress.
func (sqlStore *SQLStore) TriggerInstallationDBMigration(dbMigrationOp *model.InstallationDBMigrationOperation, installation *model.Installation) (*model.InstallationDBMigrationOperation, error) {
	dbMigrationOp.InstallationID = installation.ID
	dbMigrationOp.State = model.InstallationDBMigrationStateRequested
	dbMigrationOp.InstallationDBRestorationOperationID = ""

	tx, err := sqlStore.beginTransaction(sqlStore.db)
	if err != nil {
		return nil, errors.Wrap(err, "failed to start transaction")
	}
	defer tx.RollbackUnlessCommitted()

	err = sqlStore.createInstallationDBMigration(tx, dbMigrationOp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create installation db migration")
	}

	installation.State = model.InstallationStateDBMigrationInProgress
	err = sqlStore.updateInstallation(tx, installation)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update installation")
	}

	err = tx.Commit()
	if err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}

	return dbMigrationOp, nil
}

// CreateInstallationDBMigrationOperation records installation db migration to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateInstallationDBMigrationOperation(dbMigration *model.InstallationDBMigrationOperation) error {
	return sqlStore.createInstallationDBMigration(sqlStore.db, dbMigration)
}

// createInstallationDBMigration records installation db migration to the database, assigning it a unique ID.
func (sqlStore *SQLStore) createInstallationDBMigration(db execer, dbMigration *model.InstallationDBMigrationOperation) error {
	dbMigration.ID = model.NewID()
	dbMigration.RequestAt = GetMillis()

	insertMap := map[string]interface{}{
		"ID":                                   dbMigration.ID,
		"InstallationID":                       dbMigration.InstallationID,
		"RequestAt":                            dbMigration.RequestAt,
		"State":                                dbMigration.State,
		"SourceDatabase":                       dbMigration.SourceDatabase,
		"DestinationDatabase":                  dbMigration.DestinationDatabase,
		"BackupID":                             dbMigration.BackupID,
		"InstallationDBRestorationOperationID": dbMigration.InstallationDBRestorationOperationID,
		"CompleteAt":                           dbMigration.CompleteAt,
		"DeleteAt":                             0,
		"LockAcquiredBy":                       dbMigration.LockAcquiredBy,
		"LockAcquiredAt":                       dbMigration.LockAcquiredAt,
	}

	if dbMigration.SourceMultiTenant != nil {
		multiTenantSourceRaw, err := json.Marshal(dbMigration.SourceMultiTenant)
		if err != nil {
			return errors.Wrap(err, "failed to marshal source multi tenant db")
		}
		insertMap["SourceMultiTenantRaw"] = multiTenantSourceRaw
	}
	if dbMigration.DestinationMultiTenant != nil {
		multiTenantDestinationRaw, err := json.Marshal(dbMigration.DestinationMultiTenant)
		if err != nil {
			return errors.Wrap(err, "failed to marshal destination multi tenant db")
		}
		insertMap["DestinationMultiTenantRaw"] = multiTenantDestinationRaw
	}

	_, err := sqlStore.execBuilder(db, sq.
		Insert(installationDBMigrationTable).
		SetMap(insertMap),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create installation db migration operation")
	}

	return nil
}

// GetInstallationDBMigrationOperation fetches the given installation db migration.
func (sqlStore *SQLStore) GetInstallationDBMigrationOperation(id string) (*model.InstallationDBMigrationOperation, error) {
	builder := installationDBMigrationSelect.
		Where("ID = ?", id)

	var migrationOpRaw rawDBMigrationOperation
	err := sqlStore.getBuilder(sqlStore.db, &migrationOpRaw, builder)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to query for installation db migration")
	}

	migrationOp, err := migrationOpRaw.toDBMigrationOperation()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create migration operation from raw")
	}

	return migrationOp, nil
}

// GetInstallationDBMigrationOperations fetches the given page of created installation db migration. The first page is 0.
func (sqlStore *SQLStore) GetInstallationDBMigrationOperations(filter *model.InstallationDBMigrationFilter) ([]*model.InstallationDBMigrationOperation, error) {
	builder := installationDBMigrationSelect.
		OrderBy("RequestAt DESC")
	builder = sqlStore.applyInstallationDBMigrationFilter(builder, filter)

	return sqlStore.getInstallationDBMigrationOperations(builder)
}

// GetUnlockedInstallationDBMigrationOperationsPendingWork returns unlocked installation db migrations in a pending state.
func (sqlStore *SQLStore) GetUnlockedInstallationDBMigrationOperationsPendingWork() ([]*model.InstallationDBMigrationOperation, error) {
	builder := installationDBMigrationSelect.
		Where(sq.Eq{
			"State": model.AllInstallationDBMigrationOperationsStatesPendingWork,
		}).
		Where("LockAcquiredAt = 0").
		OrderBy("RequestAt ASC")

	return sqlStore.getInstallationDBMigrationOperations(builder)
}

func (sqlStore *SQLStore) getInstallationDBMigrationOperations(builder builder) ([]*model.InstallationDBMigrationOperation, error) {
	var rawMigrationOps rawDBMigrationOperations
	err := sqlStore.selectBuilder(sqlStore.db, &rawMigrationOps, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for installation db migrations")
	}

	migrationOps, err := rawMigrationOps.toDBMigrationOperations()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create migration operations from raw")
	}

	return migrationOps, nil
}

// UpdateInstallationDBMigrationOperationState updates the given installation db migration state.
func (sqlStore *SQLStore) UpdateInstallationDBMigrationOperationState(dbMigration *model.InstallationDBMigrationOperation) error {
	return sqlStore.updateInstallationDBMigrationFields(
		sqlStore.db,
		dbMigration.ID, map[string]interface{}{
			"State": dbMigration.State,
		})
}

// UpdateInstallationDBMigrationOperation updates the given installation db migration.
func (sqlStore *SQLStore) UpdateInstallationDBMigrationOperation(dbMigration *model.InstallationDBMigrationOperation) error {
	return sqlStore.updateInstallationDBMigration(sqlStore.db, dbMigration)
}

func (sqlStore *SQLStore) updateInstallationDBMigration(db execer, dbMigration *model.InstallationDBMigrationOperation) error {
	return sqlStore.updateInstallationDBMigrationFields(
		db,
		dbMigration.ID, map[string]interface{}{
			"State":                                dbMigration.State,
			"BackupID":                             dbMigration.BackupID,
			"InstallationDBRestorationOperationID": dbMigration.InstallationDBRestorationOperationID,
			"CompleteAt":                           dbMigration.CompleteAt,
		})
}

func (sqlStore *SQLStore) updateInstallationDBMigrationFields(db execer, id string, fields map[string]interface{}) error {
	_, err := sqlStore.execBuilder(db, sq.
		Update(installationDBMigrationTable).
		SetMap(fields).
		Where("ID = ?", id))
	if err != nil {
		return errors.Wrapf(err, "failed to update installation db migration fields: %s", getMapKeys(fields))
	}

	return nil
}

// LockInstallationDBMigrationOperation marks the InstallationDBMigrationOperation as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockInstallationDBMigrationOperation(id, lockerID string) (bool, error) {
	return sqlStore.lockRows(installationDBMigrationTable, []string{id}, lockerID)
}

// LockInstallationDBMigrationOperations marks InstallationDBMigrationOperation as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockInstallationDBMigrationOperations(ids []string, lockerID string) (bool, error) {
	return sqlStore.lockRows(installationDBMigrationTable, ids, lockerID)
}

// UnlockInstallationDBMigrationOperation releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockInstallationDBMigrationOperation(id, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows(installationDBMigrationTable, []string{id}, lockerID, force)
}

// UnlockInstallationDBMigrationOperations releases a locks previously acquired against a caller.
func (sqlStore *SQLStore) UnlockInstallationDBMigrationOperations(ids []string, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows(installationDBMigrationTable, ids, lockerID, force)
}

func (sqlStore *SQLStore) applyInstallationDBMigrationFilter(builder sq.SelectBuilder, filter *model.InstallationDBMigrationFilter) sq.SelectBuilder {
	builder = applyPagingFilter(builder, filter.Paging)

	if len(filter.IDs) > 0 {
		builder = builder.Where(sq.Eq{"ID": filter.IDs})
	}
	if filter.InstallationID != "" {
		builder = builder.Where("InstallationID = ?", filter.InstallationID)
	}
	if len(filter.States) > 0 {
		builder = builder.Where(sq.Eq{
			"State": filter.States,
		})
	}

	return builder
}
