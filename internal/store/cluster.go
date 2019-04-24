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
		Columns(
			"ID", "Provider", "Provisioner", "ProviderMetadata", "ProvisionerMetadata",
			"AllowInstallations", "CreateAt", "DeleteAt", "LockAcquiredBy", "LockAcquiredAt",
		).
		Values(
			cluster.ID, cluster.Provider, cluster.Provisioner, cluster.ProviderMetadata,
			cluster.ProvisionerMetadata, cluster.AllowInstallations, cluster.CreateAt, 0, nil, 0,
		),
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
