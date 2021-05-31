// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

const (
	// InstallationDatabaseMysqlOperator is a database hosted in kubernetes via the operator.
	InstallationDatabaseMysqlOperator = "mysql-operator"
	// InstallationDatabaseSingleTenantRDSMySQL is a MySQL database hosted via
	// Amazon RDS.
	// TODO: update name value to aws-rds-mysql
	InstallationDatabaseSingleTenantRDSMySQL = "aws-rds"
	// InstallationDatabaseSingleTenantRDSPostgres is a PostgreSQL database hosted
	// via Amazon RDS.
	InstallationDatabaseSingleTenantRDSPostgres = "aws-rds-postgres"
	// InstallationDatabaseMultiTenantRDSMySQL is a MySQL multitenant database
	// hosted via Amazon RDS.
	// TODO: update name value to aws-multitenant-rds-mysql
	InstallationDatabaseMultiTenantRDSMySQL = "aws-multitenant-rds"
	// InstallationDatabaseMultiTenantRDSPostgres is a PostgreSQL multitenant
	// database hosted via Amazon RDS.
	InstallationDatabaseMultiTenantRDSPostgres = "aws-multitenant-rds-postgres"
	// InstallationDatabaseMultiTenantRDSPostgresPGBouncer is a PostgreSQL
	// multitenant database hosted via Amazon RDS that has pooled connections.
	InstallationDatabaseMultiTenantRDSPostgresPGBouncer = "aws-multitenant-rds-postgres-pgbouncer"

	// DatabaseEngineTypeMySQL is a MySQL database.
	DatabaseEngineTypeMySQL = "mysql"
	// DatabaseEngineTypePostgres is a PostgreSQL database.
	DatabaseEngineTypePostgres = "postgres"
	// DatabaseEngineTypePostgresProxy is a PostgreSQL database that is
	// configured for proxied connections.
	DatabaseEngineTypePostgresProxy = "postgres-proxy"
)

// Database is the interface for managing Mattermost databases.
type Database interface {
	Provision(store InstallationDatabaseStoreInterface, logger log.FieldLogger) error
	Teardown(store InstallationDatabaseStoreInterface, keepData bool, logger log.FieldLogger) error
	Snapshot(store InstallationDatabaseStoreInterface, logger log.FieldLogger) error
	GenerateDatabaseSecret(store InstallationDatabaseStoreInterface, logger log.FieldLogger) (*corev1.Secret, error)
	RefreshResourceMetadata(store InstallationDatabaseStoreInterface, logger log.FieldLogger) error
	MigrateOut(store InstallationDatabaseStoreInterface, dbMigration *InstallationDBMigrationOperation, logger log.FieldLogger) error
	MigrateTo(store InstallationDatabaseStoreInterface, dbMigration *InstallationDBMigrationOperation, logger log.FieldLogger) error
	TeardownMigrated(store InstallationDatabaseStoreInterface, migrationOp *InstallationDBMigrationOperation, logger log.FieldLogger) error
	RollbackMigration(store InstallationDatabaseStoreInterface, dbMigration *InstallationDBMigrationOperation, logger log.FieldLogger) error
}

// InstallationDatabaseStoreInterface is the interface necessary for SQLStore
// functionality to correlate an installation to a cluster for database creation.
// TODO(gsagula): Consider renaming this interface to InstallationDatabaseInterface. For reference,
// https://github.com/mattermost/mattermost-cloud/pull/209#discussion_r424597373
type InstallationDatabaseStoreInterface interface {
	GetClusterInstallations(filter *ClusterInstallationFilter) ([]*ClusterInstallation, error)
	GetMultitenantDatabase(multitenantdatabaseID string) (*MultitenantDatabase, error)
	GetMultitenantDatabases(filter *MultitenantDatabaseFilter) ([]*MultitenantDatabase, error)
	GetMultitenantDatabaseForInstallationID(installationID string) (*MultitenantDatabase, error)
	GetInstallationsTotalDatabaseWeight(installationIDs []string) (float64, error)
	CreateMultitenantDatabase(multitenantDatabase *MultitenantDatabase) error
	UpdateMultitenantDatabase(multitenantDatabase *MultitenantDatabase) error
	LockMultitenantDatabase(multitenantdatabaseID, lockerID string) (bool, error)
	UnlockMultitenantDatabase(multitenantdatabaseID, lockerID string, force bool) (bool, error)
	LockMultitenantDatabases(ids []string, lockerID string) (bool, error)
	UnlockMultitenantDatabases(ids []string, lockerID string, force bool) (bool, error)
	GetSingleTenantDatabaseConfigForInstallation(installationID string) (*SingleTenantDatabaseConfig, error)
}

// ClusterUtilityDatabaseStoreInterface is the interface necessary for SQLStore
// functionality to update cluster utilities as needed.
type ClusterUtilityDatabaseStoreInterface interface {
	GetMultitenantDatabases(filter *MultitenantDatabaseFilter) ([]*MultitenantDatabase, error)
}

// MysqlOperatorDatabase is a database backed by the MySQL operator.
type MysqlOperatorDatabase struct{}

// NewMysqlOperatorDatabase returns a new MysqlOperatorDatabase interface.
func NewMysqlOperatorDatabase() *MysqlOperatorDatabase {
	return &MysqlOperatorDatabase{}
}

// Provision completes all the steps necessary to provision a MySQL operator database.
func (d *MysqlOperatorDatabase) Provision(store InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	logger.Info("MySQL operator database requires no pre-provisioning; skipping...")

	return nil
}

// Snapshot is not supported by the operator and it should return an error.
func (d *MysqlOperatorDatabase) Snapshot(store InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	logger.Error("Snapshotting is not supported by the MySQL operator.")

	return errors.New("not implemented")
}

// Teardown removes all MySQL operator resources for a given installation.
func (d *MysqlOperatorDatabase) Teardown(store InstallationDatabaseStoreInterface, keepData bool, logger log.FieldLogger) error {
	logger.Info("MySQL operator database requires no teardown; skipping...")
	if keepData {
		logger.Warn("Database preservation was requested, but isn't currently possible with the MySQL operator")
	}

	return nil
}

// MigrateOut migrating out of MySQL Operator managed database is not supported.
func (d *MysqlOperatorDatabase) MigrateOut(store InstallationDatabaseStoreInterface, dbMigration *InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return errors.New("database migration is not supported for MySQL Operator")
}

// MigrateTo migration to MySQL Operator managed database is not supported.
func (d *MysqlOperatorDatabase) MigrateTo(store InstallationDatabaseStoreInterface, dbMigration *InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return errors.New("database migration is not supported for MySQL Operator")
}

// TeardownMigrated tearing down migrated databases is not supported for MySQL Operator managed database.
func (d *MysqlOperatorDatabase) TeardownMigrated(store InstallationDatabaseStoreInterface, migrationOp *InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return errors.New("tearing down migrated installations is not supported for MySQL Operator")
}

// RollbackMigration rolling back migration is not supported for MySQL Operator managed database.
func (d *MysqlOperatorDatabase) RollbackMigration(store InstallationDatabaseStoreInterface, dbMigration *InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return errors.New("rolling back db migration is not supported for MySQL Operator")
}

// GenerateDatabaseSecret creates the k8s database spec and secret for
// accessing the MySQL operator database.
func (d *MysqlOperatorDatabase) GenerateDatabaseSecret(store InstallationDatabaseStoreInterface, logger log.FieldLogger) (*corev1.Secret, error) {
	return nil, nil
}

// RefreshResourceMetadata ensures various operator database resource's metadata
// are correct.
func (d *MysqlOperatorDatabase) RefreshResourceMetadata(store InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	return nil
}

// InternalDatabase returns true if the installation's database is internal
// to the kubernetes cluster it is running on.
func (i *Installation) InternalDatabase() bool {
	return i.Database == InstallationDatabaseMysqlOperator
}

// IsSupportedDatabase returns true if the given database string is supported.
func IsSupportedDatabase(database string) bool {
	switch database {
	case InstallationDatabaseSingleTenantRDSMySQL:
	case InstallationDatabaseSingleTenantRDSPostgres:
	case InstallationDatabaseMultiTenantRDSMySQL:
	case InstallationDatabaseMultiTenantRDSPostgres:
	case InstallationDatabaseMultiTenantRDSPostgresPGBouncer:
	case InstallationDatabaseMysqlOperator:
	default:
		return false
	}

	return true
}

// IsSingleTenantRDS returns true if the given database is single tenant db.
func IsSingleTenantRDS(database string) bool {
	switch database {
	case InstallationDatabaseSingleTenantRDSMySQL:
	case InstallationDatabaseSingleTenantRDSPostgres:
	default:
		return false
	}

	return true
}
