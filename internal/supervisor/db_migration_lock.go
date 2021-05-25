// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import log "github.com/sirupsen/logrus"

type installationDBMigrationOperationLockStore interface {
	LockInstallationDBMigrationOperations(id []string, lockerID string) (bool, error)
	UnlockInstallationDBMigrationOperations(id []string, lockerID string, force bool) (bool, error)
}

type installationDBMigrationOperationLock struct {
	ids      []string
	lockerID string
	store    installationDBMigrationOperationLockStore
	logger   log.FieldLogger
}

func newInstallationDBMigrationOperationLock(id, lockerID string, store installationDBMigrationOperationLockStore, logger log.FieldLogger) *installationDBMigrationOperationLock {
	return &installationDBMigrationOperationLock{
		ids:      []string{id},
		lockerID: lockerID,
		store:    store,
		logger:   logger,
	}
}

func newInstallationDBMigrationOperationLocks(ids []string, lockerID string, store installationDBMigrationOperationLockStore, logger log.FieldLogger) *installationDBMigrationOperationLock {
	return &installationDBMigrationOperationLock{
		ids:      ids,
		lockerID: lockerID,
		store:    store,
		logger:   logger,
	}
}

func (l *installationDBMigrationOperationLock) TryLock() bool {
	locked, err := l.store.LockInstallationDBMigrationOperations(l.ids, l.lockerID)
	if err != nil {
		l.logger.WithError(err).Error("failed to lock installationDBMigrationOperations")
		return false
	}

	return locked
}

func (l *installationDBMigrationOperationLock) Unlock() {
	unlocked, err := l.store.UnlockInstallationDBMigrationOperations(l.ids, l.lockerID, false)
	if err != nil {
		l.logger.WithError(err).Error("failed to unlock installationDBMigrationOperations")
	} else if unlocked != true {
		l.logger.Error("failed to release lock for installationDBMigrationOperations")
	}
}
