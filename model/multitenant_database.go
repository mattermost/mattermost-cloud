package model

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// MultitenantDatabaseInstallationIDs is a container that holds a collection of installation IDs.
type MultitenantDatabaseInstallationIDs []string

// Add inserts a new installation in the container.
func (d *MultitenantDatabaseInstallationIDs) Add(installationID string) {
	*d = append(*d, installationID)
}

// Contains checks if the supplied installation ID exists in the container.
func (d *MultitenantDatabaseInstallationIDs) Contains(installationID string) bool {
	for _, id := range *d {
		if id == installationID {
			return true
		}
	}

	return false
}

// Remove deletes the installation from the container.
func (d *MultitenantDatabaseInstallationIDs) Remove(installationID string) {
	for i, installation := range *d {
		if installation == installationID {
			(*d) = append((*d)[:i], (*d)[i+1:]...)
		}
	}
}

// MultitenantDatabase represents a cluster that manages multiple databases.
type MultitenantDatabase struct {
	ID                 string
	RawInstallationIDs []byte `json:",omitempty"`
	LockAcquiredBy     *string
	LockAcquiredAt     int64
}

// SetInstallationIDs is a helper method to parse DatabaseClusterInstallationIDs to the corresponding JSON-encoded bytes.
func (c *MultitenantDatabase) SetInstallationIDs(installationIDs MultitenantDatabaseInstallationIDs) error {
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
func (c *MultitenantDatabase) GetInstallationIDs() (MultitenantDatabaseInstallationIDs, error) {
	installationIDs := MultitenantDatabaseInstallationIDs{}

	if len(c.RawInstallationIDs) < 1 {
		return installationIDs, nil
	}

	err := json.Unmarshal(c.RawInstallationIDs, &installationIDs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get installations in the database cluster")
	}

	return installationIDs, nil
}

// MultitenantDatabaseFilter filters results based on a specific installation ID and a number of
// installation's limit.
type MultitenantDatabaseFilter struct {
	InstallationID          string
	NumOfInstallationsLimit uint32
}
