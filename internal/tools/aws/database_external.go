// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

// ExternalDatabase is a database that is created and managed outside of the
// cloud provisioner.
type ExternalDatabase struct {
	installationID string
	client         *Client
}

// NewExternalDatabase returns a new instance of ExternalDatabase that
// implements database interface.
func NewExternalDatabase(installationID string, client *Client) *ExternalDatabase {
	return &ExternalDatabase{
		installationID: installationID,
		client:         client,
	}
}

// IsValid returns if the given external database configuration is valid.
func (d *ExternalDatabase) IsValid() error {
	if len(d.installationID) == 0 {
		return errors.New("installation ID is not set")
	}

	return nil
}

// Provision logs that no further setup is needed for the precreated external
// database.
func (d *ExternalDatabase) Provision(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	logger = logger.WithField("external-database", ExternalDatabaseName(d.installationID))
	logger.Info("External database requires no pre-provisioning; skipping...")

	return nil
}

// GenerateDatabaseSecret creates the k8s database spec and secret for
// accessing the external database.
func (d *ExternalDatabase) GenerateDatabaseSecret(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*corev1.Secret, error) {
	err := d.IsValid()
	if err != nil {
		return nil, errors.Wrap(err, "external database configuration is invalid")
	}

	externalDatabaseName := ExternalDatabaseName(d.installationID)

	logger = logger.WithField("external-database", externalDatabaseName)

	installation, err := store.GetInstallation(d.installationID, false, false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get installation")
	}

	logger.Debugf("Using AWS secret %s for external database connections", installation.ExternalDatabaseConfig.SecretName)

	result, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: &installation.ExternalDatabaseConfig.SecretName,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get secret value for database")
	}

	externalSecret, err := extractExternalDatabaseSecret(*result.SecretString)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal secret payload")
	}

	secret := externalSecret.ToInstallationDBSecret(externalDatabaseName)

	logger.Debug("External database configuration generated for cluster installation")

	return secret.ToK8sSecret(false), nil
}

// Teardown logs that no further actions are required for external database teardown.
func (d *ExternalDatabase) Teardown(store model.InstallationDatabaseStoreInterface, keepData bool, logger log.FieldLogger) error {
	logger = logger.WithField("external-database", ExternalDatabaseName(d.installationID))
	logger.Info("External database requires no teardown; skipping...")
	if keepData {
		logger.Warn("Database preservation was requested, but isn't currently possible with external databases")
	}

	return nil
}

// Snapshot is not supported for external databases.
func (d *ExternalDatabase) Snapshot(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	return errors.New("Snapshot is not supported for external databases")
}

// TeardownMigrated is not supported for external databases.
func (d *ExternalDatabase) TeardownMigrated(store model.InstallationDatabaseStoreInterface, migrationOp *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return errors.New("TeardownMigrated is not supported for external databases")
}

// MigrateOut is not supported for external databases.
func (d *ExternalDatabase) MigrateOut(store model.InstallationDatabaseStoreInterface, dbMigration *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return errors.New("MigrateOut is not supported for external databases")
}

// MigrateTo is not supported for external databases.
func (d *ExternalDatabase) MigrateTo(store model.InstallationDatabaseStoreInterface, dbMigration *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return errors.New("MigrateTo is not supported for external databases")
}

// RollbackMigration is not supported for external databases.
func (d *ExternalDatabase) RollbackMigration(store model.InstallationDatabaseStoreInterface, dbMigration *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return errors.New("RollbackMigration is not supported for external databases")
}

// RefreshResourceMetadata ensures various database resource's metadata are correct.
func (d *ExternalDatabase) RefreshResourceMetadata(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	return nil
}

type externalDatabaseSecret struct {
	DataSource         string `json:"DataSource"`
	DataSourceReplicas string `json:"DataSourceReplicas"`
	ConnectionCheckURL string `json:"ConnectionCheckURL"`
}

// Validate performs a basic sanity check on the exteranl database secret.
func (s *externalDatabaseSecret) Validate() error {
	if len(s.DataSource) == 0 {
		return errors.New("DataSource value is empty")
	}
	if len(s.DataSourceReplicas) == 0 {
		return errors.New("DataSourceReplicas value is empty")
	}

	return nil
}

// ToInstallationDBSecret converts an externalDatabaseSecret to a k8s secret.
func (s *externalDatabaseSecret) ToInstallationDBSecret(name string) InstallationDBSecret {
	return InstallationDBSecret{
		InstallationSecretName: name,
		ConnectionString:       s.DataSource,
		ReadReplicasURL:        s.DataSourceReplicas,
		DBCheckURL:             s.ConnectionCheckURL,
	}
}

func extractExternalDatabaseSecret(payload string) (*externalDatabaseSecret, error) {
	var secret externalDatabaseSecret
	err := json.Unmarshal([]byte(payload), &secret)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal secrets manager payload")
	}

	err = secret.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "extracted external database secret failed validation")
	}

	return &secret, nil
}

// SecretsManagerValidateExternalDatabaseSecret pulls down the secret with the
// provided name and validates it as an external database secret.
func (a *Client) SecretsManagerValidateExternalDatabaseSecret(name string) error {
	result, err := a.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: &name,
	})
	if err != nil {
		return errors.Wrap(err, "failed to get secret value for database")
	}

	_, err = extractExternalDatabaseSecret(*result.SecretString)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal secret payload")
	}

	return nil
}
