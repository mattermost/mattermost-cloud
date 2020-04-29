package store

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

var databaseClusterSelect sq.SelectBuilder

func init() {
	databaseClusterSelect = sq.
		Select("ID", "RawInstallations", "LockAcquiredBy", "LockAcquiredAt").
		From("DatabaseCluster")
}

// GetDatabaseCluster fetches the given database cluster by id.
func (sqlStore *SQLStore) GetDatabaseCluster(id string) (*model.DatabaseCluster, error) {
	var databaseCluster model.DatabaseCluster
	err := sqlStore.getBuilder(sqlStore.db, &databaseCluster, databaseClusterSelect.Where("ID = ?", id))
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster by id")
	}

	return &databaseCluster, nil
}

// GetUnlockedDatabaseCluster returns an unlocked cluster in a pending state.
func (sqlStore *SQLStore) GetUnlockedDatabaseCluster(id string) ([]*model.DatabaseCluster, error) {
	builder := clusterSelect.
		Where("LockAcquiredAt = 0").
		Where("ID = ?", id).
		OrderBy("CreateAt ASC")

	var databaseClusters []*model.DatabaseCluster
	err := sqlStore.selectBuilder(sqlStore.db, &databaseClusters, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database cluster pending work")
	}

	return databaseClusters, nil
}

// LockDatabaseCluster marks the database cluster as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockDatabaseCluster(installationID, lockerID string) (bool, error) {
	return sqlStore.lockRows("DatabaseCluster", []string{installationID}, lockerID)
}

// UnlockDatabaseCluster releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockDatabaseCluster(installationID, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows("DatabaseCluster", []string{installationID}, lockerID, force)
}

// GetDatabaseClusters fetches the given page of created database clusters. The first page is 0.
func (sqlStore *SQLStore) GetDatabaseClusters(filter *model.DatabaseClusterFilter) ([]*model.DatabaseCluster, error) {
	builder := clusterSelect.
		OrderBy("CreateAt ASC")

	if filter != nil && filter.PerPage != model.AllPerPage {
		builder = builder.
			Limit(uint64(filter.PerPage)).
			Offset(uint64(filter.Page * filter.PerPage))
	}

	var databaseClusters []*model.DatabaseCluster
	err := sqlStore.selectBuilder(sqlStore.db, &databaseClusters, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for clusters")
	}

	return databaseClusters, nil
}

// CreateDatabaseCluster records the given cluster to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateDatabaseCluster(databaseCluster *model.DatabaseCluster) error {
	if databaseCluster == nil {
		return errors.New("database cluster cannot be nil")
	}

	if len(databaseCluster.ID) < 1 {
		return errors.New("database cluster ID cannot be nil")
	}

	if len(databaseCluster.RawInstallations) < 1 {
		databaseCluster.RawInstallations = make([]byte, 0)
	}

	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Insert("DatabaseCluster").
		SetMap(map[string]interface{}{
			"ID":               databaseCluster.ID,
			"RawInstallations": databaseCluster.RawInstallations,
			"LockAcquiredBy":   nil,
			"LockAcquiredAt":   0,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create database cluster")
	}

	return nil
}

// UpdateDatabaseCluster updates the given database cluster in the database.
func (sqlStore *SQLStore) UpdateDatabaseCluster(databaseCluster *model.DatabaseCluster) error {
	if databaseCluster == nil {
		return errors.New("database cluster cannot be nil")
	}

	if len(databaseCluster.RawInstallations) < 1 {
		databaseCluster.RawInstallations = make([]byte, 0)
	}

	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("DatabaseCluster").
		SetMap(map[string]interface{}{
			"RawInstallations": databaseCluster.RawInstallations,
		}).
		Where("ID = ?", databaseCluster.ID),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update database cluster")
	}

	return nil
}
