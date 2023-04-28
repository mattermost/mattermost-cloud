// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package testutil

import (
	"errors"

	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

type noopDatabase struct{}

func NewNoopDatabase() model.Database {
	return &noopDatabase{}
}

// IsValid returns if the given external database configuration is valid.
func (d *noopDatabase) IsValid() error {
	return nil
}

// Provision logs that no further setup is needed for the precreated external
// database.
func (d *noopDatabase) Provision(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	return nil
}

// GenerateDatabaseSecret creates the k8s database spec and secret for
// accessing the external database.
func (d *noopDatabase) GenerateDatabaseSecret(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*corev1.Secret, error) {
	return nil, nil
}

// Teardown logs that no further actions are required for external database teardown.
func (d *noopDatabase) Teardown(store model.InstallationDatabaseStoreInterface, keepData bool, logger log.FieldLogger) error {
	return nil
}

// Snapshot is not supported for external databases.
func (d *noopDatabase) Snapshot(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	return errors.New("Snapshot is not supported for external databases")
}

// TeardownMigrated is not supported for external databases.
func (d *noopDatabase) TeardownMigrated(store model.InstallationDatabaseStoreInterface, migrationOp *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return errors.New("TeardownMigrated is not supported for external databases")
}

// MigrateOut is not supported for external databases.
func (d *noopDatabase) MigrateOut(store model.InstallationDatabaseStoreInterface, dbMigration *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return errors.New("MigrateOut is not supported for external databases")
}

// MigrateTo is not supported for external databases.
func (d *noopDatabase) MigrateTo(store model.InstallationDatabaseStoreInterface, dbMigration *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return errors.New("MigrateTo is not supported for external databases")
}

// RollbackMigration is not supported for external databases.
func (d *noopDatabase) RollbackMigration(store model.InstallationDatabaseStoreInterface, dbMigration *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return errors.New("RollbackMigration is not supported for external databases")
}

// RefreshResourceMetadata ensures various database resource's metadata are correct.
func (d *noopDatabase) RefreshResourceMetadata(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	return nil
}
