package store

import (
	"database/sql"
	"encoding/json"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

var clusterSelect sq.SelectBuilder

func init() {
	clusterSelect = sq.
		Select(
			"ID", "Provider", "Provisioner", "ProviderMetadataRaw", "ProvisionerMetadataRaw",
			"UtilityMetadataRaw", "State", "AllowInstallations", "CreateAt", "DeleteAt",
			"LockAcquiredBy", "LockAcquiredAt",
		).
		From("Cluster")
}

// RawClusterMetadata is the raw byte metadata for a cluster.
type RawClusterMetadata struct {
	ProviderMetadataRaw    []byte
	ProvisionerMetadataRaw []byte
	UtilityMetadataRaw     []byte
}

type rawCluster struct {
	*model.Cluster
	*RawClusterMetadata
}

type rawClusters []*rawCluster

func buildRawMetadata(cluster *model.Cluster) (*RawClusterMetadata, error) {
	providerMetadataJSON, err := json.Marshal(cluster.ProviderMetadataAWS)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal ProviderMetadataAWS")
	}
	provisionerMetadataJSON, err := json.Marshal(cluster.ProvisionerMetadataKops)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal ProvisionerMetadataKops")
	}
	utilityMetadataJSON, err := json.Marshal(cluster.UtilityMetadata)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal UtilityMetadata")
	}

	return &RawClusterMetadata{
		ProviderMetadataRaw:    providerMetadataJSON,
		ProvisionerMetadataRaw: provisionerMetadataJSON,
		UtilityMetadataRaw:     utilityMetadataJSON,
	}, nil
}

func (r *rawCluster) toCluster() (*model.Cluster, error) {
	var err error
	r.Cluster.ProviderMetadataAWS, err = model.NewAWSMetadata(r.ProviderMetadataRaw)
	if err != nil {
		return nil, err
	}
	r.Cluster.ProvisionerMetadataKops, err = model.NewKopsMetadata(r.ProvisionerMetadataRaw)
	if err != nil {
		return nil, err
	}
	r.Cluster.UtilityMetadata, err = model.NewUtilityMetadata(r.UtilityMetadataRaw)
	if err != nil {
		return nil, err
	}

	return r.Cluster, nil
}

func (rc *rawClusters) toClusters() ([]*model.Cluster, error) {
	var clusters []*model.Cluster
	for _, rawCluster := range *rc {
		cluster, err := rawCluster.toCluster()
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, cluster)
	}

	return clusters, nil
}

// GetCluster fetches the given cluster by id.
func (sqlStore *SQLStore) GetCluster(id string) (*model.Cluster, error) {
	var rawCluster rawCluster
	err := sqlStore.getBuilder(sqlStore.db, &rawCluster, clusterSelect.Where("ID = ?", id))
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster by id")
	}

	return rawCluster.toCluster()
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

	var rawClusters rawClusters
	err := sqlStore.selectBuilder(sqlStore.db, &rawClusters, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for clusters")
	}

	return rawClusters.toClusters()
}

// GetUnlockedClustersPendingWork returns an unlocked cluster in a pending state.
func (sqlStore *SQLStore) GetUnlockedClustersPendingWork() ([]*model.Cluster, error) {
	builder := clusterSelect.
		Where(sq.Eq{
			"State": model.AllClusterStatesPendingWork,
		}).
		Where("LockAcquiredAt = 0").
		OrderBy("CreateAt ASC")

	var rawClusters rawClusters
	err := sqlStore.selectBuilder(sqlStore.db, &rawClusters, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for clusters")
	}

	return rawClusters.toClusters()
}

// LockCluster marks the cluster as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockCluster(clusterID, lockerID string) (bool, error) {
	return sqlStore.lockRows("Cluster", []string{clusterID}, lockerID)
}

// UnlockCluster releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockCluster(clusterID, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows("Cluster", []string{clusterID}, lockerID, force)
}

// CreateCluster records the given cluster to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateCluster(cluster *model.Cluster) error {
	cluster.ID = model.NewID()
	cluster.CreateAt = GetMillis()

	rawMetadata, err := buildRawMetadata(cluster)
	if err != nil {
		return errors.Wrap(err, "unable to build raw cluster metadata")
	}

	_, err = sqlStore.execBuilder(sqlStore.db, sq.
		Insert("Cluster").
		SetMap(map[string]interface{}{
			"ID":                     cluster.ID,
			"State":                  cluster.State,
			"Provider":               cluster.Provider,
			"ProviderMetadataRaw":    rawMetadata.ProviderMetadataRaw,
			"Provisioner":            cluster.Provisioner,
			"ProvisionerMetadataRaw": rawMetadata.ProvisionerMetadataRaw,
			"UtilityMetadataRaw":     rawMetadata.UtilityMetadataRaw,
			"AllowInstallations":     cluster.AllowInstallations,
			"CreateAt":               cluster.CreateAt,
			"DeleteAt":               0,
			"LockAcquiredBy":         nil,
			"LockAcquiredAt":         0,
			"Size":                   "DEPRECATED",
			"Version":                "DEPRECATED",
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create cluster")
	}

	return nil
}

// UpdateCluster updates the given cluster in the database.
func (sqlStore *SQLStore) UpdateCluster(cluster *model.Cluster) error {
	rawMetadata, err := buildRawMetadata(cluster)
	if err != nil {
		return errors.Wrap(err, "unable to build raw cluster metadata")
	}

	_, err = sqlStore.execBuilder(sqlStore.db, sq.
		Update("Cluster").
		SetMap(map[string]interface{}{
			"State":                  cluster.State,
			"Provider":               cluster.Provider,
			"ProviderMetadataRaw":    rawMetadata.ProviderMetadataRaw,
			"Provisioner":            cluster.Provisioner,
			"ProvisionerMetadataRaw": rawMetadata.ProvisionerMetadataRaw,
			"UtilityMetadataRaw":     rawMetadata.UtilityMetadataRaw,
			"AllowInstallations":     cluster.AllowInstallations,
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
