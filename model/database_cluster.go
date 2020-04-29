package model

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
)

// DatabaseClusterInstallations ...
type DatabaseClusterInstallations []string

// Size ...
func (d *DatabaseClusterInstallations) Size() int {
	return len(*d)
}

// Add ...
func (d *DatabaseClusterInstallations) Add(installationID string) {
	*d = append(*d, installationID)
}

// Contains ...
func (d *DatabaseClusterInstallations) Contains(installationID string) bool {
	for _, installation := range *d {
		if installation == installationID {
			return true
		}
	}

	return false
}

// Remove ...
func (d *DatabaseClusterInstallations) Remove(installationID string) bool {
	for i, installation := range *d {
		if installation == installationID {
			(*d) = append((*d)[:i], (*d)[i+1:]...)
			return true
		}
	}
	return false
}

// DatabaseCluster represents a Kubernetes cluster.
type DatabaseCluster struct {
	ID               string
	RawInstallations []byte `json:",omitempty"`
	LockAcquiredBy   *string
	LockAcquiredAt   int64
}

// SetInstallations is a helper method to encode an interface{} as the corresponding bytes.
func (c *DatabaseCluster) SetInstallations(dbInstallations DatabaseClusterInstallations) error {
	if dbInstallations.Size() == 0 {
		c.RawInstallations = nil
		return nil
	}

	installations, err := json.Marshal(dbInstallations)
	if err != nil {
		return errors.Wrap(err, "failed to set installations in the database cluster")
	}

	c.RawInstallations = installations
	return nil
}

// GetInstallations is a helper method to encode an interface{} as the corresponding bytes.
func (c *DatabaseCluster) GetInstallations() (DatabaseClusterInstallations, error) {
	if len(c.RawInstallations) < 1 {
		return DatabaseClusterInstallations{}, nil
	}

	installations := DatabaseClusterInstallations{}

	err := json.Unmarshal(c.RawInstallations, &installations)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get installations in the database cluster")
	}

	return installations, nil
}

// DatabaseClusterFromReader decodes a json-encoded DatabaseCluster from the given io.Reader.
func DatabaseClusterFromReader(reader io.Reader) (*DatabaseCluster, error) {
	databaseCluster := DatabaseCluster{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&databaseCluster)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &databaseCluster, nil
}

// DatabaseClustersFromReader decodes a json-encoded list of clusters from the given io.Reader.
func DatabaseClustersFromReader(reader io.Reader) ([]*DatabaseCluster, error) {
	databaseClusters := []*DatabaseCluster{}
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&databaseClusters)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return databaseClusters, nil
}

type DatabaseClusterFilter struct {
	Page    int
	PerPage int
}
