// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	log "github.com/sirupsen/logrus"
)

type groupLockStore interface {
	LockGroup(groupID, lockerID string) (bool, error)
	UnlockGroup(groupID, lockerID string, force bool) (bool, error)
}

type groupLock struct {
	groupID  string
	lockerID string
	store    groupLockStore
	logger   log.FieldLogger
}

func newGroupLock(groupID, lockerID string, store groupLockStore, logger log.FieldLogger) *groupLock {
	return &groupLock{
		groupID:  groupID,
		lockerID: lockerID,
		store:    store,
		logger:   logger,
	}
}

func (l *groupLock) TryLock() bool {
	locked, err := l.store.LockGroup(l.groupID, l.lockerID)
	if err != nil {
		l.logger.WithError(err).Error("failed to lock group")
		return false
	}

	return locked
}

func (l *groupLock) Unlock() {
	unlocked, err := l.store.UnlockGroup(l.groupID, l.lockerID, false)
	if err != nil {
		l.logger.WithError(err).Error("failed to unlock group")
	} else if !unlocked {
		l.logger.Error("failed to release lock for group")
	}
}
