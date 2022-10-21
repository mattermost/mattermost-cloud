package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func getTestGroups() []*GroupDTO {
	return []*GroupDTO{
		{
			Group: &Group{
				ID:              "id1",
				Sequence:        0,
				Name:            "group1",
				Version:         "1",
				APISecurityLock: false,
				LockAcquiredBy:  nil,
				LockAcquiredAt:  int64(0),
			},
			Annotations: []*Annotation{{ID: "id", Name: "foo"}},
			Status: &GroupStatus{
				InstallationsTotal:   1,
				InstallationsUpdated: 2,
			},
		},
		{
			Group: &Group{
				ID:              "id2",
				Sequence:        0,
				Name:            "group2",
				Version:         "1",
				APISecurityLock: false,
				LockAcquiredBy:  nil,
				LockAcquiredAt:  int64(0),
			},
			Annotations: []*Annotation{{ID: "id", Name: "foo"}},
			Status: &GroupStatus{
				InstallationsTotal:   2,
				InstallationsUpdated: 1,
			},
		},
		{
			Group: &Group{
				ID:              "id2",
				Sequence:        0,
				Name:            "group2",
				Version:         "1",
				APISecurityLock: false,
				LockAcquiredBy:  nil,
				LockAcquiredAt:  int64(0),
			},
			Annotations: []*Annotation{{ID: "id", Name: "foo"}},
			Status: &GroupStatus{
				InstallationsTotal:   3,
				InstallationsUpdated: 2,
			},
		},
	}
}

// TestGroupAllocator common tests for group allocators
func TestGroupAllocator(t *testing.T) {
	allocators := map[string]InstallationGroupAllocator{
		"Random":        NewRandomInstallationGroupAllocator(),
		"LowestTotal":   NewLowestCountInstallationGroupAllocator("total"),
		"LowestUpdated": NewLowestCountInstallationGroupAllocator("updated"),
	}

	for name, a := range allocators {
		t.Run(name, func(t *testing.T) {
			testGroupAllocatorOk(t, a)
		})
	}
}

// testGroupAllocatorOk checks that the provider allocator at least returns a group with no errors
func testGroupAllocatorOk(t *testing.T, allocator InstallationGroupAllocator) {
	groups := getTestGroups()
	selectedGroup, err := allocator.Choose(groups)
	assert.NoError(t, err)
	assert.NotNil(t, selectedGroup)
}

// TestLowestInstallationCountTotal checks that the lowest installation count by total works ok
func TestLowestInstallationCountTotal(t *testing.T) {
	groups := getTestGroups()

	allocator := NewLowestCountInstallationGroupAllocator("total")

	chosenGroup, err := allocator.Choose(groups)
	assert.NoError(t, err)
	assert.Equal(t, groups[0].ID, chosenGroup.ID)
}

// TestLowestInstallationCountUpdated checks that the lowest installation count by updated works ok
func TestLowestInstallationCountUpdated(t *testing.T) {
	groups := getTestGroups()

	allocator := NewLowestCountInstallationGroupAllocator("updated")

	chosenGroup, err := allocator.Choose(groups)
	assert.NoError(t, err)
	assert.Equal(t, groups[1].ID, chosenGroup.ID)
}
