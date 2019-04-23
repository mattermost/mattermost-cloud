package store

import (
	"database/sql"
	"encoding/json"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"
)

// Cluster represents a Kubernetes cluster as tracked by the store.
type Cluster struct {
	ID                  string
	Provider            string
	Provisioner         string
	ProviderMetadata    []byte
	ProvisionerMetadata []byte
	AllowInstallations  bool
	CreateAt            int64
	DeleteAt            int64
	LockAcquiredBy      string
	LockAcquiredAt      int64
}

// Clone returns a deep copy the cluster.
func (c *Cluster) Clone() *Cluster {
	var clone Cluster
	data, _ := json.Marshal(c)
	json.Unmarshal(data, &clone)

	return &clone
}

// SetProviderMetadata is a helper method to encode an interface{} as the corresponding bytes.
func (c *Cluster) SetProviderMetadata(data interface{}) error {
	if data == nil {
		c.ProviderMetadata = nil
		return nil
	}

	providerMetadata, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to set provider metadata")
	}

	c.ProviderMetadata = providerMetadata
	return nil
}

// SetProvisionerMetadata is a helper method to encode an interface{} as the corresponding bytes.
func (c *Cluster) SetProvisionerMetadata(data interface{}) error {
	if data == nil {
		c.ProvisionerMetadata = nil
		return nil
	}

	provisionerMetadata, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to set provisioner metadata")
	}

	c.ProvisionerMetadata = provisionerMetadata
	return nil
}

// GetCluster fetches the given cluster by id.
func (sqlStore *SQLStore) GetCluster(id string) (*Cluster, error) {
	var cluster Cluster

	err := sqlStore.get(sqlStore.db, &cluster, `SELECT * FROM Cluster WHERE ID = ?`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster by id")
	}

	return &cluster, nil
}

// GetClusters fetches the given page of created clusters. The first page is 0.
func (sqlStore *SQLStore) GetClusters(page, perPage int, includeDeleted bool) ([]*Cluster, error) {
	var clusters []*Cluster

	builder := sq.
		Select("*").
		From("Cluster").
		OrderBy("CreateAt ASC").
		Limit(uint64(perPage)).
		Offset(uint64(page * perPage))

	if !includeDeleted {
		builder = builder.Where("DeleteAt = 0")
	}

	err := sqlStore.selectBuilder(sqlStore.db, &clusters, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for clusters")
	}

	return clusters, nil
}

// CreateCluster records the given cluster to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateCluster(cluster *Cluster) error {
	cluster.ID = model.NewId()
	cluster.CreateAt = GetMillis()

	builder := sq.
		Insert("Cluster").
		Columns(
			"ID", "Provider", "Provisioner", "ProviderMetadata", "ProvisionerMetadata",
			"AllowInstallations", "CreateAt", "DeleteAt", "LockAcquiredBy", "LockAcquiredAt",
		).
		Values(
			cluster.ID, cluster.Provider, cluster.Provisioner, cluster.ProviderMetadata,
			cluster.ProvisionerMetadata, cluster.AllowInstallations, cluster.CreateAt, 0, "", 0,
		)

	_, err := sqlStore.execBuilder(sqlStore.db, builder)
	if err != nil {
		return errors.Wrap(err, "failed to create cluster")
	}

	return nil
}

// UpdateCluster updates the given cluster in the database.
func (sqlStore *SQLStore) UpdateCluster(cluster *Cluster) error {
	builder := sq.
		Update("Cluster").
		SetMap(map[string]interface{}{
			"Provider":            cluster.Provider,
			"Provisioner":         cluster.Provisioner,
			"ProviderMetadata":    cluster.ProviderMetadata,
			"ProvisionerMetadata": cluster.ProvisionerMetadata,
			"AllowInstallations":  cluster.AllowInstallations,
		}).
		Where("ID = ?", cluster.ID)

	_, err := sqlStore.execBuilder(sqlStore.db, builder)
	if err != nil {
		return errors.Wrap(err, "failed to update cluster")
	}

	return nil
}

// DeleteCluster marks the given cluster as deleted, but does not remove the record from the
// database.
func (sqlStore *SQLStore) DeleteCluster(id string) error {
	_, err := sqlStore.namedExec(sqlStore.db, `
		UPDATE Cluster SET DeleteAt = :DeleteAt WHERE ID = :ID AND DeleteAt = 0
	`, map[string]interface{}{
		"ID":       id,
		"DeleteAt": GetMillis(),
	})
	if err != nil {
		return errors.Wrap(err, "failed to mark cluster as deleted")
	}

	return nil
}
