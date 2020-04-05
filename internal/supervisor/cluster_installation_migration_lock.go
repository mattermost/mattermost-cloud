package supervisor

import (
	log "github.com/sirupsen/logrus"
)

type clusterInstallationMigrationLockStore interface {
	LockClusterInstallationMigration(migrationID, lockerID string) (bool, error)
	UnlockClusterInstallationMigration(migrationID, lockerID string, force bool) (bool, error)
}

type clusterInstallationMigrationLock struct {
	migrationID string
	lockerID    string
	store       clusterInstallationMigrationLockStore
	logger      log.FieldLogger
}

func newClusterInstallationMigrationLock(migrationID, lockerID string, store clusterInstallationMigrationLockStore, logger log.FieldLogger) *clusterInstallationMigrationLock {
	return &clusterInstallationMigrationLock{
		migrationID: migrationID,
		lockerID:    lockerID,
		store:       store,
		logger:      logger,
	}
}

func (l *clusterInstallationMigrationLock) TryLock() bool {
	locked, err := l.store.LockClusterInstallationMigration(l.migrationID, l.lockerID)
	if err != nil {
		l.logger.WithError(err).Error("failed to lock migration")
		return false
	}

	return locked
}

func (l *clusterInstallationMigrationLock) Unlock() {
	unlocked, err := l.store.UnlockClusterInstallationMigration(l.migrationID, l.lockerID, false)
	if err != nil {
		l.logger.WithError(err).Error("failed to unlock cluster installation migration")
	} else if unlocked != true {
		l.logger.Error("failed to release cluster installation migration locker")
	}
}
