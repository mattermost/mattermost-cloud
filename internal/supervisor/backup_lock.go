// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	log "github.com/sirupsen/logrus"
)

type installationBackupLockStore interface {
	LockInstallationBackups(backupsID []string, lockerID string) (bool, error)
	UnlockInstallationBackups(backupsID []string, lockerID string, force bool) (bool, error)
}

type backupLock struct {
	backupsID []string
	lockerID  string
	store     installationBackupLockStore
	logger    log.FieldLogger
}

func newBackupLock(backupID, lockerID string, store installationBackupLockStore, logger log.FieldLogger) *backupLock {
	return &backupLock{
		backupsID: []string{backupID},
		lockerID:  lockerID,
		store:     store,
		logger:    logger,
	}
}

func newBackupsLock(backupsID []string, lockerID string, store installationBackupLockStore, logger log.FieldLogger) *backupLock {
	return &backupLock{
		backupsID: backupsID,
		lockerID:  lockerID,
		store:     store,
		logger:    logger,
	}
}

func (l *backupLock) TryLock() bool {
	locked, err := l.store.LockInstallationBackups(l.backupsID, l.lockerID)
	if err != nil {
		l.logger.WithError(err).Error("failed to lock backup")
		return false
	}

	return locked
}

func (l *backupLock) Unlock() {
	unlocked, err := l.store.UnlockInstallationBackups(l.backupsID, l.lockerID, false)
	if err != nil {
		l.logger.WithError(err).Error("failed to unlock backup")
	} else if unlocked != true {
		l.logger.Error("failed to release lock for backup")
	}
}
