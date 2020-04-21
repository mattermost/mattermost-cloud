package model

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClusterInstallationClone(t *testing.T) {
	clusterInstallation := &ClusterInstallation{
		ID:             "id",
		ClusterID:      NewID(),
		InstallationID: NewID(),
		Namespace:      "namespace",
		State:          ClusterInstallationStateStable,
	}

	clone := clusterInstallation.Clone()
	require.Equal(t, clusterInstallation, clone)

	// Verify changing values in the clone doesn't affect the original.
	clone.Namespace = "new"
	require.NotEqual(t, clusterInstallation, clone)
}

func TestClusterInstallationIsDeleted(t *testing.T) {
	clusterInstallation := &ClusterInstallation{
		DeleteAt: 0,
	}

	t.Run("not deleted", func(t *testing.T) {
		require.False(t, clusterInstallation.IsDeleted())
	})

	clusterInstallation.DeleteAt = 1

	t.Run("deleted", func(t *testing.T) {
		require.True(t, clusterInstallation.IsDeleted())
	})
}

func TestClusterInstallationFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		clusterInstallation, err := ClusterInstallationFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, &ClusterInstallation{}, clusterInstallation)
	})

	t.Run("invalid request", func(t *testing.T) {
		clusterInstallation, err := ClusterInstallationFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, clusterInstallation)
	})

	t.Run("request", func(t *testing.T) {
		clusterInstallation, err := ClusterInstallationFromReader(bytes.NewReader([]byte(`{
			"ID":"id",
			"ClusterID":"cluster_id",
			"InstallationID":"installation_id",
			"Namespace":"namespace",
			"State":"state",
			"CreateAt":10,
			"DeleteAt":20,
			"LockAcquiredAt":0
		}`)))
		require.NoError(t, err)
		require.Equal(t, &ClusterInstallation{
			ID:             "id",
			ClusterID:      "cluster_id",
			InstallationID: "installation_id",
			Namespace:      "namespace",
			State:          "state",
			CreateAt:       10,
			DeleteAt:       20,
			LockAcquiredBy: nil,
			LockAcquiredAt: int64(0),
		}, clusterInstallation)
	})
}

func TestClusterInstallationsFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		clusterInstallations, err := ClusterInstallationsFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, []*ClusterInstallation{}, clusterInstallations)
	})

	t.Run("invalid request", func(t *testing.T) {
		clusterInstallations, err := ClusterInstallationsFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, clusterInstallations)
	})

	t.Run("request", func(t *testing.T) {
		clusterInstallation, err := ClusterInstallationsFromReader(bytes.NewReader([]byte(`[
			{
				"ID":"id1",
				"ClusterID":"cluster_id1",
				"InstallationID":"installation_id1",
				"Namespace":"namespace1",
				"State":"state1",
				"CreateAt":10,
				"DeleteAt":20,
				"LockAcquiredAt":0
			},
			{
				"ID":"id2",
				"ClusterID":"cluster_id2",
				"InstallationID":"installation_id2",
				"Namespace":"namespace2",
				"State":"state2",
				"CreateAt":30,
				"DeleteAt":40,
				"LockAcquiredBy": "tester2",
				"LockAcquiredAt":50
			}
		]`)))
		require.NoError(t, err)
		require.Equal(t, []*ClusterInstallation{
			{
				ID:             "id1",
				ClusterID:      "cluster_id1",
				InstallationID: "installation_id1",
				Namespace:      "namespace1",
				State:          "state1",
				CreateAt:       10,
				DeleteAt:       20,
				LockAcquiredBy: nil,
				LockAcquiredAt: 0,
			},
			{
				ID:             "id2",
				ClusterID:      "cluster_id2",
				InstallationID: "installation_id2",
				Namespace:      "namespace2",
				State:          "state2",
				CreateAt:       30,
				DeleteAt:       40,
				LockAcquiredBy: sToP("tester2"),
				LockAcquiredAt: 50,
			},
		}, clusterInstallation)
	})
}

func TestClusterInstallationConfigFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		config, err := ClusterInstallationConfigFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, map[string]interface{}{}, config)
	})

	t.Run("invalid request", func(t *testing.T) {
		config, err := ClusterInstallationConfigFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, config)
	})

	t.Run("request", func(t *testing.T) {
		config, err := ClusterInstallationConfigFromReader(bytes.NewReader([]byte(
			`{"ServiceSettings":{"SiteURL":"test.example.com"}}`,
		)))
		require.NoError(t, err)
		require.Equal(t, map[string]interface{}{"ServiceSettings": map[string]interface{}{"SiteURL": "test.example.com"}}, config)
	})
}
