// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// InstallationBackup contains information about installation's backup.
type InstallationBackup struct {
	ID             string
	InstallationID string
	// ClusterInstallationID is set when backup is scheduled.
	ClusterInstallationID string
	BackedUpDatabaseType  string
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
	Region     string
	URL        string
	Bucket     string
	PathPrefix string
	ObjectKey  string
}

// FullPath returns joined path of object in the file store.
func (dr S3DataResidence) FullPath() string {
	return filepath.Join(dr.PathPrefix, dr.ObjectKey)
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
	// InstallationBackupStateDeletionRequested is a backup marked for deletion.
	InstallationBackupStateDeletionRequested InstallationBackupState = "deletion-requested"
	// InstallationBackupStateDeleted is a deleted backup.
	InstallationBackupStateDeleted InstallationBackupState = "deleted"
	// InstallationBackupStateDeletionFailed is a backup which deletion failed.
	InstallationBackupStateDeletionFailed InstallationBackupState = "deletion-failed"
)

// AllInstallationBackupStatesPendingWork is a list of all backup states that
// the supervisor will attempt to transition towards stable on the next "tick".
var AllInstallationBackupStatesPendingWork = []InstallationBackupState{
	InstallationBackupStateBackupRequested,
	InstallationBackupStateBackupInProgress,
	InstallationBackupStateDeletionRequested,
}

// AllInstallationBackupsStatesRunning is a list of all backup states that are
// currently running.
var AllInstallationBackupsStatesRunning = []InstallationBackupState{
	InstallationBackupStateBackupRequested,
	InstallationBackupStateBackupInProgress,
}

// InstallationBackupFilter describes the parameters used to constrain a set of backup.
type InstallationBackupFilter struct {
	Paging
	IDs                   []string
	InstallationID        string
	ClusterInstallationID string
	States                []InstallationBackupState
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

// EnsureInstallationReadyForBackup ensures that installation can be backed up.
func EnsureInstallationReadyForBackup(installation *Installation) error {
	if installation.State != InstallationStateHibernating && installation.State != InstallationStateDBMigrationInProgress {
		return errors.Errorf("invalid installation state, only hibernated or migrating installations can be backed up, state is %q", installation.State)
	}

	return EnsureBackupRestoreCompatible(installation)
}

// EnsureBackupRestoreCompatible check if installation is compatible with backup-restore functionality.
func EnsureBackupRestoreCompatible(installation *Installation) error {
	var errs []string

	if installation.Database != InstallationDatabaseMultiTenantRDSPostgres &&
		installation.Database != InstallationDatabaseSingleTenantRDSPostgres {
		errs = append(errs, fmt.Sprintf("invalid installation database, backup-restore is supported only for Postgres database, the database type is %q", installation.Database))
	}

	if installation.Filestore == InstallationFilestoreMinioOperator {
		errs = append(errs, "invalid installation file store, backup-restore is not supported for installation using local Minio file store")
	}

	if len(errs) > 0 {
		return errors.Errorf("some installation settings are incompatible with backup-resotre: %s", strings.Join(errs, "; "))
	}

	return nil
}

// ValidTransitionState returns whether an installation backup can be transitioned into
// the new state or not based on its current state.
func (b *InstallationBackup) ValidTransitionState(newState InstallationBackupState) bool {
	validStates, found := validInstallationBackupTransitions[newState]
	if !found {
		return false
	}

	return stateIn(b.State, validStates)
}

var (
	validInstallationBackupTransitions = map[InstallationBackupState][]InstallationBackupState{
		InstallationBackupStateDeletionRequested: {
			InstallationBackupStateBackupRequested,
			InstallationBackupStateBackupInProgress,
			InstallationBackupStateBackupSucceeded,
			InstallationBackupStateBackupFailed,
			InstallationBackupStateDeletionRequested,
			InstallationBackupStateDeletionFailed,
		},
	}
)

func stateIn(state InstallationBackupState, states []InstallationBackupState) bool {
	for _, s := range states {
		if s == state {
			return true
		}
	}
	return false
}
