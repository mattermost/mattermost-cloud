package model

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
)

const (
	// ClusterStateCreationRequested is a cluster in the process of being created.
	ClusterStateCreationRequested = "creation-requested"
	// ClusterStateCreationFailed is a cluster that failed creation.
	ClusterStateCreationFailed = "creation-failed"
	// ClusterStateProvisioningRequested is a cluster in the process of being
	// provisioned with operators.
	ClusterStateProvisioningRequested = "provisioning-requested"
	// ClusterStateProvisioningFailed is a cluster that failed provisioning.
	ClusterStateProvisioningFailed = "provisioning-failed"
	// ClusterStateDeletionRequested is a cluster in the process of being deleted.
	ClusterStateDeletionRequested = "deletion-requested"
	// ClusterStateDeletionFailed is a cluster that failed deletion.
	ClusterStateDeletionFailed = "deletion-failed"
	// ClusterStateDeleted is a cluster that has been deleted
	ClusterStateDeleted = "deleted"
	// ClusterStateUpgradeRequested is a cluster in the process of upgrading.
	ClusterStateUpgradeRequested = "upgrade-requested"
	// ClusterStateUpgradeFailed is a cluster that failed to upgrade.
	ClusterStateUpgradeFailed = "upgrade-failed"
	// ClusterStateStable is a cluster in a stable state and undergoing no changes.
	ClusterStateStable = "stable"
)

// Cluster represents a Kubernetes cluster.
type Cluster struct {
	ID                  string
	Provider            string
	Provisioner         string
	ProviderMetadata    []byte `json:",omitempty"`
	ProvisionerMetadata []byte `json:",omitempty"`
	AllowInstallations  bool
	Size                string
	State               string
	CreateAt            int64
	DeleteAt            int64
	LockAcquiredBy      *string
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

// ClusterFromReader decodes a json-encoded cluster from the given io.Reader.
func ClusterFromReader(reader io.Reader) (*Cluster, error) {
	cluster := Cluster{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&cluster)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &cluster, nil
}

// ClustersFromReader decodes a json-encoded list of clusters from the given io.Reader.
func ClustersFromReader(reader io.Reader) ([]*Cluster, error) {
	clusters := []*Cluster{}
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&clusters)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return clusters, nil
}

// ClusterFilter describes the parameters used to constrain a set of clusters.
type ClusterFilter struct {
	Page           int
	PerPage        int
	IncludeDeleted bool
}
