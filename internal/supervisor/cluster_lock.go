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
	LockClusterScheduling(clusterID, lockerID string) (bool, error)
	UnlockClusterScheduling(clusterID, lockerID string, force bool) (bool, error)
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
	} else if !unlocked {
		l.logger.Error("failed to release lock for cluster")
	}
}

type clusterScheduleLock struct {
	clusterID string
	lockerID  string
	store     clusterLockStore
	logger    log.FieldLogger
}

func newClusterScheduleLock(clusterID, lockerID string, store clusterLockStore, logger log.FieldLogger) *clusterScheduleLock {
	return &clusterScheduleLock{
		clusterID: clusterID,
		lockerID:  lockerID,
		store:     store,
		logger:    logger,
	}
}

func (l *clusterScheduleLock) TryLock() bool {
	locked, err := l.store.LockClusterScheduling(l.clusterID, l.lockerID)
	if err != nil {
		l.logger.WithError(err).Error("failed to lock cluster scheduling")
		return false
	}

	return locked
}

func (l *clusterScheduleLock) Unlock() {
	unlocked, err := l.store.UnlockClusterScheduling(l.clusterID, l.lockerID, false)
	if err != nil {
		l.logger.WithError(err).Error("failed to unlock cluster scheduling")
	} else if !unlocked {
		l.logger.Error("failed to release lock for cluster scheduling")
	}
}
