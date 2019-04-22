package api

import (
	"bytes"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewCreateClusterRequestFromReader(t *testing.T) {
	defaultCreateClusterRequest := &CreateClusterRequest{
		Provider: "aws",
		Size:     "SizeAlef500",
		Zones:    []string{"us-east-1a"},
	}

	t.Run("empty request", func(t *testing.T) {
		clusterRequest, err := newCreateClusterRequestFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, defaultCreateClusterRequest, clusterRequest)
	})

	t.Run("invalid request", func(t *testing.T) {
		clusterRequest, err := newCreateClusterRequestFromReader(bytes.NewReader([]byte(
			`{`,
		)))
		require.Error(t, err)
		require.Nil(t, clusterRequest)
	})

	t.Run("partial request", func(t *testing.T) {
		clusterRequest, err := newCreateClusterRequestFromReader(bytes.NewReader([]byte(
			`{"Size": "SizeAlef1000"}`,
		)))
		require.NoError(t, err)
		require.Equal(t, &CreateClusterRequest{Provider: "aws", Size: "SizeAlef1000", Zones: []string{"us-east-1a"}}, clusterRequest)
	})

	t.Run("full request", func(t *testing.T) {
		clusterRequest, err := newCreateClusterRequestFromReader(bytes.NewReader([]byte(
			`{"Provider": "azure", "Size": "SizeAlef1000", "Zones":["zone1", "zone2"]}`,
		)))
		require.NoError(t, err)
		require.Equal(t, &CreateClusterRequest{Provider: "azure", Size: "SizeAlef1000", Zones: []string{"zone1", "zone2"}}, clusterRequest)
	})
}

func TestGetClustersRequestApplyToURL(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		u, err := url.Parse("http://localhost:8075")
		require.NoError(t, err)

		getClustersRequest := &GetClustersRequest{}
		getClustersRequest.ApplyToURL(u)

		require.Equal(t, "page=0&per_page=0", u.RawQuery)
	})

	t.Run("changes", func(t *testing.T) {
		u, err := url.Parse("http://localhost:8075")
		require.NoError(t, err)

		getClustersRequest := &GetClustersRequest{
			Page:           10,
			PerPage:        123,
			IncludeDeleted: true,
		}
		getClustersRequest.ApplyToURL(u)

		require.Equal(t, "include_deleted=true&page=10&per_page=123", u.RawQuery)
	})
}
