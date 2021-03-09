// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
)

// BackupMetadata contains information about installation's backup.
type BackupMetadata struct {
	ID             string
	InstallationID string
	// ClusterInstallationID is set when backup is scheduled.
	ClusterInstallationID string
	DataResidence         *S3DataResidence
	State                 BackupState
	RequestAt             int64
	// StartAt is a start time of job that successfully completed backup.
	StartAt         int64
	DeleteAt        int64
	APISecurityLock bool
	LockAcquiredBy  *string
	LockAcquiredAt  int64
}

// S3DataResidence contains information about backup location.
type S3DataResidence struct {
	Region    string
	URL       string
	Bucket    string
	ObjectKey string
}

// BackupState represents the state of backup.
type BackupState string

const (
	// BackupStateBackupRequested is a requested backup that was not yet triggered.
	BackupStateBackupRequested BackupState = "backup-requested"
	// BackupStateBackupInProgress is a backup that is currently running.
	BackupStateBackupInProgress BackupState = "backup-in-progress"
	// BackupStateBackupSucceeded is a backup that have finished with success.
	BackupStateBackupSucceeded BackupState = "backup-succeeded"
	// BackupStateBackupFailed if a backup that have failed.
	BackupStateBackupFailed BackupState = "backup-failed"
)

// AllBackupMetadataStatesPendingWork is a list of all backup metadata states that
// the supervisor will attempt to transition towards stable on the next "tick".
var AllBackupMetadataStatesPendingWork = []BackupState{
	BackupStateBackupRequested,
	BackupStateBackupInProgress,
}

// BackupMetadataFilter describes the parameters used to constrain a set of backup metadata.
type BackupMetadataFilter struct {
	InstallationID        string
	ClusterInstallationID string
	State                 BackupState
	Page                  int
	PerPage               int
	IncludeDeleted        bool
}

// NewBackupMetadataFromReader will create a BackupMetadata from an
// io.Reader with JSON data.
func NewBackupMetadataFromReader(reader io.Reader) (*BackupMetadata, error) {
	var backupMetadata BackupMetadata
	err := json.NewDecoder(reader).Decode(&backupMetadata)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode backup metadata")
	}

	return &backupMetadata, nil
}

// NewBackupsMetadataFromReader will create a slice of BackupMetadata from an
// io.Reader with JSON data.
func NewBackupsMetadataFromReader(reader io.Reader) ([]*BackupMetadata, error) {
	backupMetadata := []*BackupMetadata{}
	err := json.NewDecoder(reader).Decode(&backupMetadata)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode backups metadata")
	}

	return backupMetadata, nil
}

// EnsureBackupCompatible ensures that installation can be backed up.
func EnsureBackupCompatible(installation *Installation) error {
	var errs []string

	if installation.State != InstallationStateHibernating {
		errs = append(errs, fmt.Sprintf("invalid installation state, only hibernated installations can be backed up, state is %q", installation.State))
	}

	if installation.Database != InstallationDatabaseMultiTenantRDSPostgres &&
		installation.Database != InstallationDatabaseSingleTenantRDSPostgres {
		errs = append(errs, fmt.Sprintf("invalid installation database, backup is supported only for Postgres database, the database type is %q", installation.Database))
	}

	if installation.Filestore == InstallationFilestoreMinioOperator {
		errs = append(errs, "invalid installation file store, cannot backup database for installation using local Minio file store")
	}

	if len(errs) > 0 {
		return errors.Errorf("some settings are incompatible with backup: [%s]", strings.Join(errs, "; "))
	}

	return nil
}
