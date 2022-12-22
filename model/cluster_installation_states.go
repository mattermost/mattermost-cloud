// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

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
	// ClusterInstallationStateReady is a cluster installation in a ready state where it is nearly stable.
	ClusterInstallationStateReady = "ready"
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
	ClusterInstallationStateReady,
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
	ClusterInstallationStateReady,
	ClusterInstallationStateDeletionRequested,
}

// ClusterInstallationStateWorkPriority is a map of states to their priority. Default priority is 0.
// States with higher priority will be processed first.
var ClusterInstallationStateWorkPriority = map[string]int{
	ClusterInstallationStateCreationRequested: 3,
	ClusterInstallationStateReconciling:       2,
	ClusterInstallationStateReady:             1,
}
