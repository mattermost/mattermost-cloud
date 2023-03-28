// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

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
			Provider:           "aws",
			Version:            "latest",
			MasterInstanceType: "t3.medium",
			MasterCount:        1,
			NodeInstanceType:   "m5.large",
			NodeMinCount:       2,
			NodeMaxCount:       2,
			MaxPodsPerNode:     200,
			Zones:              []string{"us-east-1a"},
			Networking:         model.NetworkingCalico,
			Provisioner:        model.ProvisionerKops,
			DesiredUtilityVersions: map[string]*model.HelmUtilityVersion{
				"fluentbit":             {Chart: "0.20.1", ValuesPath: ""},
				"nginx":                 {Chart: "4.5.2", ValuesPath: ""},
				"nginx-internal":        {Chart: "4.5.2", ValuesPath: ""},
				"prometheus-operator":   {Chart: "40.5.0", ValuesPath: ""},
				"thanos":                {Chart: "11.5.4", ValuesPath: ""},
				"teleport-kube-agent":   {Chart: "7.3.26", ValuesPath: ""},
				"pgbouncer":             {Chart: "1.2.0", ValuesPath: ""},
				"promtail":              {Chart: "6.2.2", ValuesPath: ""},
				"rtcd":                  {Chart: "1.1.0", ValuesPath: ""},
				"node-problem-detector": {Chart: "2.3.2", ValuesPath: ""},
				"metrics-server":        {Chart: "3.8.3", ValuesPath: ""},
				"velero":                {Chart: "3.1.2", ValuesPath: ""},
				"cloudprober":           {Chart: "0.1.1", ValuesPath: ""},
			},
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
			`{"Provider": "azure", "Zones":["zone1", "zone2"]}`,
		)))
		require.EqualError(t, err, "create cluster request failed validation: unsupported provider azure")
		require.Nil(t, clusterRequest)
	})

	t.Run("partial request", func(t *testing.T) {
		clusterRequest, err := model.NewCreateClusterRequestFromReader(bytes.NewReader([]byte(
			`{"node-min-count": 1337}`,
		)))
		require.NoError(t, err)
		modifiedDefaultCreateClusterRequest := defaultCreateClusterRequest()
		modifiedDefaultCreateClusterRequest.NodeMinCount = 1337
		modifiedDefaultCreateClusterRequest.NodeMaxCount = 1337
		require.Equal(t, modifiedDefaultCreateClusterRequest, clusterRequest)
	})

	t.Run("full request", func(t *testing.T) {
		clusterRequest, err := model.NewCreateClusterRequestFromReader(bytes.NewReader([]byte(
			`{"Provider": "aws", "Version": "1.12.4", "Size": "SizeAlef1000", "Zones": ["zone1", "zone2"]}`,
		)))
		require.NoError(t, err)
		require.Equal(t, &model.CreateClusterRequest{
			Provider:           model.ProviderAWS,
			Version:            "1.12.4",
			MasterInstanceType: "t3.medium",
			MasterCount:        1,
			NodeInstanceType:   "m5.large",
			NodeMinCount:       2,
			NodeMaxCount:       2,
			MaxPodsPerNode:     200,
			Zones:              []string{"zone1", "zone2"},
			Networking:         model.NetworkingCalico,
			Provisioner:        model.ProvisionerKops,
			DesiredUtilityVersions: map[string]*model.HelmUtilityVersion{
				"fluentbit":             {Chart: "0.20.1", ValuesPath: ""},
				"nginx":                 {Chart: "4.5.2", ValuesPath: ""},
				"nginx-internal":        {Chart: "4.5.2", ValuesPath: ""},
				"prometheus-operator":   {Chart: "40.5.0", ValuesPath: ""},
				"thanos":                {Chart: "11.5.4", ValuesPath: ""},
				"teleport-kube-agent":   {Chart: "7.3.26", ValuesPath: ""},
				"pgbouncer":             {Chart: "1.2.0", ValuesPath: ""},
				"promtail":              {Chart: "6.2.2", ValuesPath: ""},
				"rtcd":                  {Chart: "1.1.0", ValuesPath: ""},
				"node-problem-detector": {Chart: "2.3.2", ValuesPath: ""},
				"metrics-server":        {Chart: "3.8.3", ValuesPath: ""},
				"velero":                {Chart: "3.1.2", ValuesPath: ""},
				"cloudprober":           {Chart: "0.1.1", ValuesPath: ""},
			},
		}, clusterRequest)
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
			Paging: model.Paging{
				Page:           10,
				PerPage:        123,
				IncludeDeleted: true,
			},
		}
		getClustersRequest.ApplyToURL(u)

		require.Equal(t, "include_deleted=true&page=10&per_page=123", u.RawQuery)
	})
}
