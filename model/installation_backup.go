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

// InstallationBackup contains information about installation's backup.
type InstallationBackup struct {
	ID             string
	InstallationID string
	// ClusterInstallationID is set when backup is scheduled.
	ClusterInstallationID string
	DataResidence         *S3DataResidence
	State                 InstallationBackupState
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

// InstallationBackupState represents the state of backup.
type InstallationBackupState string

const (
	// InstallationBackupStateBackupRequested is a requested backup that was not yet triggered.
	InstallationBackupStateBackupRequested InstallationBackupState = "backup-requested"
	// InstallationBackupStateBackupInProgress is a backup that is currently running.
	InstallationBackupStateBackupInProgress InstallationBackupState = "backup-in-progress"
	// InstallationBackupStateBackupSucceeded is a backup that have finished with success.
	InstallationBackupStateBackupSucceeded InstallationBackupState = "backup-succeeded"
	// InstallationBackupStateBackupFailed if a backup that have failed.
	InstallationBackupStateBackupFailed InstallationBackupState = "backup-failed"
)

// AllInstallationBackupStatesPendingWork is a list of all backup states that
// the supervisor will attempt to transition towards stable on the next "tick".
var AllInstallationBackupStatesPendingWork = []InstallationBackupState{
	InstallationBackupStateBackupRequested,
	InstallationBackupStateBackupInProgress,
}

// InstallationBackupFilter describes the parameters used to constrain a set of backup.
type InstallationBackupFilter struct {
	InstallationID        string
	ClusterInstallationID string
	State                 InstallationBackupState
	Page                  int
	PerPage               int
	IncludeDeleted        bool
}

// NewInstallationBackupFromReader will create a InstallationBackup from an
// io.Reader with JSON data.
func NewInstallationBackupFromReader(reader io.Reader) (*InstallationBackup, error) {
	var backupMetadata InstallationBackup
	err := json.NewDecoder(reader).Decode(&backupMetadata)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode backup")
	}

	return &backupMetadata, nil
}

// NewInstallationBackupsFromReader will create a slice of InstallationBackup from an
// io.Reader with JSON data.
func NewInstallationBackupsFromReader(reader io.Reader) ([]*InstallationBackup, error) {
	backupMetadata := []*InstallationBackup{}
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
