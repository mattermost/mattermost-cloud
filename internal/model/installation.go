package model

import (
	"encoding/json"
	"io"
)

const (
	// InstallationStateCreationRequested is an installation in the process of being created.
	InstallationStateCreationRequested = "creation-requested"
	// InstallationStateCreationFailed is an installation that failed creation.
	InstallationStateCreationFailed = "creation-failed"
	// InstallationStateDeletionRequested is an installation to be deleted.
	InstallationStateDeletionRequested = "deletion-requested"
	// InstallationStateDeletionInProgress is an installation being deleted.
	InstallationStateDeletionInProgress = "deletion-in-progress"
	// InstallationStateDeletionFailed is an installation that failed deletion.
	InstallationStateDeletionFailed = "deletion-failed"
	// InstallationStateDeleted is an installation that has been deleted
	InstallationStateDeleted = "deleted"
	// InstallationStateUpgradeRequested is an installation that is about to undergo a version change.
	InstallationStateUpgradeRequested = "upgrade-requested"
	// InstallationStateUpgradeInProgress is an installation that is undergoing a version change.
	InstallationStateUpgradeInProgress = "upgrade-in-progress"
	// InstallationStateUpgradeFailed is an installation that failed to change versions.
	InstallationStateUpgradeFailed = "upgrade-failed"
	// InstallationStateStable is an installation in a stable state and undergoing no changes.
	InstallationStateStable = "stable"
)

// Installation represents a Mattermost installation.
type Installation struct {
	ID             string
	OwnerID        string
	Version        string
	DNS            string
	Affinity       string
	GroupID        *string
	State          string
	CreateAt       int64
	DeleteAt       int64
	LockAcquiredBy *string
	LockAcquiredAt int64
}

// InstallationFilter describes the parameters used to constrain a set of installations.
type InstallationFilter struct {
	OwnerID        string
	Page           int
	PerPage        int
	IncludeDeleted bool
}

// Clone returns a deep copy the installation.
func (c *Installation) Clone() *Installation {
	var clone Installation
	data, _ := json.Marshal(c)
	json.Unmarshal(data, &clone)

	return &clone
}

// InstallationFromReader decodes a json-encoded installation from the given io.Reader.
func InstallationFromReader(reader io.Reader) (*Installation, error) {
	installation := Installation{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&installation)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &installation, nil
}

// InstallationsFromReader decodes a json-encoded list of installations from the given io.Reader.
func InstallationsFromReader(reader io.Reader) ([]*Installation, error) {
	installations := []*Installation{}
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&installations)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return installations, nil
}
