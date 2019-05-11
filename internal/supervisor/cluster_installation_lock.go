package supervisor

import (
	log "github.com/sirupsen/logrus"
)

type clusterInstallationLockStore interface {
	LockClusterInstallations(clusterInstallationID []string, lockerID string) (bool, error)
	UnlockClusterInstallations(clusterInstallationID []string, lockerID string, force bool) (bool, error)
}

type clusterInstallationLock struct {
	clusterInstallationIDs []string
	lockerID               string
	store                  clusterInstallationLockStore
	logger                 log.FieldLogger
}

func newClusterInstallationLock(clusterInstallationID, lockerID string, store clusterInstallationLockStore, logger log.FieldLogger) *clusterInstallationLock {
	return &clusterInstallationLock{
		clusterInstallationIDs: []string{clusterInstallationID},
		lockerID:               lockerID,
		store:                  store,
		logger:                 logger,
	}
}

func newClusterInstallationLocks(clusterInstallationIDs []string, lockerID string, store clusterInstallationLockStore, logger log.FieldLogger) *clusterInstallationLock {
	return &clusterInstallationLock{
		clusterInstallationIDs: clusterInstallationIDs,
		lockerID:               lockerID,
		store:                  store,
		logger:                 logger,
	}
}

func (l *clusterInstallationLock) TryLock() bool {
	locked, err := l.store.LockClusterInstallations(l.clusterInstallationIDs, l.lockerID)
	if err != nil {
		l.logger.WithError(err).Error("failed to lock cluster installations")
		return false
	}

	return locked
}

func (l *clusterInstallationLock) Unlock() {
	unlocked, err := l.store.UnlockClusterInstallations(l.clusterInstallationIDs, l.lockerID, false)
	if err != nil {
		l.logger.WithError(err).Error("failed to unlock cluster installations")
	} else if unlocked != true {
		l.logger.Error("failed to release lock for cluster installations")
	}
}
