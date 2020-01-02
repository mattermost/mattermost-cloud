package api

import (
	"bytes"
	"net/url"
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/require"
)

func TestNewCreateClusterRequestFromReader(t *testing.T) {
	defaultCreateClusterRequest := func() *model.CreateClusterRequest {
		return &model.CreateClusterRequest{
			Provider: "aws",
			Version:  "latest",
			Size:     "SizeAlef500",
			Zones:    []string{"us-east-1a"},
		}
	}

	t.Run("empty request", func(t *testing.T) {
		clusterRequest, err := model.NewCreateClusterRequestFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, defaultCreateClusterRequest(), clusterRequest)
	})

	t.Run("invalid request", func(t *testing.T) {
		clusterRequest, err := model.NewCreateClusterRequestFromReader(bytes.NewReader([]byte(
			`{`,
		)))
		require.Error(t, err)
		require.Nil(t, clusterRequest)
	})

	t.Run("unsupported provider", func(t *testing.T) {
		clusterRequest, err := model.NewCreateClusterRequestFromReader(bytes.NewReader([]byte(
			`{"Provider": "azure", "Size": "SizeAlef1000", "Zones":["zone1", "zone2"]}`,
		)))
		require.EqualError(t, err, "create cluster request failed validation: unsupported provider azure")
		require.Nil(t, clusterRequest)
	})

	t.Run("partial request", func(t *testing.T) {
		clusterRequest, err := model.NewCreateClusterRequestFromReader(bytes.NewReader([]byte(
			`{"Size": "SizeAlef1000"}`,
		)))
		require.NoError(t, err)
		modifiedDefaultCreateClusterRequest := defaultCreateClusterRequest()
		modifiedDefaultCreateClusterRequest.Size = "SizeAlef1000"
		require.Equal(t, modifiedDefaultCreateClusterRequest, clusterRequest)
	})

	t.Run("full request", func(t *testing.T) {
		clusterRequest, err := model.NewCreateClusterRequestFromReader(bytes.NewReader([]byte(
			`{"Provider": "aws", "Version": "1.12.4", "Size": "SizeAlef1000", "Zones": ["zone1", "zone2"]}`,
		)))
		require.NoError(t, err)
		require.Equal(t, &model.CreateClusterRequest{Provider: model.ProviderAWS, Version: "1.12.4", Size: model.SizeAlef1000, Zones: []string{"zone1", "zone2"}}, clusterRequest)
	})
}

func TestGetClustersRequestApplyToURL(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		u, err := url.Parse("http://localhost:8075")
		require.NoError(t, err)

		getClustersRequest := &model.GetClustersRequest{}
		getClustersRequest.ApplyToURL(u)

		require.Equal(t, "page=0&per_page=0", u.RawQuery)
	})

	t.Run("changes", func(t *testing.T) {
		u, err := url.Parse("http://localhost:8075")
		require.NoError(t, err)

		getClustersRequest := &model.GetClustersRequest{
			Page:           10,
			PerPage:        123,
			IncludeDeleted: true,
		}
		getClustersRequest.ApplyToURL(u)

		require.Equal(t, "include_deleted=true&page=10&per_page=123", u.RawQuery)
	})
}
