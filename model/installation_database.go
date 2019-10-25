package model

import (
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	mmv1alpha1 "github.com/mattermost/mattermost-operator/pkg/apis/mattermost/v1alpha1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

const (
	// InstallationDatabaseMysqlOperator is a database hosted in kubernetes via the operator.
	InstallationDatabaseMysqlOperator = "mysql-operator"
	// InstallationDatabaseAwsRDS is a database hosted via Amazon RDS.
	InstallationDatabaseAwsRDS = "aws-rds"
)

// Database is the interface for managing Mattermost databases.
type Database interface {
	Provision(logger log.FieldLogger) error
	Teardown(keepData bool, logger log.FieldLogger) error
	GenerateDatabaseSpecAndSecret(logger log.FieldLogger) (*mmv1alpha1.Database, *corev1.Secret, error)
}

// MysqlOperatorDatabase is a database backed by the MySQL operator.
type MysqlOperatorDatabase struct{}

// NewMysqlOperatorDatabase returns a new MysqlOperatorDatabase interface.
func NewMysqlOperatorDatabase() *MysqlOperatorDatabase {
	return &MysqlOperatorDatabase{}
}

// Provision completes all the steps necessary to provision a MySQL operator database.
func (d *MysqlOperatorDatabase) Provision(logger log.FieldLogger) error {
	logger.Info("MySQL operator database requires no pre-provisioning; skipping...")

	return nil
}

// Teardown removes all MySQL operator resources for a given installation.
func (d *MysqlOperatorDatabase) Teardown(keepData bool, logger log.FieldLogger) error {
	logger.Info("MySQL operator database requires no teardown; skipping...")
	if keepData {
		logger.Warn("Database preservation was requested, but isn't currently possible with the MySQL operator")
	}

	return nil
}

// GenerateDatabaseSpecAndSecret creates the k8s database spec and secret for
// accessing the MySQL operator database.
func (d *MysqlOperatorDatabase) GenerateDatabaseSpecAndSecret(logger log.FieldLogger) (*mmv1alpha1.Database, *corev1.Secret, error) {
	return nil, nil, nil
}

// GetDatabase returns the Database interface that matches the installation.
func (i *Installation) GetDatabase() Database {
	switch i.Database {
	case InstallationDatabaseMysqlOperator:
		return NewMysqlOperatorDatabase()
	case InstallationDatabaseAwsRDS:
		return aws.NewRDSDatabase(i.ID)
	}

	// Warning: we should never get here as it would mean that we didn't match
	// our database type.
	return NewMysqlOperatorDatabase()
}

// InternalDatabase returns true if the installation's database is internal
// to the kubernetes cluster it is running on.
func (i *Installation) InternalDatabase() bool {
	return i.Database == InstallationDatabaseMysqlOperator
}

// IsSupportedDatabase returns true if the given database string is supported.
func IsSupportedDatabase(database string) bool {
	return database == InstallationDatabaseMysqlOperator || database == InstallationDatabaseAwsRDS
}
