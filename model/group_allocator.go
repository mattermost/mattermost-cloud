package model

import (
	"math/rand"
)

// InstallationGroupAllocator defines a way to allocate installations into groups based on
// each allocator internal logic.
type InstallationGroupAllocator interface {
	Choose(groups []*GroupDTO) (*GroupDTO, error)
}

// RandomInstallationGroupAllocator allocates the installation in a random group
type RandomInstallationGroupAllocator struct{}

func (iga *RandomInstallationGroupAllocator) Choose(groups []*GroupDTO) (*GroupDTO, error) {
	return groups[rand.Intn(len(groups))], nil
}

// NewRandomInstallationGroupAllocator
func NewRandomInstallationGroupAllocator() InstallationGroupAllocator {
	return &RandomInstallationGroupAllocator{}
}

// LowestCountInstallationGroupAllocator allocates the installation in the group that has the lowest
// count for the defined state
type LowestCountInstallationGroupAllocator struct {
	// the state to check
	installationState string
}

// getStateValue gets the value for the selected state to check from the provided group
// defaults to the total instalations silently if the provided state is not handled
func (iga *LowestCountInstallationGroupAllocator) getStateValue(group *GroupDTO) int64 {
	switch iga.installationState {
	case "updated":
		return group.Status.InstallationsUpdated
	default:
		return group.Status.InstallationsTotal
	}
}

// getLowestInstallationCount retrieves the lowest installation count group based on the defined state field
func (iga *LowestCountInstallationGroupAllocator) getLowestInstallationCount(groups []*GroupDTO) (group *GroupDTO) {
	for _, g := range groups {
		if group == nil || iga.getStateValue(g) < iga.getStateValue(group) {
			group = g
		}
	}

	return
}

func (iga *LowestCountInstallationGroupAllocator) Choose(groups []*GroupDTO) (*GroupDTO, error) {
	return iga.getLowestInstallationCount(groups), nil
}

// NewLowestCountInstallationGroupAllocator
func NewLowestCountInstallationGroupAllocator(state string) InstallationGroupAllocator {
	return &LowestCountInstallationGroupAllocator{
		installationState: state,
	}
}
