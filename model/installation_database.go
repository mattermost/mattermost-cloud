// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

// DefaultMattermostDatabaseUsername the default database username for an installation
const DefaultMattermostDatabaseUsername = "mmcloud"

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
	// multitenant database hosted via Amazon RDS that has pooled connections
	// through PGBouncer.
	InstallationDatabaseMultiTenantRDSPostgresPGBouncer = "aws-multitenant-rds-postgres-pgbouncer"
	// InstallationDatabasePerseus is a PostgreSQL multitenant database hosted
	// via Amazon RDS that has pooled connections through Perseus.
	InstallationDatabasePerseus = "perseus"
	// InstallationDatabaseExternal is a database that is created and managed
	// outside of the cloud provisioner. No provisioning or teardown is performed
	// on this database type. An AWS secret with connection strings and
	// credentials must be specified on installation creation when using this
	// database type.
	InstallationDatabaseExternal = "external"

	// DatabaseEngineTypeMySQL is a MySQL database.
	DatabaseEngineTypeMySQL = "mysql"
	// DatabaseEngineTypePostgres is a PostgreSQL database.
	DatabaseEngineTypePostgres = "postgres"
	// DatabaseEngineTypePostgresProxy is a PostgreSQL database that is
	// configured for proxied connections.
	DatabaseEngineTypePostgresProxy = "postgres-proxy"
	// DatabaseEngineTypePostgresProxyPerseus is a PostgreSQL database that is
	// configured for proxied connections from Perseus.
	DatabaseEngineTypePostgresProxyPerseus = "postgres-proxy-perseus"
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
	GetInstallation(id string, includeGroupConfig, includeGroupConfigOverrides bool) (*Installation, error)
	GetClusterInstallations(filter *ClusterInstallationFilter) ([]*ClusterInstallation, error)
	GetMultitenantDatabases(filter *MultitenantDatabaseFilter) ([]*MultitenantDatabase, error)
	GetMultitenantDatabase(multitenantdatabaseID string) (*MultitenantDatabase, error)
	GetMultitenantDatabaseForInstallationID(installationID string) (*MultitenantDatabase, error)
	GetInstallationsTotalDatabaseWeight(installationIDs []string) (float64, error)
	CreateMultitenantDatabase(multitenantDatabase *MultitenantDatabase) error
	UpdateMultitenantDatabase(multitenantDatabase *MultitenantDatabase) error
	LockMultitenantDatabase(multitenantdatabaseID, lockerID string) (bool, error)
	UnlockMultitenantDatabase(multitenantdatabaseID, lockerID string, force bool) (bool, error)
	LockMultitenantDatabases(ids []string, lockerID string) (bool, error)
	UnlockMultitenantDatabases(ids []string, lockerID string, force bool) (bool, error)
	GetLogicalDatabases(filter *LogicalDatabaseFilter) ([]*LogicalDatabase, error)
	GetLogicalDatabase(logicalDatabaseID string) (*LogicalDatabase, error)
	GetDatabaseSchemas(filter *DatabaseSchemaFilter) ([]*DatabaseSchema, error)
	GetDatabaseSchema(databaseSchemaID string) (*DatabaseSchema, error)
	GetSingleTenantDatabaseConfigForInstallation(installationID string) (*SingleTenantDatabaseConfig, error)
	GetProxyDatabaseResourcesForInstallation(installationID string) (*DatabaseResourceGrouping, error)
	GetOrCreateProxyDatabaseResourcesForInstallation(installationID, multitenantDatabaseID string) (*DatabaseResourceGrouping, error)
	DeleteInstallationProxyDatabaseResources(multitenantDatabase *MultitenantDatabase, databaseSchema *DatabaseSchema) error
	GetGroupDTOs(filter *GroupFilter) ([]*GroupDTO, error)
}

// ClusterUtilityDatabaseStoreInterface is the interface necessary for SQLStore
// functionality to update cluster utilities as needed.
type ClusterUtilityDatabaseStoreInterface interface {
	GetMultitenantDatabases(filter *MultitenantDatabaseFilter) ([]*MultitenantDatabase, error)
	GetLogicalDatabases(filter *LogicalDatabaseFilter) ([]*LogicalDatabase, error)
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

// IsSupportedDatabase returns true if the given database string is supported.
func IsSupportedDatabase(database string) bool {
	switch database {
	case InstallationDatabaseSingleTenantRDSMySQL:
	case InstallationDatabaseSingleTenantRDSPostgres:
	case InstallationDatabaseMultiTenantRDSMySQL:
	case InstallationDatabaseMultiTenantRDSPostgres:
	case InstallationDatabaseMultiTenantRDSPostgresPGBouncer:
	case InstallationDatabasePerseus:
	case InstallationDatabaseMysqlOperator:
	case InstallationDatabaseExternal:
	default:
		return false
	}

	return true
}

// InternalDatabase returns true if the installation's database is internal
// to the kubernetes cluster it is running on.
func (i *Installation) InternalDatabase() bool {
	return i.Database == InstallationDatabaseMysqlOperator
}

// IsSingleTenantRDS returns true if the given database is single tenant RDS db.
func IsSingleTenantRDS(database string) bool {
	switch database {
	case InstallationDatabaseSingleTenantRDSMySQL:
	case InstallationDatabaseSingleTenantRDSPostgres:
	default:
		return false
	}

	return true
}

// IsMultiTenantRDS returns true if the given database is multitenant RDS db.
func IsMultiTenantRDS(database string) bool {
	switch database {
	case InstallationDatabaseMultiTenantRDSMySQL:
	case InstallationDatabaseMultiTenantRDSPostgres:
	case InstallationDatabaseMultiTenantRDSPostgresPGBouncer:
	case InstallationDatabasePerseus:
	default:
		return false
	}

	return true
}
