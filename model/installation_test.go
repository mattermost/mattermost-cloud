package model

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallationClone(t *testing.T) {
	installation := &Installation{
		ID:       "id",
		OwnerID:  "owner",
		Version:  "version",
		DNS:      "test.example.com",
		License:  "this_is_my_license",
		Affinity: InstallationAffinityIsolated,
		GroupID:  sToP("group_id"),
		State:    InstallationStateStable,
	}

	clone := installation.Clone()
	require.Equal(t, installation, clone)

	// Verify changing pointers in the clone doesn't affect the original.
	clone.GroupID = sToP("new_group_id")
	require.NotEqual(t, installation, clone)
}

func TestInstallationFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		installation, err := InstallationFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, &Installation{}, installation)
	})

	t.Run("invalid request", func(t *testing.T) {
		installation, err := InstallationFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, installation)
	})

	t.Run("request", func(t *testing.T) {
		installation, err := InstallationFromReader(bytes.NewReader([]byte(`{
			"ID":"id",
			"OwnerID":"owner",
			"Version":"version",
			"DNS":"dns",
			"License": "this_is_my_license",
			"Affinity":"affinity",
			"GroupID":"group_id",
			"State":"state",
			"CreateAt":10,
			"DeleteAt":20,
			"LockAcquiredAt":0
		}`)))
		require.NoError(t, err)
		require.Equal(t, &Installation{
			ID:             "id",
			OwnerID:        "owner",
			Version:        "version",
			DNS:            "dns",
			License:        "this_is_my_license",
			Affinity:       "affinity",
			GroupID:        sToP("group_id"),
			State:          "state",
			CreateAt:       10,
			DeleteAt:       20,
			LockAcquiredBy: nil,
			LockAcquiredAt: int64(0),
		}, installation)
	})
}

func TestInstallationsFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		installations, err := InstallationsFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, []*Installation{}, installations)
	})

	t.Run("invalid request", func(t *testing.T) {
		installations, err := InstallationsFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, installations)
	})

	t.Run("request", func(t *testing.T) {
		installation, err := InstallationsFromReader(bytes.NewReader([]byte(`[
			{
				"ID":"id1",
				"OwnerID":"owner1",
				"Version":"version1",
				"DNS":"dns1",
				"Affinity":"affinity1",
				"GroupID":"group_id1",
				"State":"state1",
				"CreateAt":10,
				"DeleteAt":20,
				"LockAcquiredAt":0
			},
			{
				"ID":"id2",
				"OwnerID":"owner2",
				"Version":"version2",
				"DNS":"dns2",
				"License": "this_is_my_license",
				"Affinity":"affinity2",
				"GroupID":"group_id2",
				"State":"state2",
				"CreateAt":30,
				"DeleteAt":40,
				"LockAcquiredBy": "tester",
				"LockAcquiredAt":50
			}
		]`)))
		require.NoError(t, err)
		require.Equal(t, []*Installation{
			&Installation{
				ID:             "id1",
				OwnerID:        "owner1",
				Version:        "version1",
				DNS:            "dns1",
				Affinity:       "affinity1",
				GroupID:        sToP("group_id1"),
				State:          "state1",
				CreateAt:       10,
				DeleteAt:       20,
				LockAcquiredBy: nil,
				LockAcquiredAt: 0,
			},
			&Installation{
				ID:             "id2",
				OwnerID:        "owner2",
				Version:        "version2",
				DNS:            "dns2",
				License:        "this_is_my_license",
				Affinity:       "affinity2",
				GroupID:        sToP("group_id2"),
				State:          "state2",
				CreateAt:       30,
				DeleteAt:       40,
				LockAcquiredBy: sToP("tester"),
				LockAcquiredAt: 50,
			},
		}, installation)
	})
}

func TestIsValidInstallationSize(t *testing.T) {
	t.Run("valid sizes", func(t *testing.T) {
		for _, size := range InstallationSizes {
			t.Run(size, func(t *testing.T) {
				assert.True(t, IsValidInstallationSize(size))
			})
		}
	})

	t.Run("invalid sizes", func(t *testing.T) {
		var invalidSizes = []string{"invalid", "1user", ""}

		for _, size := range invalidSizes {
			t.Run(size, func(t *testing.T) {
				assert.False(t, IsValidInstallationSize(size))
			})
		}
	})
}
