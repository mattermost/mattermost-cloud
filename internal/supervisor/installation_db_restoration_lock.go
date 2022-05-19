// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import log "github.com/sirupsen/logrus"

type installationDBRestorationLockStore interface {
	LockInstallationDBRestorationOperations(id []string, lockerID string) (bool, error)
	UnlockInstallationDBRestorationOperations(id []string, lockerID string, force bool) (bool, error)
}

type installationDBRestorationLock struct {
	ids      []string
	lockerID string
	store    installationDBRestorationLockStore
	logger   log.FieldLogger
}

func newInstallationDBRestorationLock(id, lockerID string, store installationDBRestorationLockStore, logger log.FieldLogger) *installationDBRestorationLock {
	return &installationDBRestorationLock{
		ids:      []string{id},
		lockerID: lockerID,
		store:    store,
		logger:   logger,
	}
}

func newInstallationDBRestorationLocks(ids []string, lockerID string, store installationDBRestorationLockStore, logger log.FieldLogger) *installationDBRestorationLock {
	return &installationDBRestorationLock{
		ids:      ids,
		lockerID: lockerID,
		store:    store,
		logger:   logger,
	}
}

func (l *installationDBRestorationLock) TryLock() bool {
	locked, err := l.store.LockInstallationDBRestorationOperations(l.ids, l.lockerID)
	if err != nil {
		l.logger.WithError(err).Error("failed to lock installationDBRestorations")
		return false
	}

	return locked
}

func (l *installationDBRestorationLock) Unlock() {
	unlocked, err := l.store.UnlockInstallationDBRestorationOperations(l.ids, l.lockerID, false)
	if err != nil {
		l.logger.WithError(err).Error("failed to unlock installationDBRestorations")
	} else if !unlocked {
		l.logger.Error("failed to release lock for installationDBRestorations")
	}
}
