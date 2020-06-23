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
			"DeleteAt", "LockAcquiredBy", "LockAcquiredAt",
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

// LockClusterInstallations marks the cluster installation as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockClusterInstallations(clusterInstallationIDs []string, lockerID string) (bool, error) {
	return sqlStore.lockRows("ClusterInstallation", clusterInstallationIDs, lockerID)
}

// UnlockClusterInstallations releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockClusterInstallations(clusterInstallationIDs []string, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows("ClusterInstallation", clusterInstallationIDs, lockerID, force)
}

// CreateClusterInstallation records the given cluster installation to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateClusterInstallation(clusterInstallation *model.ClusterInstallation) error {
	clusterInstallation.ID = model.NewID()
	clusterInstallation.CreateAt = GetMillis()

	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Insert("ClusterInstallation").
		SetMap(map[string]interface{}{
			"ID":             clusterInstallation.ID,
			"ClusterID":      clusterInstallation.ClusterID,
			"InstallationID": clusterInstallation.InstallationID,
			"Namespace":      clusterInstallation.Namespace,
			"State":          clusterInstallation.State,
			"CreateAt":       clusterInstallation.CreateAt,
			"DeleteAt":       0,
			"LockAcquiredBy": nil,
			"LockAcquiredAt": 0,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create cluster installation")
	}

	return nil
}

// GetClusterInstallations fetches the given page of created clusters. The first page is 0.
func (sqlStore *SQLStore) GetClusterInstallations(filter *model.ClusterInstallationFilter) ([]*model.ClusterInstallation, error) {
	builder := clusterInstallationSelect.
		OrderBy("CreateAt ASC")

	if filter.PerPage != model.AllPerPage {
		builder = builder.
			Limit(uint64(filter.PerPage)).
			Offset(uint64(filter.Page * filter.PerPage))
	}

	if len(filter.IDs) > 0 {
		builder = builder.Where(sq.Eq{"ID": filter.IDs})
	}
	if filter.ClusterID != "" {
		builder = builder.Where("ClusterID = ?", filter.ClusterID)
	}
	if filter.InstallationID != "" {
		builder = builder.Where("InstallationID = ?", filter.InstallationID)
	}
	if !filter.IncludeDeleted {
		builder = builder.Where("DeleteAt = 0")
	}

	var clusterInstallations []*model.ClusterInstallation
	err := sqlStore.selectBuilder(sqlStore.db, &clusterInstallations, builder)
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

// DeleteClusterInstallation marks the given cluster installation as deleted, but does not remove
// the record from the database.
func (sqlStore *SQLStore) DeleteClusterInstallation(id string) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("ClusterInstallation").
		Set("DeleteAt", GetMillis()).
		Where("ID = ?", id).
		Where("DeleteAt = 0"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to mark cluster installation as deleted")
	}

	return nil
}
