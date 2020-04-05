package store

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

const clusterInstallationMigrationTable = "ClusterInstallationMigration"

var clusterInstallationMigrationSelect sq.SelectBuilder

func init() {
	clusterInstallationMigrationSelect = sq.
		Select(
			"ID", "ClusterID", "ClusterInstallationID", "State", "CreateAt", "DeleteAt", "LockAcquiredBy",
			"LockAcquiredAt",
		).
		From(clusterInstallationMigrationTable)
}

// GetClusterInstallationMigration fetches the given cluster installation migration by id.
func (sqlStore *SQLStore) GetClusterInstallationMigration(id string) (*model.ClusterInstallationMigration, error) {
	var migration model.ClusterInstallationMigration
	err := sqlStore.getBuilder(sqlStore.db, &migration,
		clusterInstallationMigrationSelect.Where("ID = ?", id),
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster installation migration by id")
	}

	return &migration, nil
}

// GetUnlockedClusterInstallationMigrationsPendingWork returns an unlocked cluster installation migration in a pending state.
func (sqlStore *SQLStore) GetUnlockedClusterInstallationMigrationsPendingWork() ([]*model.ClusterInstallationMigration, error) {
	builder := clusterInstallationMigrationSelect.
		Where(sq.Eq{
			"State": model.AllCIMigrationsPendingWork,
		}).
		Where("LockAcquiredAt = 0").
		OrderBy("CreateAt ASC")

	var migrations []*model.ClusterInstallationMigration
	err := sqlStore.selectBuilder(sqlStore.db, &migrations, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster installation migration pending work")
	}

	return migrations, nil
}

// LockClusterInstallationMigration marks the installation as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockClusterInstallationMigration(migrationID, lockerID string) (bool, error) {
	return sqlStore.lockRows(clusterInstallationMigrationTable, []string{migrationID}, lockerID)
}

// UnlockClusterInstallationMigration releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockClusterInstallationMigration(migrationID, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows(clusterInstallationMigrationTable, []string{migrationID}, lockerID, force)
}

// GetClusterInstallationMigrations fetches the given page of created cluster installation migration. The first page is 0.
func (sqlStore *SQLStore) GetClusterInstallationMigrations(filter *model.ClusterInstallationMigrationFilter) ([]*model.ClusterInstallationMigration, error) {
	builder := clusterInstallationMigrationSelect.
		OrderBy("CreateAt ASC")

	if filter.PerPage != model.AllPerPage {
		builder = builder.
			Limit(uint64(filter.PerPage)).
			Offset(uint64(filter.Page * filter.PerPage))
	}

	if !filter.IncludeDeleted {
		builder = builder.Where("DeleteAt = 0")
	}

	var migrations []*model.ClusterInstallationMigration
	err := sqlStore.selectBuilder(sqlStore.db, &migrations, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for cluster installation migrations")
	}

	return migrations, nil
}

// CreateClusterInstallationMigration records the given cluster installation migration to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateClusterInstallationMigration(migration *model.ClusterInstallationMigration) error {
	migration.ID = model.NewID()
	migration.CreateAt = GetMillis()

	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Insert(clusterInstallationMigrationTable).
		SetMap(map[string]interface{}{
			"ID":                    migration.ID,
			"ClusterID":             migration.ClusterID,
			"ClusterInstallationID": migration.ClusterInstallationID,
			"State":                 migration.State,
			"CreateAt":              migration.CreateAt,
			"DeleteAt":              0,
			"LockAcquiredBy":        nil,
			"LockAcquiredAt":        0,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create cluster installation migration")
	}

	return nil
}

// UpdateClusterInstallationMigration updates the given cluster installation migration in the database.
func (sqlStore *SQLStore) UpdateClusterInstallationMigration(migration *model.ClusterInstallationMigration) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update(clusterInstallationMigrationTable).
		SetMap(map[string]interface{}{
			"ID":                    migration.ID,
			"ClusterID":             migration.ClusterID,
			"ClusterInstallationID": migration.ClusterInstallationID,
			"State":                 migration.State,
			"CreateAt":              migration.CreateAt,
			"DeleteAt":              migration.DeleteAt,
			"LockAcquiredBy":        migration.LockAcquiredBy,
			"LockAcquiredAt":        migration.LockAcquiredAt,
		}).
		Where("ID = ?", migration.ID),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update cluster installation migration")
	}

	return nil
}

// DeleteClusterInstallationMigration marks the given cluster installation migration as deleted, but does not remove the record from the
// database.
func (sqlStore *SQLStore) DeleteClusterInstallationMigration(id string) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update(clusterInstallationMigrationTable).
		Set("DeleteAt", GetMillis()).
		Where("ID = ?", id).
		Where("DeleteAt = 0"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to mark cluster installation migration as deleted")
	}

	return nil
}
