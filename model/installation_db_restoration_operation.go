// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
)

// InstallationDBRestorationOperation contains information about installation's database restoration operation.
type InstallationDBRestorationOperation struct {
	ID             string
	InstallationID string
	BackupID       string
	RequestAt      int64
	State          InstallationDBRestorationState
	// TargetInstallationState is an installation State to which installation
	// will be transitioned when the restoration finishes successfully.
	TargetInstallationState string
	ClusterInstallationID   string
	CompleteAt              int64
	DeleteAt                int64
	LockAcquiredBy          *string
	LockAcquiredAt          int64
}

// InstallationDBRestorationState represents the state of db restoration operation.
type InstallationDBRestorationState string

const (
	// InstallationDBRestorationStateRequested is a requested installation db restoration that was not yet started.
	InstallationDBRestorationStateRequested InstallationDBRestorationState = "installation-db-restoration-requested"
	// InstallationDBRestorationStateInProgress is an installation db restoration that is currently running.
	InstallationDBRestorationStateInProgress InstallationDBRestorationState = "installation-db-restoration-in-progress"
	// InstallationDBRestorationStateFinalizing is an installation db restoration that is finalizing restoration.
	InstallationDBRestorationStateFinalizing InstallationDBRestorationState = "installation-db-restoration-finishing"
	// InstallationDBRestorationStateSucceeded is an installation db restoration that have finished with success.
	InstallationDBRestorationStateSucceeded InstallationDBRestorationState = "installation-db-restoration-succeeded"
	// InstallationDBRestorationStateFailing is an installation db restoration that is failing.
	InstallationDBRestorationStateFailing InstallationDBRestorationState = "installation-db-restoration-failing"
	// InstallationDBRestorationStateFailed is an installation db restoration that have failed.
	InstallationDBRestorationStateFailed InstallationDBRestorationState = "installation-db-restoration-failed"
	// InstallationDBRestorationStateInvalid is an installation db restoration that is invalid.
	InstallationDBRestorationStateInvalid InstallationDBRestorationState = "installation-db-restoration-invalid"
	// InstallationDBRestorationStateDeletionRequested is an installation db restoration scheduled for deletion.
	InstallationDBRestorationStateDeletionRequested InstallationDBRestorationState = "installation-db-restoration-deletion-requested"
	// InstallationDBRestorationStateDeleted is an installation db restoration that is deleted.
	InstallationDBRestorationStateDeleted InstallationDBRestorationState = "installation-db-restoration-deleted"
)

// AllInstallationDBRestorationStatesPendingWork is a list of all installation restoration operation
// states that the supervisor will attempt to transition towards succeeded on the next "tick".
var AllInstallationDBRestorationStatesPendingWork = []InstallationDBRestorationState{
	InstallationDBRestorationStateRequested,
	InstallationDBRestorationStateInProgress,
	InstallationDBRestorationStateFinalizing,
	InstallationDBRestorationStateFailing,
	InstallationDBRestorationStateDeletionRequested,
}

// InstallationDBRestorationFilter describes the parameters used to constrain a set of installation-db-restoration.
type InstallationDBRestorationFilter struct {
	Paging
	IDs                   []string
	InstallationID        string
	ClusterInstallationID string
	States                []InstallationDBRestorationState
}

// EnsureInstallationReadyForDBRestoration ensures that installation can be restored.
func EnsureInstallationReadyForDBRestoration(installation *Installation, backup *InstallationBackup) error {
	if installation.ID != backup.InstallationID {
		return errors.New("Backup belongs to different installation")
	}
	if backup.State != InstallationBackupStateBackupSucceeded {
		return errors.Errorf("Only backups in succeeded state can be restored, the state is %q", backup.State)
	}
	if backup.DeleteAt > 0 {
		return errors.New("Backup files are deleted")
	}

	if installation.State != InstallationStateHibernating && installation.State != InstallationStateDBMigrationInProgress {
		return errors.Errorf("invalid installation state, only hibernated installations can be restored, state is %q", installation.State)
	}

	return EnsureBackupRestoreCompatible(installation)
}

// DetermineAfterRestorationState returns installation state that should be set after successful restoration.
func DetermineAfterRestorationState(installation *Installation) (string, error) {
	switch installation.State {
	case InstallationStateHibernating:
		return InstallationStateHibernating, nil
	case InstallationStateDBMigrationInProgress:
		return InstallationStateDBMigrationInProgress, nil
	}
	return "", errors.Errorf("restoration is not supported for installation in state %s", installation.State)
}

// NewInstallationDBRestorationOperationFromReader will create a InstallationDBRestorationOperation from an
// io.Reader with JSON data.
func NewInstallationDBRestorationOperationFromReader(reader io.Reader) (*InstallationDBRestorationOperation, error) {
	var installationDBRestorationOperation InstallationDBRestorationOperation
	err := json.NewDecoder(reader).Decode(&installationDBRestorationOperation)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode InstallationDBRestorationOperation")
	}

	return &installationDBRestorationOperation, nil
}

// NewInstallationDBRestorationOperationsFromReader will create a slice of InstallationDBRestorationOperations from an
// io.Reader with JSON data.
func NewInstallationDBRestorationOperationsFromReader(reader io.Reader) ([]*InstallationDBRestorationOperation, error) {
	installationDBRestorationOperations := []*InstallationDBRestorationOperation{}
	err := json.NewDecoder(reader).Decode(&installationDBRestorationOperations)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode InstallationDBRestorationOperations")
	}

	return installationDBRestorationOperations, nil
}
