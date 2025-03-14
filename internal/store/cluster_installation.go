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

var clusterInstallationSelect sq.SelectBuilder

func init() {
	clusterInstallationSelect = sq.
		Select(
			"ID", "ClusterID", "InstallationID", "Namespace", "State", "CreateAt",
			"DeleteAt", "APISecurityLock", "LockAcquiredBy", "LockAcquiredAt", "IsActive",
		).
		From("ClusterInstallation")
}

// GetClusterInstallation fetches the given cluster installation by id.
func (sqlStore *SQLStore) GetClusterInstallation(id string) (*model.ClusterInstallation, error) {
	var clusterInstallation model.ClusterInstallation
	err := sqlStore.getBuilder(sqlStore.db, &clusterInstallation,
		clusterInstallationSelect.Where("ID = ?", id),
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster installation by id")
	}

	return &clusterInstallation, nil
}

// GetUnlockedClusterInstallationsPendingWork returns an unlocked cluster installation in a pending state.
func (sqlStore *SQLStore) GetUnlockedClusterInstallationsPendingWork() ([]*model.ClusterInstallation, error) {

	builder := clusterInstallationSelect.
		Where(sq.Eq{
			"State": model.AllClusterInstallationStatesPendingWork,
		}).
		Where("LockAcquiredAt = 0").
		OrderBy("CreateAt ASC")

	var clusterInstallations []*model.ClusterInstallation
	err := sqlStore.selectBuilder(sqlStore.db, &clusterInstallations, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster installations pending work")
	}

	return clusterInstallations, nil
}

// CreateClusterInstallation records the given cluster installation to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateClusterInstallation(clusterInstallation *model.ClusterInstallation) error {
	return sqlStore.createClusterInstallation(sqlStore.db, clusterInstallation)
}

// GetClusterInstallations fetches the given page of created clusters. The first page is 0.
func (sqlStore *SQLStore) GetClusterInstallations(filter *model.ClusterInstallationFilter) ([]*model.ClusterInstallation, error) {
	return sqlStore.getClusterInstallations(sqlStore.db, filter)
}

func (sqlStore *SQLStore) getClusterInstallations(db dbInterface, filter *model.ClusterInstallationFilter) ([]*model.ClusterInstallation, error) {
	builder := clusterInstallationSelect.
		OrderBy("CreateAt ASC")

	builder = applyPagingFilter(builder, filter.Paging)

	if len(filter.IDs) > 0 {
		builder = builder.Where(sq.Eq{"ID": filter.IDs})
	}
	if filter.ClusterID != "" {
		builder = builder.Where("ClusterID = ?", filter.ClusterID)
	}
	if filter.InstallationID != "" {
		builder = builder.Where("InstallationID = ?", filter.InstallationID)
	}
	if filter.IsActive != nil {
		builder = builder.Where("IsActive = ?", *filter.IsActive)
	}
	var clusterInstallations []*model.ClusterInstallation
	err := sqlStore.selectBuilder(db, &clusterInstallations, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for clusterInstallations")
	}

	return clusterInstallations, nil
}

func (sqlStore *SQLStore) GetClusterInstallationsForInstallation(installationID string) ([]*model.ClusterInstallation, error) {
	return sqlStore.getClusterInstallationsForInstallations(sqlStore.db, installationID)
}

func (sqlStore *SQLStore) GetClusterInstallationsForInstallations(installationIDs []string) ([]*model.ClusterInstallation, error) {
	return sqlStore.getClusterInstallationsForInstallations(sqlStore.db, installationIDs...)
}

func (sqlStore *SQLStore) getClusterInstallationsForInstallations(db queryer, installationIDs ...string) ([]*model.ClusterInstallation, error) {
	builder := clusterInstallationSelect.
		Where(sq.Eq{"InstallationID": installationIDs}).
		Where("DeleteAt = 0").
		OrderBy("CreateAt ASC")

	var clusterInstallations []*model.ClusterInstallation
	err := sqlStore.selectBuilder(db, &clusterInstallations, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for clusterInstallations")
	}

	return clusterInstallations, nil
}

// UpdateClusterInstallation updates the given cluster installation in the database.
func (sqlStore *SQLStore) UpdateClusterInstallation(clusterInstallation *model.ClusterInstallation) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("ClusterInstallation").
		SetMap(map[string]interface{}{
			"ClusterID":      clusterInstallation.ClusterID,
			"InstallationID": clusterInstallation.InstallationID,
			"Namespace":      clusterInstallation.Namespace,
			"State":          clusterInstallation.State,
		}).
		Where("ID = ?", clusterInstallation.ID),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update cluster installation")
	}

	return nil
}

// UpdateClusterInstallationsActiveStatus updates the stale status of all cluster installations for a given cluster.
func (sqlStore *SQLStore) UpdateClusterInstallationsActiveStatus(db execer, clusterInstallationIDs []string, isActive bool) error {
	_, err := sqlStore.execBuilder(db, sq.
		Update("ClusterInstallation").
		SetMap(map[string]interface{}{
			"IsActive": isActive,
		}).
		Where(sq.Eq{
			"ID": clusterInstallationIDs,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update cluster installation")
	}

	return nil
}

// DeleteClusterInstallation marks the given cluster installation as deleted, but does not remove
// the record from the database.
func (sqlStore *SQLStore) DeleteClusterInstallation(id string) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("ClusterInstallation").
		Set("DeleteAt", model.GetMillis()).
		Where("ID = ?", id).
		Where("DeleteAt = 0"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to mark cluster installation as deleted")
	}

	return nil
}

// RecoverClusterInstallation recovers a given cluster installation from the deleted state.
func (sqlStore *SQLStore) RecoverClusterInstallation(clusterInstallation *model.ClusterInstallation) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("ClusterInstallation").
		Set("State", clusterInstallation.State).
		Set("DeleteAt", 0).
		Where("ID = ?", clusterInstallation.ID).
		Where("State = ?", model.ClusterInstallationStateDeleted).
		Where("DeleteAt != 0"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update cluster installation recovery values")
	}

	return nil
}

// DeleteInActiveClusterInstallationByClusterID marks the inactive cluster installation as deleted for a given cluster, but does not remove
// the record from the database.
func (sqlStore *SQLStore) DeleteInActiveClusterInstallationByClusterID(clusterID string) (int64, error) {
	result, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("ClusterInstallation").
		Set("State", model.ClusterInstallationStateDeletionRequested).
		Where("ClusterID = ?", clusterID).
		Where("IsActive = ?", false).
		Where("DeleteAt = 0"),
	)
	if err != nil {
		return 0, errors.Wrap(err, "failed to mark inactive cluster installation as deleted")
	}
	rowsUpdated, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get total affcted rows from updating inactive cluster statement")
	}
	return rowsUpdated, nil
}

// LockClusterInstallations marks the cluster installation as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockClusterInstallations(clusterInstallationIDs []string, lockerID string) (bool, error) {
	return sqlStore.lockRows("ClusterInstallation", clusterInstallationIDs, lockerID)
}

// UnlockClusterInstallations releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockClusterInstallations(clusterInstallationIDs []string, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows("ClusterInstallation", clusterInstallationIDs, lockerID, force)
}

// LockClusterInstallationAPI locks updates to the cluster installation from the API.
func (sqlStore *SQLStore) LockClusterInstallationAPI(id string) error {
	return sqlStore.setClusterInstallationAPILock(id, true)
}

// UnlockClusterInstallationAPI unlocks updates to the cluster installation from the API.
func (sqlStore *SQLStore) UnlockClusterInstallationAPI(id string) error {
	return sqlStore.setClusterInstallationAPILock(id, false)
}

func (sqlStore *SQLStore) setClusterInstallationAPILock(id string, lock bool) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("ClusterInstallation").
		Set("APISecurityLock", lock).
		Where("ID = ?", id),
	)
	if err != nil {
		return errors.Wrap(err, "failed to store cluster installation API lock")
	}

	return nil
}

// MigrateClusterInstallations updates the given cluster installation in the database.
func (sqlStore *SQLStore) MigrateClusterInstallations(clusterInstallations []*model.ClusterInstallation, targetCluster string) error {
	tx, err := sqlStore.beginTransaction(sqlStore.db)
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer tx.RollbackUnlessCommitted()

	for _, clusterInstallation := range clusterInstallations {
		clusterInstallation.ClusterID = targetCluster
		clusterInstallation.State = model.ClusterInstallationStateCreationRequested
		clusterInstallation.IsActive = false
		err = sqlStore.createClusterInstallation(tx, clusterInstallation)

		if err != nil {
			return errors.Wrap(err, "failed to create cluster installation")
		}
	}
	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// createClusterInstallation records the cluster installation(s) as a single transaction to the database, assigning it a unique ID.
func (sqlStore *SQLStore) createClusterInstallation(db execer, clusterInstallation *model.ClusterInstallation) error {
	clusterInstallation.ID = model.NewID()
	clusterInstallation.CreateAt = model.GetMillis()

	_, err := sqlStore.execBuilder(db, sq.
		Insert("ClusterInstallation").
		SetMap(map[string]interface{}{
			"ID":              clusterInstallation.ID,
			"ClusterID":       clusterInstallation.ClusterID,
			"InstallationID":  clusterInstallation.InstallationID,
			"Namespace":       clusterInstallation.Namespace,
			"State":           clusterInstallation.State,
			"CreateAt":        clusterInstallation.CreateAt,
			"DeleteAt":        0,
			"APISecurityLock": clusterInstallation.APISecurityLock,
			"LockAcquiredBy":  nil,
			"LockAcquiredAt":  0,
			"IsActive":        clusterInstallation.IsActive,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create cluster installation")
	}

	return nil
}

// SwitchDNS performs the dns switch from source cluster to target cluster
func (sqlStore *SQLStore) SwitchDNS(oldCIsIDs, newCIsIDs, installationIDs []string, hibernatingInstallationIDs []string) error {
	tx, err := sqlStore.beginTransaction(sqlStore.db)
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer tx.RollbackUnlessCommitted()

	err = sqlStore.UpdateClusterInstallationsActiveStatus(tx, oldCIsIDs, false)
	if err != nil {
		return errors.Wrap(err, "failed to update cluster installation")
	}

	err = sqlStore.UpdateClusterInstallationsActiveStatus(tx, newCIsIDs, true)
	if err != nil {
		return errors.Wrap(err, "failed to update cluster installation")
	}

	if len(installationIDs) > 0 {
		err = sqlStore.UpdateInstallationsState(tx, installationIDs, model.InstallationStateCreationDNS)
		if err != nil {
			return errors.Wrap(err, "failed to update cluster installation")
		}
	}

	if len(hibernatingInstallationIDs) > 0 {
		err = sqlStore.UpdateInstallationsState(tx, hibernatingInstallationIDs, model.InstallationStateDNSMigrationHibernating)
		if err != nil {
			return errors.Wrap(err, "failed to update cluster installation")
		}
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}
