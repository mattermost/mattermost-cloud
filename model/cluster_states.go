// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

const (
	// ClusterStateStable is a cluster in a stable state and undergoing no changes.
	ClusterStateStable = "stable"
	// ClusterStateCreationRequested is a cluster in the process of being created.
	ClusterStateCreationRequested = "creation-requested"
	// ClusterStateCreationInProgress is a cluster that is being actively created.
	ClusterStateCreationInProgress = "creation-in-progress"
	// ClusterStateWaitingForNodes is a cluster that is waiting for nodes to be ready
	ClusterStateWaitingForNodes = "waiting-for-nodes"
	// ClusterStateProvisionInProgress is a cluster in the process of being provisioned.
	ClusterStateProvisionInProgress = "provision-in-progress"
	// ClusterStateCreationFailed is a cluster that failed creation.
	ClusterStateCreationFailed = "creation-failed"
	// ClusterStateProvisioningRequested is a cluster in the process of being
	// provisioned with operators.
	ClusterStateProvisioningRequested = "provisioning-requested"
	// ClusterStateRefreshMetadata is a cluster that will have metadata refreshed.
	ClusterStateRefreshMetadata = "refresh-metadata"
	// ClusterStateProvisioningFailed is a cluster that failed provisioning.
	ClusterStateProvisioningFailed = "provisioning-failed"
	// ClusterStateUpgradeRequested is a cluster in the process of upgrading.
	ClusterStateUpgradeRequested = "upgrade-requested"
	// ClusterStateUpgradeFailed is a cluster that failed to upgrade.
	ClusterStateUpgradeFailed = "upgrade-failed"
	// ClusterStateResizeRequested is a cluster in the process of resizing.
	ClusterStateResizeRequested = "resize-requested"
	// ClusterStateResizeFailed is a cluster that failed to resize.
	ClusterStateResizeFailed = "resize-failed"
	// ClusterStateNodegroupsCreationRequested is a cluster in the process of creating nodegroups.
	ClusterStateNodegroupsCreationRequested = "nodegroups-creation-requested"
	// ClusterStateNodegroupsCreationFailed is a cluster that failed to create nodegroups.
	ClusterStateNodegroupsCreationFailed = "nodegroups-creation-failed"
	// ClusterStateNodegroupsDeletionRequested is a cluster in the process of deleting nodegroups.
	ClusterStateNodegroupsDeletionRequested = "nodegroups-deletion-requested"
	// ClusterStateNodegroupsDeletionFailed is a cluster that failed to delete nodegroups.
	ClusterStateNodegroupsDeletionFailed = "nodegroups-deletion-failed"
	// ClusterStateDeletionRequested is a cluster in the process of being deleted.
	ClusterStateDeletionRequested = "deletion-requested"
	// ClusterStateDeletionFailed is a cluster that failed deletion.
	ClusterStateDeletionFailed = "deletion-failed"
	// ClusterStateDeleted is a cluster that has been deleted
	ClusterStateDeleted = "deleted"
)

// AllClusterStates is a list of all states a cluster can be in.
// Warning:
// When creating a new cluster state, it must be added to this list.
var AllClusterStates = []string{
	ClusterStateStable,
	ClusterStateRefreshMetadata,
	ClusterStateCreationRequested,
	ClusterStateCreationInProgress,
	ClusterStateWaitingForNodes,
	ClusterStateProvisionInProgress,
	ClusterStateCreationFailed,
	ClusterStateProvisioningRequested,
	ClusterStateProvisioningFailed,
	ClusterStateNodegroupsCreationRequested,
	ClusterStateNodegroupsCreationFailed,
	ClusterStateUpgradeRequested,
	ClusterStateUpgradeFailed,
	ClusterStateResizeRequested,
	ClusterStateResizeFailed,
	ClusterStateDeletionRequested,
	ClusterStateDeletionFailed,
	ClusterStateDeleted,
}

// AllClusterStatesPendingWork is a list of all cluster states that the supervisor
// will attempt to transition towards stable on the next "tick".
// Warning:
// When creating a new cluster state, it must be added to this list if the cloud
// cluster supervisor should perform some action on its next work cycle.
var AllClusterStatesPendingWork = []string{
	ClusterStateCreationRequested,
	ClusterStateCreationInProgress,
	ClusterStateWaitingForNodes,
	ClusterStateProvisionInProgress,
	ClusterStateProvisioningRequested,
	ClusterStateRefreshMetadata,
	ClusterStateUpgradeRequested,
	ClusterStateResizeRequested,
	ClusterStateNodegroupsCreationRequested,
	ClusterStateNodegroupsDeletionRequested,
	ClusterStateDeletionRequested,
}

// ClusterStateWorkPriority is a map of states to their priority. Default priority is 0.
// States with higher priority will be processed first.
var ClusterStateWorkPriority = map[string]int{
	ClusterStateCreationRequested:   4,
	ClusterStateCreationInProgress:  3,
	ClusterStateWaitingForNodes:     2,
	ClusterStateProvisionInProgress: 1,
}

// AllClusterRequestStates is a list of all states that a cluster can be put in
// via the API.
// Warning:
// When creating a new cluster state, it must be added to this list if an API
// endpoint should put the cluster in this state.
var AllClusterRequestStates = []string{
	ClusterStateCreationRequested,
	ClusterStateProvisioningRequested,
	ClusterStateUpgradeRequested,
	ClusterStateResizeRequested,
	ClusterStateNodegroupsCreationRequested,
	ClusterStateDeletionRequested,
}

// ValidTransitionState returns whether a cluster can be transitioned into the
// new state or not based on its current state.
func (c *Cluster) ValidTransitionState(newState string) bool {
	validStates, found := validClusterTransitions[newState]
	if !found {
		return false
	}

	return contains(validStates, c.State)
}

var (
	validClusterTransitions = map[string][]string{
		ClusterStateCreationRequested: {
			ClusterStateCreationRequested,
			ClusterStateCreationFailed,
		},
		ClusterStateProvisioningRequested: {
			ClusterStateStable,
			ClusterStateProvisioningFailed,
			ClusterStateProvisioningRequested,
		},
		ClusterStateUpgradeRequested: {
			ClusterStateStable,
			ClusterStateUpgradeRequested,
			ClusterStateUpgradeFailed,
		},
		ClusterStateResizeRequested: {
			ClusterStateStable,
			ClusterStateResizeRequested,
			ClusterStateResizeFailed,
		},
		ClusterStateNodegroupsCreationRequested: {
			ClusterStateStable,
			ClusterStateNodegroupsCreationRequested,
			ClusterStateNodegroupsCreationFailed,
		},
		ClusterStateNodegroupsDeletionRequested: {
			ClusterStateStable,
			ClusterStateNodegroupsDeletionRequested,
			ClusterStateNodegroupsDeletionFailed,
		},
		ClusterStateDeletionRequested: {
			ClusterStateStable,
			ClusterStateCreationRequested,
			ClusterStateCreationFailed,
			ClusterStateProvisioningFailed,
			ClusterStateUpgradeRequested,
			ClusterStateUpgradeFailed,
			ClusterStateDeletionRequested,
			ClusterStateDeletionFailed,
		},
	}
)

// ClusterStateReport is a report of all cluster requests states.
type ClusterStateReport []StateReportEntry

// StateReportEntry is a report entry of a given request state.
type StateReportEntry struct {
	RequestedState string
	ValidStates    StateList
	InvalidStates  StateList
}

// StateList is a list of states
type StateList []string

// Count provides the number of states in a StateList.
func (sl *StateList) Count() int {
	return len(*sl)
}

// GetClusterRequestStateReport returns a ClusterStateReport based on the current
// model of cluster states.
func GetClusterRequestStateReport() ClusterStateReport {
	report := ClusterStateReport{}

	for _, requestState := range AllClusterRequestStates {
		entry := StateReportEntry{
			RequestedState: requestState,
		}

		for _, newState := range AllClusterStates {
			c := Cluster{State: newState}
			if c.ValidTransitionState(requestState) {
				entry.ValidStates = append(entry.ValidStates, newState)
			} else {
				entry.InvalidStates = append(entry.InvalidStates, newState)
			}
		}

		report = append(report, entry)
	}

	return report
}
