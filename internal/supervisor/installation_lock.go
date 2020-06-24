// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	log "github.com/sirupsen/logrus"
)

type installationLockStore interface {
	LockInstallation(installationID, lockerID string) (bool, error)
	UnlockInstallation(installationID, lockerID string, force bool) (bool, error)
}

type installationLock struct {
	installationID string
	lockerID       string
	store          installationLockStore
	logger         log.FieldLogger
}

func newInstallationLock(installationID, lockerID string, store installationLockStore, logger log.FieldLogger) *installationLock {
	return &installationLock{
		installationID: installationID,
		lockerID:       lockerID,
		store:          store,
		logger:         logger,
	}
}

func (l *installationLock) TryLock() bool {
	locked, err := l.store.LockInstallation(l.installationID, l.lockerID)
	if err != nil {
		l.logger.WithError(err).Error("failed to lock installation")
		return false
	}

	return locked
}

func (l *installationLock) Unlock() {
	unlocked, err := l.store.UnlockInstallation(l.installationID, l.lockerID, false)
	if err != nil {
		l.logger.WithError(err).Error("failed to unlock installation")
	} else if unlocked != true {
		l.logger.Error("failed to release lock for installation")
	}
}
