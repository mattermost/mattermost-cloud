package store

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/internal/model"
	"github.com/pkg/errors"
)

var clusterSelect sq.SelectBuilder

func init() {
	clusterSelect = sq.
		Select(
			"ID", "Provider", "Provisioner", "ProviderMetadata", "ProvisionerMetadata",
			"Size", "State", "AllowInstallations", "CreateAt", "DeleteAt",
			"LockAcquiredBy", "LockAcquiredAt",
		).
		From("Cluster")
}

// GetCluster fetches the given cluster by id.
func (sqlStore *SQLStore) GetCluster(id string) (*model.Cluster, error) {
	var cluster model.Cluster
	err := sqlStore.getBuilder(sqlStore.db, &cluster, clusterSelect.Where("ID = ?", id))
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
	err := sqlStore.getBuilder(sqlStore.db, &cluster, clusterSelect.
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
	return sqlStore.lockRow("Cluster", clusterID, lockerID)
}

// UnlockCluster releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockCluster(clusterID, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRow("Cluster", clusterID, lockerID, force)
}

// GetClusters fetches the given page of created clusters. The first page is 0.
func (sqlStore *SQLStore) GetClusters(filter *model.ClusterFilter) ([]*model.Cluster, error) {
	builder := clusterSelect.
		OrderBy("CreateAt ASC")

	if filter.PerPage != model.AllPerPage {
		builder = builder.
			Limit(uint64(filter.PerPage)).
			Offset(uint64(filter.Page * filter.PerPage))
	}

	if !filter.IncludeDeleted {
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
		Where("DeleteAt = 0"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to mark cluster as deleted")
	}

	return nil
}
