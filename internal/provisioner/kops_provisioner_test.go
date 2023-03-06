// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCachedKopsClient(t *testing.T) {
	logger := testlib.MakeLogger(t)
	provisioner := NewKopsProvisioner(ProvisioningParams{}, nil, nil, logger)

	// Using &kops.Cmd{} here because kops.New() checks for the binary in your
	// PATH which isn't needed for the test and fails in CI/CD.
	provisioner.kopsCache["test"] = &kops.Cmd{}

	t.Run("get cached client", func(t *testing.T) {
		cachedClient, err := provisioner.getCachedKopsClient("test", logger)
		require.NoError(t, err)
		assert.NotNil(t, cachedClient)
	})

	t.Run("get cached kubecfg", func(t *testing.T) {
		config, err := provisioner.getCachedKopsClusterKubecfg("test", logger)
		require.NoError(t, err)
		assert.NotEmpty(t, config)
	})

	t.Run("invalidate cache", func(t *testing.T) {
		err := provisioner.invalidateCachedKopsClient("test", logger)
		require.NoError(t, err)
		require.Nil(t, provisioner.kopsCache["test"])
	})

	t.Run("invalidate missing cache", func(t *testing.T) {
		err := provisioner.invalidateCachedKopsClient("test1", logger)
		require.Error(t, err)
	})

	provisioner.kopsCache["test"] = &kops.Cmd{}

	t.Run("invalidate cache on error; error is nil", func(t *testing.T) {
		var cacheError error
		provisioner.invalidateCachedKopsClientOnError(cacheError, "test", logger)
		require.NotNil(t, provisioner.kopsCache["test"])
	})

	t.Run("invalidate cache on error; error is not nil", func(t *testing.T) {
		cacheError := errors.New("not nil")
		provisioner.invalidateCachedKopsClientOnError(cacheError, "test", logger)
		require.Nil(t, provisioner.kopsCache["test"])
	})
}

func TestGenerateCILicenseName(t *testing.T) {
	license := model.NewID()
	installation := &model.Installation{
		ID:      model.NewID(),
		License: license,
	}
	clusterInstallation := &model.ClusterInstallation{
		ID:             model.NewID(),
		InstallationID: installation.ID,
		Namespace:      installation.ID,
	}

	licenseName := generateCILicenseName(installation, clusterInstallation)
	assert.Contains(t, licenseName, makeClusterInstallationName(clusterInstallation))
	assert.Contains(t, licenseName, fmt.Sprintf("%x", sha256.Sum256([]byte(installation.License)))[0:6])
	assert.Contains(t, licenseName, "-license")
}

const (
	multipleKopsClustersOutput = `[
  {
    "kind": "Cluster",
    "apiVersion": "kops.k8s.io/v1alpha2",
    "metadata": {
      "name": "cluster-kops.k8s.local",
      "generation": 2,
      "creationTimestamp": "2021-09-09T07:51:20Z"
    },
    "spec": {
      "channel": "stable",
      "configBase": "s3://kops-state/cluster-kops.k8s.local",
      "cloudProvider": "aws",
      "kubernetesVersion": "1.20.9",
      "masterPublicName": "api.cluster-kops.k8s.local",
      "networkID": "vpc-cluster"
    }
  },
  {
    "kind": "Cluster",
    "apiVersion": "kops.k8s.io/v1alpha2",
    "metadata": {
      "name": "test-kops.k8s.local",
      "generation": 2,
      "creationTimestamp": "2020-10-26T16:27:47Z"
    },
    "spec": {
      "channel": "stable",
      "configBase": "s3://kops-state/test-kops.k8s.local",
      "cloudProvider": "aws",
      "kubernetesVersion": "1.17.12",
      "subnets": [
        {
          "name": "us-east-1a",
          "zone": "us-east-1a",
          "cidr": "10.10.10.0/23",
          "id": "subnet-abc",
          "type": "Private"
        }
      ],
      "masterPublicName": "api.test-kops.k8s.local",
      "networkCIDR": "10.240.112.0/20",
      "networkID": "vpc-0717d2e34d0e0b137"
    }
  }
]`
	singleClusterResponse = `{
    "kind": "Cluster",
    "apiVersion": "kops.k8s.io/v1alpha2",
    "metadata": {
      "name": "single-cluster-kops.k8s.local",
      "generation": 2,
      "creationTimestamp": "2021-09-09T07:51:20Z"
    },
    "spec": {
      "channel": "stable",
      "configBase": "s3://kops-state/cluster-kops.k8s.local",
      "cloudProvider": "aws",
      "kubernetesVersion": "1.20.9",
      "masterPublicName": "api.cluster-kops.k8s.local",
      "networkID": "vpc-cluster"
    }
  }`
)

func TestUnmarshalKopsListClustersResponse(t *testing.T) {
	t.Run("single cluster", func(t *testing.T) {
		clusters, err := unmarshalKopsListClustersResponse(singleClusterResponse)
		require.NoError(t, err)

		assert.Len(t, clusters, 1)
		assert.Equal(t, clusters[0].Metadata.Name, "single-cluster-kops.k8s.local")
	})

	t.Run("multiple clusters", func(t *testing.T) {
		clusters, err := unmarshalKopsListClustersResponse(multipleKopsClustersOutput)
		require.NoError(t, err)

		assert.Len(t, clusters, 2)
		assert.Equal(t, clusters[0].Metadata.Name, "cluster-kops.k8s.local")
		assert.Equal(t, clusters[1].Metadata.Name, "test-kops.k8s.local")
	})
}
