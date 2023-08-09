// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type clusterInstallationClaimStore interface {
	GetClusterInstallations(*model.ClusterInstallationFilter) ([]*model.ClusterInstallation, error)
	clusterInstallationLockStore
}

func claimClusterInstallation(store clusterInstallationClaimStore, installation *model.Installation, instanceID string, logger log.FieldLogger) (*model.ClusterInstallation, *clusterInstallationLock, error) {
	clusterInstallationFilter := &model.ClusterInstallationFilter{
		InstallationID: installation.ID,
		Paging:         model.AllPagesNotDeleted(),
	}
	clusterInstallations, err := store.GetClusterInstallations(clusterInstallationFilter)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get cluster installations")
	}

	if len(clusterInstallations) == 0 {
		return nil, nil, errors.Wrap(err, "expected at least one cluster installation for the installation but found none")
	}

	claimedCI := clusterInstallations[0]
	ciLock := newClusterInstallationLock(claimedCI.ID, instanceID, store, logger)
	if !ciLock.TryLock() {
		return nil, nil, errors.Errorf("failed to lock cluster installation %s", claimedCI.ID)
	}

	return claimedCI, ciLock, nil
}

type getAndLockInstallationStore interface {
	GetInstallation(installationID string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.Installation, error)
	installationLockStore
}

func getAndLockInstallation(store getAndLockInstallationStore, installationID, instanceID string, logger log.FieldLogger) (*model.Installation, *installationLock, error) {
	installation, err := store.GetInstallation(installationID, false, false)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get installation")
	}
	if installation == nil {
		return nil, nil, errors.New("could not find the installation")
	}

	lock := newInstallationLock(installation.ID, instanceID, store, logger)
	if !lock.TryLock() {
		return nil, nil, errors.Errorf("failed to lock installation %s", installationID)
	}
	return installation, lock, nil
}

type getClusterForClusterInstallationStore interface {
	GetClusterInstallation(string) (*model.ClusterInstallation, error)
	GetCluster(string) (*model.Cluster, error)
}

func getClusterForClusterInstallation(store getClusterForClusterInstallationStore, clusterInstallationID string) (*model.Cluster, error) {
	clusterInstallation, err := store.GetClusterInstallation(clusterInstallationID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster installations")
	}
	if clusterInstallation == nil {
		return nil, errors.Errorf("could not find cluster installation %s", clusterInstallationID)
	}

	cluster, err := store.GetCluster(clusterInstallation.ClusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster")
	}
	if cluster == nil {
		return nil, errors.Errorf("cluster not found for cluster installation %s", clusterInstallationID)
	}

	return cluster, nil
}
