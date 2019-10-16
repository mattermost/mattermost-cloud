package model

import (
	"encoding/json"
	"io"

	mmv1alpha1 "github.com/mattermost/mattermost-operator/pkg/apis/mattermost/v1alpha1"
)

const (
	// InstallationStateCreationRequested is an installation waiting to be created.
	InstallationStateCreationRequested = "creation-requested"
	// InstallationStateCreationPreProvisioning in an installation in the process
	// of having managed services created along with any other preparation.
	InstallationStateCreationPreProvisioning = "creation-pre-provisioning"
	// InstallationStateCreationInProgress is an installation in the process of
	// being created.
	InstallationStateCreationInProgress = "creation-in-progress"
	// InstallationStateCreationDNS is an installation in the process having configuring DNS.
	InstallationStateCreationDNS = "creation-configuring-dns"
	// InstallationStateCreationFailed is an installation that failed creation.
	InstallationStateCreationFailed = "creation-failed"
	// InstallationStateCreationNoCompatibleClusters is an installation that
	// can't be fully created because there are no compatible clusters.
	InstallationStateCreationNoCompatibleClusters = "creation-no-compatible-clusters"
	// InstallationStateDeletionRequested is an installation to be deleted.
	InstallationStateDeletionRequested = "deletion-requested"
	// InstallationStateDeletionInProgress is an installation being deleted.
	InstallationStateDeletionInProgress = "deletion-in-progress"
	// InstallationStateDeletionFinalCleanup is the final step of installation deletion.
	InstallationStateDeletionFinalCleanup = "deletion-final-cleanup"
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

	// InstallationDefaultSize is the default size for an installation.
	InstallationDefaultSize = mmv1alpha1.Size100String
)

// Installation represents a Mattermost installation.
type Installation struct {
	ID             string
	OwnerID        string
	Version        string
	DNS            string
	Database       string
	Filestore      string
	License        string
	Size           string
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
