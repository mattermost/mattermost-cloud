// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	"errors"
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
)

type mockGetClusterForClusterInstallationStore struct {
	ClusterInstallation *model.ClusterInstallation
	Cluster             *model.Cluster

	ClusterInstallationStoreError error
	ClusterStoreError             error
}

func (s mockGetClusterForClusterInstallationStore) GetClusterInstallation(string) (*model.ClusterInstallation, error) {
	return s.ClusterInstallation, s.ClusterInstallationStoreError
}

func (s mockGetClusterForClusterInstallationStore) GetCluster(string) (*model.Cluster, error) {
	return s.Cluster, s.ClusterStoreError
}

func TestGetClusterForClusterInstallation(t *testing.T) {
	t.Run("cluster installation store error", func(t *testing.T) {
		mockStore := &mockGetClusterForClusterInstallationStore{
			ClusterInstallation:           nil,
			Cluster:                       nil,
			ClusterInstallationStoreError: errors.New("ci store error"),
			ClusterStoreError:             nil,
		}

		cluster, err := getClusterForClusterInstallation(mockStore, "")
		assert.Error(t, err)
		assert.Nil(t, cluster)
	})

	t.Run("missing cluster installation", func(t *testing.T) {
		mockStore := &mockGetClusterForClusterInstallationStore{nil, nil, nil, nil}

		cluster, err := getClusterForClusterInstallation(mockStore, "")
		assert.Error(t, err)
		assert.Nil(t, cluster)
	})

	t.Run("cluster store error", func(t *testing.T) {
		mockStore := &mockGetClusterForClusterInstallationStore{
			ClusterInstallation:           &model.ClusterInstallation{},
			Cluster:                       nil,
			ClusterInstallationStoreError: nil,
			ClusterStoreError:             errors.New("cluster store error"),
		}

		cluster, err := getClusterForClusterInstallation(mockStore, "")
		assert.Error(t, err)
		assert.Nil(t, cluster)
	})

	t.Run("missing cluster", func(t *testing.T) {
		mockStore := &mockGetClusterForClusterInstallationStore{
			ClusterInstallation:           &model.ClusterInstallation{},
			Cluster:                       nil,
			ClusterInstallationStoreError: nil,
			ClusterStoreError:             nil,
		}

		cluster, err := getClusterForClusterInstallation(mockStore, "")
		assert.Error(t, err)
		assert.Nil(t, cluster)
	})

	t.Run("success", func(t *testing.T) {
		mockStore := &mockGetClusterForClusterInstallationStore{
			ClusterInstallation:           &model.ClusterInstallation{},
			Cluster:                       &model.Cluster{},
			ClusterInstallationStoreError: nil,
			ClusterStoreError:             nil,
		}

		cluster, err := getClusterForClusterInstallation(mockStore, "")
		assert.NoError(t, err)
		assert.Equal(t, mockStore.Cluster, cluster)
	})

}
