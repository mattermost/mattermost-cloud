// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
)

// InstallationDBMigrationOperation contains information about installation's database migration operation.
type InstallationDBMigrationOperation struct {
	ID             string
	InstallationID string
	RequestAt      int64
	State          InstallationDBMigrationOperationState
	// SourceDatabase is current Installation database.
	SourceDatabase string
	// DestinationDatabase is database type to which migration will be performed.
	DestinationDatabase string
	// For now only supported migration is from multi-tenant DB to multi-tenant DB.
	SourceMultiTenant                    *MultiTenantDBMigrationData `json:"SourceMultiTenant,omitempty"`
	DestinationMultiTenant               *MultiTenantDBMigrationData `json:"DestinationMultiTenant,omitempty"`
	BackupID                             string
	InstallationDBRestorationOperationID string
	CompleteAt                           int64
	DeleteAt                             int64
	LockAcquiredBy                       *string
	LockAcquiredAt                       int64
}

// MultiTenantDBMigrationData represents migration data for Multi-tenant database.
type MultiTenantDBMigrationData struct {
	DatabaseID string
}

// InstallationDBMigrationOperationState represents the state of db migration operation.
type InstallationDBMigrationOperationState string

const (
	// InstallationDBMigrationStateRequested is requested DB migration operation.
	InstallationDBMigrationStateRequested InstallationDBMigrationOperationState = "installation-db-migration-requested"
	// InstallationDBMigrationStateBackupInProgress is DB migration operation waiting for backup to complete.
	InstallationDBMigrationStateBackupInProgress InstallationDBMigrationOperationState = "installation-db-migration-installation-backup-in-progress"
	// InstallationDBMigrationStateDatabaseSwitch is DB migration operation that is switching to new database.
	InstallationDBMigrationStateDatabaseSwitch InstallationDBMigrationOperationState = "installation-db-migration-database switch"
	// InstallationDBMigrationStateRefreshSecrets is DB migration operation that is refreshing secrets.
	InstallationDBMigrationStateRefreshSecrets InstallationDBMigrationOperationState = "installation-db-migration-refresh-secrets"
	// InstallationDBMigrationStateTriggerRestoration is DB migration operation that is triggering database restoration.
	InstallationDBMigrationStateTriggerRestoration InstallationDBMigrationOperationState = "installation-db-migration-trigger-restoration"
	// InstallationDBMigrationStateRestorationInProgress is DB migration operation that is waiting for restoration to complete.
	InstallationDBMigrationStateRestorationInProgress InstallationDBMigrationOperationState = "installation-db-migration-restoration-in-progress"
	// InstallationDBMigrationStateUpdatingInstallationConfig is DB migration operation that is updating Installation configuration.
	InstallationDBMigrationStateUpdatingInstallationConfig InstallationDBMigrationOperationState = "installation-db-migration-updating-installation-config"
	// InstallationDBMigrationStateFinalizing is DB migration operation that is finalizing the migration.
	InstallationDBMigrationStateFinalizing InstallationDBMigrationOperationState = "installation-db-migration-finalizing"
	// InstallationDBMigrationStateFailing is DB migration operation that is failing.
	InstallationDBMigrationStateFailing InstallationDBMigrationOperationState = "installation-db-migration-failing"
	// InstallationDBMigrationStateSucceeded is DB migration operation that finished with success.
	InstallationDBMigrationStateSucceeded InstallationDBMigrationOperationState = "installation-db-migration-succeeded"
	// InstallationDBMigrationStateFailed is DB migration operation that failed.
	InstallationDBMigrationStateFailed InstallationDBMigrationOperationState = "installation-db-migration-failed"
)

// AllInstallationDBMigrationOperationsStatesPendingWork is a list of all db migration operations states
// that the supervisor will attempt to transition towards stable on the next "tick".
var AllInstallationDBMigrationOperationsStatesPendingWork = []InstallationDBMigrationOperationState{
	InstallationDBMigrationStateRequested,
	InstallationDBMigrationStateBackupInProgress,
	InstallationDBMigrationStateDatabaseSwitch,
	InstallationDBMigrationStateRefreshSecrets,
	InstallationDBMigrationStateTriggerRestoration,
	InstallationDBMigrationStateRestorationInProgress,
	InstallationDBMigrationStateUpdatingInstallationConfig,
	InstallationDBMigrationStateFinalizing,
	InstallationDBMigrationStateFailing,
}

// InstallationDBMigrationFilter describes the parameters used to constrain a set of installation db migration operations.
type InstallationDBMigrationFilter struct {
	Paging
	IDs            []string
	InstallationID string
	States         []InstallationDBMigrationOperationState
}

// NewDBMigrationOperationFromReader will create a InstallationDBMigrationOperation from an
// io.Reader with JSON data.
func NewDBMigrationOperationFromReader(reader io.Reader) (*InstallationDBMigrationOperation, error) {
	var dBMigrationOperation InstallationDBMigrationOperation
	err := json.NewDecoder(reader).Decode(&dBMigrationOperation)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode InstallationDBMigrationOperation")
	}

	return &dBMigrationOperation, nil
}

// NewDBMigrationOperationsFromReader will create a slice of DBMigrationOperations from an
// io.Reader with JSON data.
func NewDBMigrationOperationsFromReader(reader io.Reader) ([]*InstallationDBMigrationOperation, error) {
	dBMigrationOperations := []*InstallationDBMigrationOperation{}
	err := json.NewDecoder(reader).Decode(&dBMigrationOperations)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode DBMigrationOperations")
	}

	return dBMigrationOperations, nil
}
