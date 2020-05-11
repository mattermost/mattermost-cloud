package store

import (
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

var databaseClusterSelect sq.SelectBuilder

func init() {
	databaseClusterSelect = sq.
		Select("ID", "RawInstallationIDs", "LockAcquiredBy", "LockAcquiredAt").
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

// GetDatabaseClusters fetches the given page of created database clusters. The first page is 0.
func (sqlStore *SQLStore) GetDatabaseClusters(filter *model.DatabaseClusterFilter) ([]*model.DatabaseCluster, error) {
	builder := databaseClusterSelect

	if filter != nil && len(filter.InstallationID) > 0 {
		builder = builder.Where("RawInstallationIDs LIKE ?", fmt.Sprint("%", filter.InstallationID, "%"))
	}

	var databaseClusters []*model.DatabaseCluster

	err := sqlStore.selectBuilder(sqlStore.db, &databaseClusters, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for database clusters")
	}

	if filter != nil && filter.NumOfInstallationsLimit > 0 {
		for i, cluster := range databaseClusters {
			installationIDs, err := cluster.GetInstallationIDs()
			if err != nil {
				return nil, errors.Wrap(err, "failed to query for database clusters installations")
			}

			if len(installationIDs) > int(filter.NumOfInstallationsLimit) {
				databaseClusters = append(databaseClusters[:i], databaseClusters[i+1:]...)
			}
		}
	}

	return databaseClusters, nil
}

// CreateDatabaseCluster records the given cluster to the database.
func (sqlStore *SQLStore) CreateDatabaseCluster(databaseCluster *model.DatabaseCluster) error {
	if databaseCluster == nil {
		return errors.New("database cluster cannot be nil")
	}

	if len(databaseCluster.ID) < 1 {
		return errors.New("database cluster ID cannot be nil")
	}

	if len(databaseCluster.RawInstallationIDs) < 1 {
		databaseCluster.RawInstallationIDs = make([]byte, 0)
	}

	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Insert("DatabaseCluster").
		SetMap(map[string]interface{}{
			"ID":                 databaseCluster.ID,
			"RawInstallationIDs": databaseCluster.RawInstallationIDs,
			"LockAcquiredBy":     nil,
			"LockAcquiredAt":     0,
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

	if len(databaseCluster.RawInstallationIDs) < 1 {
		databaseCluster.RawInstallationIDs = make([]byte, 0)
	}

	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("DatabaseCluster").
		SetMap(map[string]interface{}{
			"RawInstallationIDs": databaseCluster.RawInstallationIDs,
		}).
		Where("ID = ?", databaseCluster.ID),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update database cluster")
	}

	return nil
}

// LockDatabaseCluster marks the database cluster as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockDatabaseCluster(installationID, lockerID string) (bool, error) {
	return sqlStore.lockRows("DatabaseCluster", []string{installationID}, lockerID)
}

// UnlockDatabaseCluster releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockDatabaseCluster(installationID, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows("DatabaseCluster", []string{installationID}, lockerID, force)
}
