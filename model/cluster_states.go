package model

const (
	// ClusterStateStable is a cluster in a stable state and undergoing no changes.
	ClusterStateStable = "stable"
	// ClusterStateCreationRequested is a cluster in the process of being created.
	ClusterStateCreationRequested = "creation-requested"
	// ClusterStateCreationFailed is a cluster that failed creation.
	ClusterStateCreationFailed = "creation-failed"
	// ClusterStateProvisioningRequested is a cluster in the process of being
	// provisioned with operators.
	ClusterStateProvisioningRequested = "provisioning-requested"
	// ClusterStateProvisioningFailed is a cluster that failed provisioning.
	ClusterStateProvisioningFailed = "provisioning-failed"
	// ClusterStateUpgradeRequested is a cluster in the process of upgrading.
	ClusterStateUpgradeRequested = "upgrade-requested"
	// ClusterStateUpgradeFailed is a cluster that failed to upgrade.
	ClusterStateUpgradeFailed = "upgrade-failed"
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
	ClusterStateCreationRequested,
	ClusterStateCreationFailed,
	ClusterStateProvisioningRequested,
	ClusterStateProvisioningFailed,
	ClusterStateUpgradeRequested,
	ClusterStateUpgradeFailed,
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
	ClusterStateProvisioningRequested,
	ClusterStateUpgradeRequested,
	ClusterStateDeletionRequested,
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
	ClusterStateDeletionRequested,
}

// ValidTransitionState returns whether a cluster can be transitioned into the
// new state or not based on its current state.
func (c *Cluster) ValidTransitionState(newState string) bool {
	switch newState {
	case ClusterStateCreationRequested:
		return validTransitionToClusterStateCreationRequested(c.State)
	case ClusterStateProvisioningRequested:
		return validTransitionToClusterStateProvisioningRequested(c.State)
	case ClusterStateUpgradeRequested:
		return validTransitionToClusterStateUpgradeRequested(c.State)
	case ClusterStateDeletionRequested:
		return validTransitionToClusterStateDeletionRequested(c.State)
	}

	return false
}

func validTransitionToClusterStateCreationRequested(currentState string) bool {
	switch currentState {
	case ClusterStateCreationRequested,
		ClusterStateCreationFailed:
		return true
	}

	return false
}

func validTransitionToClusterStateProvisioningRequested(currentState string) bool {
	switch currentState {
	case ClusterStateStable,
		ClusterStateProvisioningFailed,
		ClusterStateProvisioningRequested:
		return true
	}

	return false
}

func validTransitionToClusterStateUpgradeRequested(currentState string) bool {
	switch currentState {
	case ClusterStateStable,
		ClusterStateUpgradeRequested,
		ClusterStateUpgradeFailed:
		return true
	}

	return false
}

func validTransitionToClusterStateDeletionRequested(currentState string) bool {
	switch currentState {
	case ClusterStateStable,
		ClusterStateCreationRequested,
		ClusterStateCreationFailed,
		ClusterStateProvisioningFailed,
		ClusterStateUpgradeRequested,
		ClusterStateUpgradeFailed,
		ClusterStateDeletionRequested,
		ClusterStateDeletionFailed:
		return true
	}

	return false
}

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
