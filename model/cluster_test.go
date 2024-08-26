// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"bytes"
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClusterClone(t *testing.T) {
	cluster := &Cluster{
		Provider:                "aws",
		Provisioner:             "kops",
		ProviderMetadataAWS:     &AWSMetadata{Zones: []string{"zone1"}},
		ProvisionerMetadataKops: &KopsMetadata{Version: "version1"},
		AllowInstallations:      false,
	}

	clone := cluster.Clone()
	require.Equal(t, cluster, clone)

	// Verify changing pointers in the clone doesn't affect the original.
	clone.ProviderMetadataAWS = &AWSMetadata{Zones: []string{"zone2"}}
	clone.ProvisionerMetadataKops = &KopsMetadata{Version: "version2"}
	require.NotEqual(t, cluster, clone)
}

func TestClusterVpcID(t *testing.T) {
	tests := []struct {
		name          string
		cluster       Cluster
		expectedVpcID string
	}{
		{
			"kops-vpc-id",
			Cluster{
				Provisioner:             ProvisionerKops,
				ProvisionerMetadataKops: &KopsMetadata{VPC: "kops-id-1"},
			},
			"kops-id-1",
		},
		{
			"eks-vpc-id",
			Cluster{
				Provisioner:            ProvisionerEKS,
				ProvisionerMetadataEKS: &EKSMetadata{VPC: "eks-id-1"},
			},
			"eks-id-1",
		},
		{
			"external-vpc-id",
			Cluster{
				Provisioner:                 ProvisionerExternal,
				ProvisionerMetadataExternal: &ExternalClusterMetadata{VPC: "external-id-1"},
			},
			"external-id-1",
		},
		{
			"invalid",
			Cluster{
				Provisioner: "invalid",
			},
			"",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedVpcID, test.cluster.VpcID())
		})
	}
}

func TestClusterHasAWSInfrastructure(t *testing.T) {
	tests := []struct {
		name             string
		cluster          Cluster
		expectedHasInfra bool
	}{
		{
			"kops",
			Cluster{
				Provider:    ProviderAWS,
				Provisioner: ProvisionerKops,
			},
			true,
		},
		{
			"eks",
			Cluster{
				Provider:    ProviderAWS,
				Provisioner: ProvisionerEKS,
			},
			true,
		},
		{
			"external-has-aws-infra",
			Cluster{
				Provider: ProviderExternal,
				ProviderMetadataExternal: &ExternalProviderMetadata{
					HasAWSInfrastructure: true,
				},
				Provisioner: ProvisionerExternal,
			},
			true,
		},
		{
			"external-no-aws-infra",
			Cluster{
				Provider: ProviderExternal,
				ProviderMetadataExternal: &ExternalProviderMetadata{
					HasAWSInfrastructure: false,
				},
				Provisioner: ProvisionerExternal,
			},
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedHasInfra, test.cluster.HasAWSInfrastructure())
		})
	}
}

func TestClusterApplyClusterUpdatePatch(t *testing.T) {
	tests := []struct {
		name            string
		cluster         Cluster
		patch           *UpdateClusterRequest
		expectedApply   bool
		expectedCluster Cluster
	}{
		{
			"empty patch",
			Cluster{},
			&UpdateClusterRequest{},
			false,
			Cluster{},
		},
		{
			"name only",
			Cluster{},
			&UpdateClusterRequest{Name: util.SToP("new1")},
			true,
			Cluster{Name: "new1"},
		},
		{
			"allow installations only",
			Cluster{},
			&UpdateClusterRequest{AllowInstallations: util.BToP(true)},
			true,
			Cluster{AllowInstallations: true},
		},
		{
			"all values",
			Cluster{},
			&UpdateClusterRequest{
				Name:               util.SToP("new2"),
				AllowInstallations: util.BToP(true),
			},
			true,
			Cluster{
				Name:               "new2",
				AllowInstallations: true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			applied := test.cluster.ApplyClusterUpdatePatch(test.patch)
			assert.Equal(t, test.expectedApply, applied)
			assert.Equal(t, test.expectedCluster, test.cluster)
		})
	}
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
			{ID: "id1", Provider: "aws"},
			{ID: "id2", Provider: "aws"},
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
