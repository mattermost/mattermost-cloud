// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	log "github.com/sirupsen/logrus"
)

type backupLockStore interface {
	LockInstallationBackup(backupID, lockerID string) (bool, error)
	UnlockInstallationBackup(backupID, lockerID string, force bool) (bool, error)
}

type backupLock struct {
	backupID string
	lockerID string
	store    backupLockStore
	logger   log.FieldLogger
}

func newBackupLock(backupID, lockerID string, store backupLockStore, logger log.FieldLogger) *backupLock {
	return &backupLock{
		backupID: backupID,
		lockerID: lockerID,
		store:    store,
		logger:   logger,
	}
}

func (l *backupLock) TryLock() bool {
	locked, err := l.store.LockInstallationBackup(l.backupID, l.lockerID)
	if err != nil {
		l.logger.WithError(err).Error("failed to lock backup")
		return false
	}

	return locked
}

func (l *backupLock) Unlock() {
	unlocked, err := l.store.UnlockInstallationBackup(l.backupID, l.lockerID, false)
	if err != nil {
		l.logger.WithError(err).Error("failed to unlock backup")
	} else if unlocked != true {
		l.logger.Error("failed to release lock for backup")
	}
}
