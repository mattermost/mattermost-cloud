package model

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClusterClone(t *testing.T) {
	cluster := &Cluster{
		Provider:            "aws",
		Provisioner:         "kops",
		ProviderMetadata:    []byte(`{"provider": "test1"}`),
		ProvisionerMetadata: []byte(`{"provisioner": "test1"}`),
		AllowInstallations:  false,
	}

	clone := cluster.Clone()
	require.Equal(t, cluster, clone)

	// Verify changing pointers in the clone doesn't affect the original.
	clone.ProviderMetadata = []byte("override")
	clone.ProvisionerMetadata = []byte("override")
	require.NotEqual(t, cluster, clone)
}

func TestSetProviderMetadata(t *testing.T) {
	t.Run("set nil", func(t *testing.T) {
		cluster := Cluster{}
		err := cluster.SetProviderMetadata(nil)
		require.NoError(t, err)
		require.Nil(t, cluster.ProviderMetadata)
	})

	t.Run("set data", func(t *testing.T) {
		cluster := Cluster{}
		err := cluster.SetProviderMetadata(struct{ Test string }{"test"})
		require.NoError(t, err)
		require.Equal(t, `{"Test":"test"}`, string(cluster.ProviderMetadata))
	})
}

func TestSetProvisionerMetadata(t *testing.T) {
	t.Run("set nil", func(t *testing.T) {
		cluster := Cluster{}
		err := cluster.SetProvisionerMetadata(nil)
		require.NoError(t, err)
		require.Nil(t, cluster.ProvisionerMetadata)
	})

	t.Run("set data", func(t *testing.T) {
		cluster := Cluster{}
		err := cluster.SetProvisionerMetadata(struct{ Test string }{"test"})
		require.NoError(t, err)
		require.Equal(t, `{"Test":"test"}`, string(cluster.ProvisionerMetadata))
	})
}

func TestClusterFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		cluster, err := ClusterFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, &Cluster{}, cluster)
	})

	t.Run("invalid request", func(t *testing.T) {
		cluster, err := ClusterFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, cluster)
	})

	t.Run("request", func(t *testing.T) {
		cluster, err := ClusterFromReader(bytes.NewReader([]byte(
			`{"ID":"id","Provider":"aws"}`,
		)))
		require.NoError(t, err)
		require.Equal(t, &Cluster{ID: "id", Provider: "aws"}, cluster)
	})
}

func TestClustersFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		clusters, err := ClustersFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, []*Cluster{}, clusters)
	})

	t.Run("invalid request", func(t *testing.T) {
		clusters, err := ClustersFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, clusters)
	})

	t.Run("request", func(t *testing.T) {
		cluster, err := ClustersFromReader(bytes.NewReader([]byte(
			`[{"ID":"id1", "Provider":"aws"}, {"ID":"id2", "Provider":"aws"}]`,
		)))
		require.NoError(t, err)
		require.Equal(t, []*Cluster{
			&Cluster{ID: "id1", Provider: "aws"},
			&Cluster{ID: "id2", Provider: "aws"},
		}, cluster)
	})
}

func TestValidClusterVersion(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"latest", true},
		{"0.0.0", true},
		{"1.1.1", true},
		{"1.11.11", true},
		{"1.111.111", true},
		{"1.12.34", true},
		{"1.12.0", true},
		{"0.12.34", true},
		{"latest1", false},
		{"lates", false},
		{"1.12.34.56", false},
		{"1111.1.2", false},
		{"bad.bad.bad", false},
		{"1.bad.2", false},
		{".", false},
		{"..", false},
		{"...", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.valid, ValidClusterVersion(test.name))
		})
	}
}
