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
	// InstallationStateWakeUpRequested is an installation that is about to be
	// woken up from hibernation.
	InstallationStateWakeUpRequested = "wake-up-requested"
	// InstallationStateUpdateRequested is an installation that is about to undergo an update.
	InstallationStateUpdateRequested = "update-requested"
	// InstallationStateUpdateInProgress is an installation that is being updated.
	InstallationStateUpdateInProgress = "update-in-progress"
	// InstallationStateUpdateFailed is an installation that failed to update.
	InstallationStateUpdateFailed = "update-failed"
	// InstallationStateImportInProgress is an installation into which a
	// Workspace archive is being imported from another service or
	// on-premise
	InstallationStateImportInProgress = "import-in-progress"
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
	// InstallationStateDBRestorationInProgress is an installation that is being restored from backup.
	InstallationStateDBRestorationInProgress = "db-restoration-in-progress"
	// InstallationStateDBMigrationInProgress is an installation that is being migrated to different database.
	InstallationStateDBMigrationInProgress = "db-migration-in-progress"
	// InstallationStateDBMigrationRollbackInProgress is an installation that is being migrated back to original database.
	InstallationStateDBMigrationRollbackInProgress = "db-migration-rollback-in-progress"
	// InstallationStateDBRestorationFailed is an installation for which database restoration failed.
	InstallationStateDBRestorationFailed = "db-restoration-failed"
	// InstallationStateDBMigrationFailed is an installation for which database migration failed.
	InstallationStateDBMigrationFailed = "db-migration-failed"
	// InstallationStateDNSMigrationHibernating is an hibernated installation that is being migrated to different cluster.
	InstallationStateDNSMigrationHibernating = "dns-migration-hibernated"
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
	InstallationStateWakeUpRequested,
	InstallationStateUpdateRequested,
	InstallationStateUpdateInProgress,
	InstallationStateUpdateFailed,
	InstallationStateImportInProgress,
	InstallationStateDeletionRequested,
	InstallationStateDeletionInProgress,
	InstallationStateDeletionFinalCleanup,
	InstallationStateDeletionFailed,
	InstallationStateDeleted,
	InstallationStateDBRestorationInProgress,
	InstallationStateDBMigrationInProgress,
	InstallationStateDBMigrationRollbackInProgress,
	InstallationStateDBRestorationFailed,
	InstallationStateDBMigrationFailed,
	InstallationStateDNSMigrationHibernating,
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
	InstallationStateWakeUpRequested,
	InstallationStateUpdateRequested,
	InstallationStateUpdateInProgress,
	InstallationStateDeletionRequested,
	InstallationStateDeletionInProgress,
	InstallationStateDeletionFinalCleanup,
	InstallationStateDNSMigrationHibernating,
}

// AllInstallationRequestStates is a list of all states that an installation can
// be put in via the API.
// Warning:
// When creating a new installation state, it must be added to this list if an
// API endpoint should put the installation in this state.
var AllInstallationRequestStates = []string{
	InstallationStateCreationRequested,
	InstallationStateHibernationRequested,
	InstallationStateWakeUpRequested,
	InstallationStateUpdateRequested,
	InstallationStateDeletionRequested,
	InstallationStateDNSMigrationHibernating,
}

// ValidTransitionState returns whether an installation can be transitioned into
// the new state or not based on its current state.
func (i *Installation) ValidTransitionState(newState string) bool {
	validStates, found := validInstallationTransitions[newState]
	if !found {
		return false
	}

	return contains(validStates, i.State)
}

var (
	validInstallationTransitions = map[string][]string{
		InstallationStateCreationRequested: {
			InstallationStateCreationRequested,
			InstallationStateCreationFailed,
		},
		InstallationStateHibernationRequested: {
			InstallationStateStable,
		},
		InstallationStateWakeUpRequested: {
			InstallationStateHibernating,
		},
		InstallationStateUpdateRequested: {
			InstallationStateStable,
			InstallationStateUpdateRequested,
			InstallationStateUpdateInProgress,
			InstallationStateUpdateFailed,
		},
		InstallationStateDeletionRequested: {
			InstallationStateStable,
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
			InstallationStateImportInProgress,
			InstallationStateHibernationRequested,
			InstallationStateHibernationInProgress,
			InstallationStateHibernating,
			InstallationStateDeletionRequested,
			InstallationStateDeletionInProgress,
			InstallationStateDeletionFinalCleanup,
			InstallationStateDeletionFailed,
		},
		InstallationStateDBRestorationInProgress: {
			InstallationStateHibernating,
		},
		InstallationStateDBMigrationInProgress: {
			InstallationStateHibernating,
		},
		InstallationStateDNSMigrationHibernating: {
			InstallationStateHibernating,
		},
	}
)

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
