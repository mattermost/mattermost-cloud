package model

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// DatabaseClusterInstallations is a container that holds a collection of installation IDs.
type DatabaseClusterInstallations []string

// Size returns the number of installations in the container.
func (d *DatabaseClusterInstallations) Size() int {
	return len(*d)
}

// Add inserts a new installation in the container.
func (d *DatabaseClusterInstallations) Add(installationID string) {
	*d = append(*d, installationID)
}

// Contains checks if the supplied installation ID exists in the container.
func (d *DatabaseClusterInstallations) Contains(installationID string) bool {
	for _, installation := range *d {
		if installation == installationID {
			return true
		}
	}

	return false
}

// Remove deletes the installation from the container.
func (d *DatabaseClusterInstallations) Remove(installationID string) bool {
	for i, installation := range *d {
		if installation == installationID {
			(*d) = append((*d)[:i], (*d)[i+1:]...)
			return true
		}
	}
	return false
}

// DatabaseCluster represents a cluster that manages multiple databases.
type DatabaseCluster struct {
	ID                 string
	RawInstallationIDs []byte `json:",omitempty"`
	LockAcquiredBy     *string
	LockAcquiredAt     int64
}

// SetInstallations is a helper method to encode an interface{} as the corresponding bytes.
func (c *DatabaseCluster) SetInstallations(dbInstallations DatabaseClusterInstallations) error {
	if dbInstallations.Size() == 0 {
		c.RawInstallationIDs = nil
		return nil
	}

	installations, err := json.Marshal(dbInstallations)
	if err != nil {
		return errors.Wrap(err, "failed to set installations in the database cluster")
	}

	c.RawInstallationIDs = installations
	return nil
}

// GetInstallations is a helper method to encode an interface{} as the corresponding bytes.
func (c *DatabaseCluster) GetInstallations() (DatabaseClusterInstallations, error) {
	if len(c.RawInstallationIDs) < 1 {
		return DatabaseClusterInstallations{}, nil
	}

	installations := DatabaseClusterInstallations{}

	err := json.Unmarshal(c.RawInstallationIDs, &installations)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get installations in the database cluster")
	}

	return installations, nil
}

// DatabaseClusterFilter filter results based on a specific installation ID.
type DatabaseClusterFilter struct {
	InstallationID          string
	NumOfInstallationsLimit uint32
}
