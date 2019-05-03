package store

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/internal/model"
	"github.com/pkg/errors"
)

// GetCluster fetches the given cluster by id.
func (sqlStore *SQLStore) GetCluster(id string) (*model.Cluster, error) {
	var cluster model.Cluster
	err := sqlStore.getBuilder(sqlStore.db, &cluster,
		sq.Select("*").From("Cluster").Where("ID = ?", id),
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster by id")
	}

	return &cluster, nil
}

// GetUnlockedClusterPendingWork returns an unlocked cluster in a pending state.
func (sqlStore *SQLStore) GetUnlockedClusterPendingWork() (*model.Cluster, error) {
	var cluster model.Cluster
	err := sqlStore.getBuilder(sqlStore.db, &cluster, sq.
		Select("*").
		From("Cluster").
		Where(sq.Eq{
			"State": []string{
				model.ClusterStateCreationRequested,
				model.ClusterStateUpgradeRequested,
				model.ClusterStateDeletionRequested,
			},
		}).
		Where("LockAcquiredAt = 0").
		OrderBy("State ASC"),
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster pending work")
	}

	return &cluster, nil
}

// LockCluster marks the cluster as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockCluster(clusterID, lockerID string) (bool, error) {
	result, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("Cluster").
		SetMap(map[string]interface{}{
			"LockAcquiredBy": lockerID,
			"LockAcquiredAt": GetMillis(),
		}).
		Where(sq.Eq{
			"ID":             clusterID,
			"LockAcquiredAt": 0,
		}),
	)
	if err != nil {
		return false, errors.Wrap(err, "failed to lock cluster")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, errors.Wrap(err, "failed to count rows affected")
	}

	locked := false
	if count > 0 {
		locked = true
	}

	return locked, nil
}

// UnlockCluster releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockCluster(clusterID, lockerID string, force bool) (bool, error) {
	builder := sq.Update("Cluster").
		SetMap(map[string]interface{}{
			"LockAcquiredBy": nil,
			"LockAcquiredAt": 0,
		}).
		Where(sq.Eq{
			"ID": clusterID,
		})

	if force {
		// If forcing the unlock, only require that a lock was held by someone.
		builder = builder.Where("LockAcquiredAt <> 0")
	} else {
		// If not forcing the unlock, require that the current instance held the lock.
		builder = builder.Where(sq.Eq{
			"LockAcquiredBy": lockerID,
		})
	}

	result, err := sqlStore.execBuilder(sqlStore.db, builder)
	if err != nil {
		return false, errors.Wrap(err, "failed to unlock cluster")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return false, errors.Wrap(err, "failed to count rows affected")
	}

	unlocked := false
	if count > 0 {
		unlocked = true
	}

	return unlocked, nil
}

// GetClusters fetches the given page of created clusters. The first page is 0.
func (sqlStore *SQLStore) GetClusters(page, perPage int, includeDeleted bool) ([]*model.Cluster, error) {
	builder := sq.
		Select("*").
		From("Cluster").
		OrderBy("CreateAt ASC").
		Limit(uint64(perPage)).
		Offset(uint64(page * perPage))

	if !includeDeleted {
		builder = builder.Where("DeleteAt = 0")
	}

	var clusters []*model.Cluster
	err := sqlStore.selectBuilder(sqlStore.db, &clusters, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for clusters")
	}

	return clusters, nil
}

// CreateCluster records the given cluster to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateCluster(cluster *model.Cluster) error {
	cluster.ID = model.NewID()
	cluster.CreateAt = GetMillis()

	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Insert("Cluster").
		SetMap(map[string]interface{}{
			"ID":                  cluster.ID,
			"Provider":            cluster.Provider,
			"Provisioner":         cluster.Provisioner,
			"ProviderMetadata":    cluster.ProviderMetadata,
			"ProvisionerMetadata": cluster.ProvisionerMetadata,
			"Size":                cluster.Size,
			"State":               cluster.State,
			"AllowInstallations":  cluster.AllowInstallations,
			"CreateAt":            cluster.CreateAt,
			"DeleteAt":            0,
			"LockAcquiredBy":      nil,
			"LockAcquiredAt":      0,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create cluster")
	}

	return nil
}

// UpdateCluster updates the given cluster in the database.
func (sqlStore *SQLStore) UpdateCluster(cluster *model.Cluster) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("Cluster").
		SetMap(map[string]interface{}{
			"Provider":            cluster.Provider,
			"Provisioner":         cluster.Provisioner,
			"ProviderMetadata":    cluster.ProviderMetadata,
			"ProvisionerMetadata": cluster.ProvisionerMetadata,
			"Size":                cluster.Size,
			"State":               cluster.State,
			"AllowInstallations":  cluster.AllowInstallations,
		}).
		Where("ID = ?", cluster.ID),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update cluster")
	}

	return nil
}

// DeleteCluster marks the given cluster as deleted, but does not remove the record from the
// database.
func (sqlStore *SQLStore) DeleteCluster(id string) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("Cluster").
		Set("DeleteAt", GetMillis()).
		Where("ID = ?", id).
		Where("DeleteAt = ?", 0),
	)
	if err != nil {
		return errors.Wrap(err, "failed to mark cluster as deleted")
	}

	return nil
}
