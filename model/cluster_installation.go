package model

import (
	"encoding/json"
	"io"
)

const (
	// ClusterInstallationStateCreationRequested is a cluster installation in the process of being created.
	ClusterInstallationStateCreationRequested = "creation-requested"
	// ClusterInstallationStateCreationFailed is a cluster installation that failed creation.
	ClusterInstallationStateCreationFailed = "creation-failed"
	// ClusterInstallationStateDeletionRequested is a cluster installation in the process of being deleted.
	ClusterInstallationStateDeletionRequested = "deletion-requested"
	// ClusterInstallationStateDeletionFailed is a cluster installation that failed deletion.
	ClusterInstallationStateDeletionFailed = "deletion-failed"
	// ClusterInstallationStateDeleted is a cluster installation that has been deleted
	ClusterInstallationStateDeleted = "deleted"
	// ClusterInstallationStateReconciling is a cluster installation that in undergoing changes and is not yet stable.
	ClusterInstallationStateReconciling = "reconciling"
	// ClusterInstallationStateStable is a cluster installation in a stable state and undergoing no changes.
	ClusterInstallationStateStable = "stable"
)

// ClusterInstallation is a single namespace within a cluster composing a potentially larger installation.
type ClusterInstallation struct {
	ID             string
	ClusterID      string
	InstallationID string
	Namespace      string
	State          string
	CreateAt       int64
	DeleteAt       int64
	LockAcquiredBy *string
	LockAcquiredAt int64
}

// ClusterInstallationFilter describes the parameters used to constrain a set of cluster installations.
type ClusterInstallationFilter struct {
	IDs            []string
	InstallationID string
	ClusterID      string
	Page           int
	PerPage        int
	IncludeDeleted bool
}

// Clone returns a deep copy the cluster installation.
func (c *ClusterInstallation) Clone() *ClusterInstallation {
	var clone ClusterInstallation
	data, _ := json.Marshal(c)
	json.Unmarshal(data, &clone)

	return &clone
}

// IsDeleted returns whether the cluster installation was marked as deleted or not.
func (c *ClusterInstallation) IsDeleted() bool {
	return c.DeleteAt != 0
}

// ClusterInstallationFromReader decodes a json-encoded cluster installation from the given io.Reader.
func ClusterInstallationFromReader(reader io.Reader) (*ClusterInstallation, error) {
	clusterInstallation := ClusterInstallation{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&clusterInstallation)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &clusterInstallation, nil
}

// ClusterInstallationsFromReader decodes a json-encoded list of cluster installations from the given io.Reader.
func ClusterInstallationsFromReader(reader io.Reader) ([]*ClusterInstallation, error) {
	clusterInstallations := []*ClusterInstallation{}
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&clusterInstallations)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return clusterInstallations, nil
}

// ClusterInstallationConfigFromReader decodes a json-encoded config from the config io.Reader.
func ClusterInstallationConfigFromReader(reader io.Reader) (map[string]interface{}, error) {
	config := make(map[string]interface{})
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&config)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return config, nil
}
