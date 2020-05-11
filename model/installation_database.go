package model

import (
	mmv1alpha1 "github.com/mattermost/mattermost-operator/pkg/apis/mattermost/v1alpha1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

const (
	// InstallationDatabaseMysqlOperator is a database hosted in kubernetes via the operator.
	InstallationDatabaseMysqlOperator = "mysql-operator"
	// InstallationDatabaseSingleTenantRDS is a database hosted via Amazon RDS.
	InstallationDatabaseSingleTenantRDS = "aws-singletenant-rds"
	// InstallationDatabaseMultiTenantRDS is a multitenant database hosted via Amazon RDS.
	InstallationDatabaseMultiTenantRDS = "aws-multitenant-rds"
)

// Database is the interface for managing Mattermost databases.
type Database interface {
	Provision(store InstallationDatabaseStoreInterface, logger log.FieldLogger) error
	Teardown(store InstallationDatabaseStoreInterface, keepData bool, logger log.FieldLogger) error
	Snapshot(store InstallationDatabaseStoreInterface, logger log.FieldLogger) error
	GenerateDatabaseSpecAndSecret(store InstallationDatabaseStoreInterface, logger log.FieldLogger) (*mmv1alpha1.Database, *corev1.Secret, error)
}

// InstallationDatabaseStoreInterface is the interface necessary for SQLStore
// functionality to correlate an installation to a cluster for database creation.
type InstallationDatabaseStoreInterface interface {
	GetClusterInstallations(filter *ClusterInstallationFilter) ([]*ClusterInstallation, error)
	GetDatabaseCluster(id string) (*DatabaseCluster, error)
	GetDatabaseClusters(filter *DatabaseClusterFilter) ([]*DatabaseCluster, error)
	CreateDatabaseCluster(databaseCluster *DatabaseCluster) error
	LockDatabaseCluster(databaseID, lockerID string) (bool, error)
	UnlockDatabaseCluster(databaseID, lockerID string, force bool) (bool, error)
	UpdateDatabaseCluster(databaseCluster *DatabaseCluster) error
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

// GenerateDatabaseSpecAndSecret creates the k8s database spec and secret for
// accessing the MySQL operator database.
func (d *MysqlOperatorDatabase) GenerateDatabaseSpecAndSecret(store InstallationDatabaseStoreInterface, logger log.FieldLogger) (*mmv1alpha1.Database, *corev1.Secret, error) {
	return nil, nil, nil
}

// InternalDatabase returns true if the installation's database is internal
// to the kubernetes cluster it is running on.
func (i *Installation) InternalDatabase() bool {
	return i.Database == InstallationDatabaseMysqlOperator
}

// IsSupportedDatabase returns true if the given database string is supported.
func IsSupportedDatabase(database string) bool {
	switch database {
	case InstallationDatabaseSingleTenantRDS:
	case InstallationDatabaseMultiTenantRDS:
	case InstallationDatabaseMysqlOperator:
	default:
		return false
	}

	return true
}
