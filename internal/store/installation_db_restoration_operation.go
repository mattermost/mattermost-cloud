// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

const (
	installationDBRestorationTable = "InstallationDBRestorationOperation"
)

var installationDBRestorationSelect sq.SelectBuilder

func init() {
	installationDBRestorationSelect = sq.
		Select("ID",
			"InstallationID",
			"BackupID",
			"RequestAt",
			"State",
			"TargetInstallationState",
			"ClusterInstallationID",
			"CompleteAt",
			"DeleteAt",
			"LockAcquiredBy",
			"LockAcquiredAt",
		).
		From(installationDBRestorationTable)
}

// TODO: we should probably create some intermediary layer to not keep this logic in store.
// For now tho transactions are not accessible outside the store, therefore it is implemented this way.

// TriggerInstallationRestoration creates new InstallationDBRestorationOperation in Requested state
// and changes installation state to InstallationStateDBRestorationInProgress.
func (sqlStore *SQLStore) TriggerInstallationRestoration(installation *model.Installation, backup *model.InstallationBackup) (*model.InstallationDBRestorationOperation, error) {
	targetInstallationState, err := model.DetermineAfterRestorationState(installation)
	if err != nil {
		return nil, errors.Wrap(err, "failed to determine target installation state")
	}

	dbRestorationOp := &model.InstallationDBRestorationOperation{
		InstallationID:          installation.ID,
		BackupID:                backup.ID,
		State:                   model.InstallationDBRestorationStateRequested,
		TargetInstallationState: targetInstallationState,
	}

	tx, err := sqlStore.beginTransaction(sqlStore.db)
	if err != nil {
		return nil, errors.Wrap(err, "failed to start transaction")
	}
	defer tx.RollbackUnlessCommitted()

	err = sqlStore.createInstallationDBRestoration(tx, dbRestorationOp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create installation db restoration")
	}

	installation.State = model.InstallationStateDBRestorationInProgress
	err = sqlStore.updateInstallation(tx, installation)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update installation")
	}

	err = tx.Commit()
	if err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}

	return dbRestorationOp, nil
}

// CreateInstallationDBRestorationOperation records installation db restoration to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateInstallationDBRestorationOperation(dbRestoration *model.InstallationDBRestorationOperation) error {
	return sqlStore.createInstallationDBRestoration(sqlStore.db, dbRestoration)
}

func (sqlStore *SQLStore) createInstallationDBRestoration(db execer, dbRestoration *model.InstallationDBRestorationOperation) error {
	dbRestoration.ID = model.NewID()
	dbRestoration.RequestAt = model.GetMillis()

	_, err := sqlStore.execBuilder(db, sq.
		Insert(installationDBRestorationTable).
		SetMap(map[string]interface{}{
			"ID":                      dbRestoration.ID,
			"InstallationID":          dbRestoration.InstallationID,
			"BackupID":                dbRestoration.BackupID,
			"State":                   dbRestoration.State,
			"RequestAt":               dbRestoration.RequestAt,
			"TargetInstallationState": dbRestoration.TargetInstallationState,
			"ClusterInstallationID":   dbRestoration.ClusterInstallationID,
			"CompleteAt":              dbRestoration.CompleteAt,
			"DeleteAt":                0,
			"LockAcquiredBy":          dbRestoration.LockAcquiredBy,
			"LockAcquiredAt":          dbRestoration.LockAcquiredAt,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create installation db restoration operation")
	}

	return nil
}

// GetInstallationDBRestorationOperation fetches the given installation db restoration.
func (sqlStore *SQLStore) GetInstallationDBRestorationOperation(id string) (*model.InstallationDBRestorationOperation, error) {
	builder := installationDBRestorationSelect.
		Where("ID = ?", id)

	var restorationOp model.InstallationDBRestorationOperation
	err := sqlStore.getBuilder(sqlStore.db, &restorationOp, builder)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to query for installation db restoration")
	}

	return &restorationOp, nil
}

// GetInstallationDBRestorationOperations fetches the given page of created installation db restoration. The first page is 0.
func (sqlStore *SQLStore) GetInstallationDBRestorationOperations(filter *model.InstallationDBRestorationFilter) ([]*model.InstallationDBRestorationOperation, error) {
	builder := installationDBRestorationSelect.
		OrderBy("RequestAt DESC")
	builder = sqlStore.applyInstallationDBRestorationFilter(builder, filter)

	var restorationOps []*model.InstallationDBRestorationOperation
	err := sqlStore.selectBuilder(sqlStore.db, &restorationOps, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for installation db restorations")
	}

	return restorationOps, nil
}

// GetUnlockedInstallationDBRestorationOperationsPendingWork returns unlocked installation db restorations in a pending state.
func (sqlStore *SQLStore) GetUnlockedInstallationDBRestorationOperationsPendingWork() ([]*model.InstallationDBRestorationOperation, error) {
	builder := installationDBRestorationSelect.
		Where(sq.Eq{
			"State": model.AllInstallationDBRestorationStatesPendingWork,
		}).
		Where("LockAcquiredAt = 0").
		OrderBy("RequestAt ASC")

	var restorationOps []*model.InstallationDBRestorationOperation
	err := sqlStore.selectBuilder(sqlStore.db, &restorationOps, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for installation db restorations")
	}

	return restorationOps, nil
}

// UpdateInstallationDBRestorationOperationState updates the given installation db restoration state.
func (sqlStore *SQLStore) UpdateInstallationDBRestorationOperationState(dbRestoration *model.InstallationDBRestorationOperation) error {
	return sqlStore.updateInstallationDBRestorationFields(
		sqlStore.db,
		dbRestoration.ID, map[string]interface{}{
			"State": dbRestoration.State,
		})
}

// UpdateInstallationDBRestorationOperation updates the given installation db restoration.
func (sqlStore *SQLStore) UpdateInstallationDBRestorationOperation(dbRestoration *model.InstallationDBRestorationOperation) error {
	return sqlStore.updateInstallationDBRestoration(sqlStore.db, dbRestoration)
}

func (sqlStore *SQLStore) updateInstallationDBRestoration(db execer, dbRestoration *model.InstallationDBRestorationOperation) error {
	return sqlStore.updateInstallationDBRestorationFields(
		db,
		dbRestoration.ID, map[string]interface{}{
			"State":                   dbRestoration.State,
			"TargetInstallationState": dbRestoration.TargetInstallationState,
			"ClusterInstallationID":   dbRestoration.ClusterInstallationID,
			"CompleteAt":              dbRestoration.CompleteAt,
		})
}

func (sqlStore *SQLStore) updateInstallationDBRestorationFields(db execer, id string, fields map[string]interface{}) error {
	_, err := sqlStore.execBuilder(db, sq.
		Update(installationDBRestorationTable).
		SetMap(fields).
		Where("ID = ?", id))
	if err != nil {
		return errors.Wrapf(err, "failed to update installation db restoration fields: %s", getMapKeys(fields))
	}

	return nil
}

// DeleteInstallationDBRestorationOperation marks the given restoration operation as deleted,
// but does not remove the record from the database.
func (sqlStore *SQLStore) DeleteInstallationDBRestorationOperation(id string) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update(installationDBRestorationTable).
		Set("DeleteAt", model.GetMillis()).
		Where("ID = ?", id).
		Where("DeleteAt = ?", 0))
	if err != nil {
		return errors.Wrapf(err, "failed to to mark restoration as deleted")
	}

	return nil
}

// LockInstallationDBRestorationOperation marks the InstallationDBRestoration as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockInstallationDBRestorationOperation(id, lockerID string) (bool, error) {
	return sqlStore.lockRows(installationDBRestorationTable, []string{id}, lockerID)
}

// LockInstallationDBRestorationOperations marks InstallationDBRestorations as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockInstallationDBRestorationOperations(ids []string, lockerID string) (bool, error) {
	return sqlStore.lockRows(installationDBRestorationTable, ids, lockerID)
}

// UnlockInstallationDBRestorationOperation releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockInstallationDBRestorationOperation(id, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows(installationDBRestorationTable, []string{id}, lockerID, force)
}

// UnlockInstallationDBRestorationOperations releases a locks previously acquired against a caller.
func (sqlStore *SQLStore) UnlockInstallationDBRestorationOperations(ids []string, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows(installationDBRestorationTable, ids, lockerID, force)
}

func (sqlStore *SQLStore) applyInstallationDBRestorationFilter(builder sq.SelectBuilder, filter *model.InstallationDBRestorationFilter) sq.SelectBuilder {
	builder = applyPagingFilter(builder, filter.Paging)

	if len(filter.IDs) > 0 {
		builder = builder.Where(sq.Eq{"ID": filter.IDs})
	}
	if filter.InstallationID != "" {
		builder = builder.Where("InstallationID = ?", filter.InstallationID)
	}
	if filter.ClusterInstallationID != "" {
		builder = builder.Where("ClusterInstallationID = ?", filter.ClusterInstallationID)
	}
	if len(filter.States) > 0 {
		builder = builder.Where(sq.Eq{
			"State": filter.States,
		})
	}

	return builder
}
