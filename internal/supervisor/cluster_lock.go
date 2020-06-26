// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	log "github.com/sirupsen/logrus"
)

type clusterLockStore interface {
	LockCluster(clusterID, lockerID string) (bool, error)
	UnlockCluster(clusterID, lockerID string, force bool) (bool, error)
}

type clusterLock struct {
	clusterID string
	lockerID  string
	store     clusterLockStore
	logger    log.FieldLogger
}

func newClusterLock(clusterID, lockerID string, store clusterLockStore, logger log.FieldLogger) *clusterLock {
	return &clusterLock{
		clusterID: clusterID,
		lockerID:  lockerID,
		store:     store,
		logger:    logger,
	}
}

func (l *clusterLock) TryLock() bool {
	locked, err := l.store.LockCluster(l.clusterID, l.lockerID)
	if err != nil {
		l.logger.WithError(err).Error("failed to lock cluster")
		return false
	}

	return locked
}

func (l *clusterLock) Unlock() {
	unlocked, err := l.store.UnlockCluster(l.clusterID, l.lockerID, false)
	if err != nil {
		l.logger.WithError(err).Error("failed to unlock cluster")
	} else if unlocked != true {
		l.logger.Error("failed to release lock for cluster")
	}
}
