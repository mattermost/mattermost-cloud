package model

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// DatabaseClusterInstallationIDs is a container that holds a collection of installation IDs.
type DatabaseClusterInstallationIDs []string

// Add inserts a new installation in the container.
func (d *DatabaseClusterInstallationIDs) Add(installationID string) {
	*d = append(*d, installationID)
}

// Contains checks if the supplied installation ID exists in the container.
func (d *DatabaseClusterInstallationIDs) Contains(installationID string) bool {
	for _, id := range *d {
		if id == installationID {
			return true
		}
	}

	return false
}

// Remove deletes the installation from the container.
func (d *DatabaseClusterInstallationIDs) Remove(installationID string) {
	for i, installation := range *d {
		if installation == installationID {
			(*d) = append((*d)[:i], (*d)[i+1:]...)
		}
	}
}

// DatabaseCluster represents a cluster that manages multiple databases.
type DatabaseCluster struct {
	ID                 string
	RawInstallationIDs []byte `json:",omitempty"`
	LockAcquiredBy     *string
	LockAcquiredAt     int64
}

// SetInstallationIDs is a helper method to parse DatabaseClusterInstallationIDs to the corresponding JSON-encoded bytes.
func (c *DatabaseCluster) SetInstallationIDs(installationIDs DatabaseClusterInstallationIDs) error {
	if len(installationIDs) == 0 {
		c.RawInstallationIDs = nil
		return nil
	}

	installations, err := json.Marshal(installationIDs)
	if err != nil {
		return errors.Wrap(err, "failed to set installations in the database cluster")
	}

	c.RawInstallationIDs = installations
	return nil
}

// GetInstallationIDs is a helper method to parse JSON-encoded bytes to DatabaseClusterInstallationIDs.
func (c *DatabaseCluster) GetInstallationIDs() (DatabaseClusterInstallationIDs, error) {
	installationIDs := DatabaseClusterInstallationIDs{}

	if len(c.RawInstallationIDs) < 1 {
		return installationIDs, nil
	}

	err := json.Unmarshal(c.RawInstallationIDs, &installationIDs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get installations in the database cluster")
	}

	return installationIDs, nil
}

// DatabaseClusterFilter filter results based on a specific installation ID and a number of
// installation's limit.
type DatabaseClusterFilter struct {
	InstallationID          string
	NumOfInstallationsLimit uint32
}
