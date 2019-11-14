package model

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

// AllClusterInstallationStates is a list of all states a cluster installation
// can be in.
// Warning:
// When creating a new cluster installation state, it must be added to this list.
var AllClusterInstallationStates = []string{
	ClusterInstallationStateCreationRequested,
	ClusterInstallationStateCreationFailed,
	ClusterInstallationStateDeletionRequested,
	ClusterInstallationStateDeletionFailed,
	ClusterInstallationStateDeleted,
	ClusterInstallationStateReconciling,
	ClusterInstallationStateStable,
}

// AllClusterInstallationStatesPendingWork is a list of all cluster installation
// states that the supervisor will attempt to transition towards stable on the
// next "tick".
// Warning:
// When creating a new cluster installation state, it must be added to this list
// if the cloud installation supervisor should perform some action on its next
// work cycle.
var AllClusterInstallationStatesPendingWork = []string{
	ClusterInstallationStateCreationRequested,
	ClusterInstallationStateReconciling,
	ClusterInstallationStateDeletionRequested,
}
