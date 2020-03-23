package model

import (
	log "github.com/sirupsen/logrus"
)

const (
	// DatabaseMigrationStatusError ...
	DatabaseMigrationStatusError = "setup-error"
	// DatabaseMigrationStatusSetupIP ...
	DatabaseMigrationStatusSetupIP = "setup-in-progress"
	// DatabaseMigrationStatusSetupComplete ...
	DatabaseMigrationStatusSetupComplete = "setup-complete"
	// DatabaseMigrationStatusReplicationIP ...
	DatabaseMigrationStatusReplicationIP = "replication-in-progress"
	// DatabaseMigrationStatusReplicationComplete ...
	DatabaseMigrationStatusReplicationComplete = "replication-complete"
)

// CIMigrationDatabase is the interface for managing Mattermost databases migration process.
type CIMigrationDatabase interface {
	Setup(logger log.FieldLogger) (string, error)
	Replicate(logger log.FieldLogger) (string, error)
}
