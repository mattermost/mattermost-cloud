package model

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	// DatabaseMigrationStatusSetupIP indicates that database migration setup is still running.
	DatabaseMigrationStatusSetupIP = "setup-in-progress"
	// DatabaseMigrationStatusSetupComplete indicates that database migration setup is completed.
	DatabaseMigrationStatusSetupComplete = "setup-complete"
	// DatabaseMigrationStatusTeardownIP indicates that database migration teardown is still running.
	DatabaseMigrationStatusTeardownIP = "teardown-in-progress"
	// DatabaseMigrationStatusTeardownComplete indicates that database migration teardown is completed.
	DatabaseMigrationStatusTeardownComplete = "teardown-complete"
	// DatabaseMigrationStatusRestoreIP indicates that database migration restore is still running.
	DatabaseMigrationStatusRestoreIP = "restore-in-progress"
	// DatabaseMigrationStatusRestoreComplete indicates that database migration restore is completed.
	DatabaseMigrationStatusRestoreComplete = "restore-complete"
	// DatabaseMigrationStatusReplicationIP indicates that database migration replication process is still running.
	DatabaseMigrationStatusReplicationIP = "replication-in-progress"
	// DatabaseMigrationStatusReplicationComplete indicates that database migration process is completed.
	DatabaseMigrationStatusReplicationComplete = "replication-complete"

	// ErrUnsupportedDatabaseMigrationType describes an error that occurs when the provisioner attempts
	// to migrate a database type that is not supported.
	ErrUnsupportedDatabaseMigrationType = "attempted to migrate an unsupported database type"
)

// CIMigrationDatabase is the interface for managing Mattermost databases migration process.
type CIMigrationDatabase interface {
	Replicate(logger log.FieldLogger) (string, error)
	Restore(logger log.FieldLogger) (string, error)
	Status(logger log.FieldLogger) (string, error)
	Setup(logger log.FieldLogger) (string, error)
	Teardown(logger log.FieldLogger) (string, error)
}

// NotSupportedDatabaseMigration is supplied when systems required a database type that does not
// not support migration. All methods should return an error.
type NotSupportedDatabaseMigration struct{}

// Replicate returns not supported database error.
func (n *NotSupportedDatabaseMigration) Replicate(logger log.FieldLogger) (string, error) {
	return "", errors.New(ErrUnsupportedDatabaseMigrationType)
}

// Restore returns not supported database error.
func (n *NotSupportedDatabaseMigration) Restore(logger log.FieldLogger) (string, error) {
	return "", errors.New(ErrUnsupportedDatabaseMigrationType)
}

// Status returns not supported database error.
func (n *NotSupportedDatabaseMigration) Status(logger log.FieldLogger) (string, error) {
	return "", errors.New(ErrUnsupportedDatabaseMigrationType)
}

// Setup returns not supported database error.
func (n *NotSupportedDatabaseMigration) Setup(logger log.FieldLogger) (string, error) {
	return "", errors.New(ErrUnsupportedDatabaseMigrationType)
}

// Teardown returns not supported database error.
func (n *NotSupportedDatabaseMigration) Teardown(logger log.FieldLogger) (string, error) {
	return "", errors.New(ErrUnsupportedDatabaseMigrationType)
}
