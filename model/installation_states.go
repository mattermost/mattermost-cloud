package model

import mmv1alpha1 "github.com/mattermost/mattermost-operator/pkg/apis/mattermost/v1alpha1"

const (
	// InstallationStateStable is an installation in a stable state and undergoing no changes.
	InstallationStateStable = "stable"
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
	// InstallationStateUpgradeRequested is an installation that is about to undergo a version change.
	InstallationStateUpgradeRequested = "upgrade-requested"
	// InstallationStateUpgradeInProgress is an installation that is undergoing a version change.
	InstallationStateUpgradeInProgress = "upgrade-in-progress"
	// InstallationStateUpgradeFailed is an installation that failed to change versions.
	InstallationStateUpgradeFailed = "upgrade-failed"
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
)

const (
	// InstallationDefaultSize is the default size for an installation.
	InstallationDefaultSize = mmv1alpha1.Size100String
)

// AllInstallationStates is a list of all states an installation can be in.
// Warning:
// When creating a new installation state, it must be added to this list.
var AllInstallationStates = []string{
	InstallationStateStable,
	InstallationStateCreationRequested,
	InstallationStateCreationPreProvisioning,
	InstallationStateCreationInProgress,
	InstallationStateCreationDNS,
	InstallationStateCreationFailed,
	InstallationStateCreationNoCompatibleClusters,
	InstallationStateUpgradeRequested,
	InstallationStateUpgradeInProgress,
	InstallationStateUpgradeFailed,
	InstallationStateDeletionRequested,
	InstallationStateDeletionInProgress,
	InstallationStateDeletionFinalCleanup,
	InstallationStateDeletionFailed,
	InstallationStateDeleted,
}

// AllInstallationStatesPendingWork is a list of all installation states that
// the supervisor will attempt to transition towards stable on the next "tick".
// Warning:
// When creating a new installation state, it must be added to this list if the
// cloud installation supervisor should perform some action on its next work cycle.
var AllInstallationStatesPendingWork = []string{
	InstallationStateCreationRequested,
	InstallationStateCreationPreProvisioning,
	InstallationStateCreationInProgress,
	InstallationStateCreationNoCompatibleClusters,
	InstallationStateCreationDNS,
	InstallationStateUpgradeRequested,
	InstallationStateUpgradeInProgress,
	InstallationStateDeletionRequested,
	InstallationStateDeletionInProgress,
	InstallationStateDeletionFinalCleanup,
}

// AllInstallationRequestStates is a list of all states that an installation can
// be put in via the API.
// Warning:
// When creating a new installation state, it must be added to this list if an
// API endpoint should put the installation in this state.
var AllInstallationRequestStates = []string{
	InstallationStateCreationRequested,
	InstallationStateUpgradeRequested,
	InstallationStateDeletionRequested,
}

// ValidTransitionState returns whether an installation can be transitioned into
// the new state or not based on its current state.
func (i *Installation) ValidTransitionState(newState string) bool {
	switch newState {
	case InstallationStateCreationRequested:
		return validTransitionToInstallationStateCreationRequested(i.State)
	case InstallationStateUpgradeRequested:
		return validTransitionToInstallationStateUpgradeRequested(i.State)
	case InstallationStateDeletionRequested:
		return validTransitionToInstallationStateDeletionRequested(i.State)
	}

	return false
}

func validTransitionToInstallationStateCreationRequested(currentState string) bool {
	switch currentState {
	case InstallationStateCreationRequested,
		InstallationStateCreationFailed:
		return true
	}

	return false
}

func validTransitionToInstallationStateUpgradeRequested(currentState string) bool {
	switch currentState {
	case InstallationStateStable,
		InstallationStateUpgradeRequested,
		InstallationStateUpgradeInProgress,
		InstallationStateUpgradeFailed:
		return true
	}

	return false
}

func validTransitionToInstallationStateDeletionRequested(currentState string) bool {
	switch currentState {
	case InstallationStateStable,
		InstallationStateCreationRequested,
		InstallationStateCreationPreProvisioning,
		InstallationStateCreationInProgress,
		InstallationStateCreationDNS,
		InstallationStateCreationNoCompatibleClusters,
		InstallationStateCreationFailed,
		InstallationStateUpgradeRequested,
		InstallationStateUpgradeInProgress,
		InstallationStateUpgradeFailed,
		InstallationStateDeletionRequested,
		InstallationStateDeletionInProgress,
		InstallationStateDeletionFinalCleanup,
		InstallationStateDeletionFailed:
		return true
	}

	return false
}

// InstallationStateReport is a report of all installation requests states.
type InstallationStateReport []StateReportEntry

// GetInstallationRequestStateReport returns a InstallationStateReport based on
// the current model of installation states.
func GetInstallationRequestStateReport() InstallationStateReport {
	report := InstallationStateReport{}

	for _, requestState := range AllInstallationRequestStates {
		entry := StateReportEntry{
			RequestedState: requestState,
		}

		for _, newState := range AllInstallationStates {
			i := Installation{State: newState}
			if i.ValidTransitionState(requestState) {
				entry.ValidStates = append(entry.ValidStates, newState)
			} else {
				entry.InvalidStates = append(entry.InvalidStates, newState)
			}
		}

		report = append(report, entry)
	}

	return report
}
