// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import mmv1alpha1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1alpha1"

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
	// InstallationStateCreationFinalTasks is the final step of the installation creation.
	InstallationStateCreationFinalTasks = "creation-final-tasks"
	// InstallationStateHibernationRequested is an installation that is about
	// to be put into hibernation.
	InstallationStateHibernationRequested = "hibernation-requested"
	// InstallationStateHibernationInProgress is an installation that is
	// transitioning to hibernation.
	InstallationStateHibernationInProgress = "hibernation-in-progress"
	// InstallationStateHibernating is an installation that is hibernating.
	InstallationStateHibernating = "hibernating"
	// InstallationStateUpdateRequested is an installation that is about to undergo an update.
	InstallationStateUpdateRequested = "update-requested"
	// InstallationStateUpdateInProgress is an installation that is being updated.
	InstallationStateUpdateInProgress = "update-in-progress"
	// InstallationStateUpdateFailed is an installation that failed to update.
	InstallationStateUpdateFailed = "update-failed"
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
	InstallationStateCreationFinalTasks,
	InstallationStateHibernationRequested,
	InstallationStateHibernationInProgress,
	InstallationStateHibernating,
	InstallationStateUpdateRequested,
	InstallationStateUpdateInProgress,
	InstallationStateUpdateFailed,
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
	InstallationStateCreationFinalTasks,
	InstallationStateCreationDNS,
	InstallationStateHibernationRequested,
	InstallationStateHibernationInProgress,
	InstallationStateUpdateRequested,
	InstallationStateUpdateInProgress,
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
	InstallationStateHibernationRequested,
	InstallationStateUpdateRequested,
	InstallationStateDeletionRequested,
}

// ValidTransitionState returns whether an installation can be transitioned into
// the new state or not based on its current state.
func (i *Installation) ValidTransitionState(newState string) bool {
	switch newState {
	case InstallationStateCreationRequested:
		return validTransitionToInstallationStateCreationRequested(i.State)
	case InstallationStateHibernationRequested:
		return validTransitionToInstallationStateHibernationRequested(i.State)
	case InstallationStateUpdateRequested:
		return validTransitionToInstallationStateUpdateRequested(i.State)
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

func validTransitionToInstallationStateHibernationRequested(currentState string) bool {
	switch currentState {
	case InstallationStateStable:
		return true
	}

	return false
}

func validTransitionToInstallationStateUpdateRequested(currentState string) bool {
	switch currentState {
	case InstallationStateStable,
		InstallationStateHibernating,
		InstallationStateUpdateRequested,
		InstallationStateUpdateInProgress,
		InstallationStateUpdateFailed:
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
		InstallationStateCreationFinalTasks,
		InstallationStateCreationFailed,
		InstallationStateUpdateRequested,
		InstallationStateUpdateInProgress,
		InstallationStateUpdateFailed,
		InstallationStateHibernationRequested,
		InstallationStateHibernationInProgress,
		InstallationStateHibernating,
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
