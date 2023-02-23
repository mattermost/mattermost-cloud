// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	"time"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/webhook"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// installationBackupStore abstracts the database operations required to query installations backup.
type installationBackupStore interface {
	GetUnlockedInstallationBackupPendingWork() ([]*model.InstallationBackup, error)
	GetInstallationBackup(id string) (*model.InstallationBackup, error)
	UpdateInstallationBackupState(backupMeta *model.InstallationBackup) error
	UpdateInstallationBackupSchedulingData(backupMeta *model.InstallationBackup) error
	UpdateInstallationBackupStartTime(backupMeta *model.InstallationBackup) error
	DeleteInstallationBackup(id string) error
	installationBackupLockStore

	GetInstallation(installationID string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.Installation, error)
	installationLockStore

	GetClusterInstallations(*model.ClusterInstallationFilter) ([]*model.ClusterInstallation, error)
	GetClusterInstallation(clusterInstallationID string) (*model.ClusterInstallation, error)
	clusterInstallationLockStore

	GetCluster(id string) (*model.Cluster, error)

	GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error)
}

// BackupSupervisor finds backup pending work and effects the required changes.
//
// The degree of parallelism is controlled by a weighted semaphore, intended to be shared with
// other clients needing to coordinate background jobs.
type BackupSupervisor struct {
	store      installationBackupStore
	aws        aws.AWS
	instanceID string
	logger     log.FieldLogger

	provisioner BackupProvisioner
}

// NewBackupSupervisor creates a new BackupSupervisor.
func NewBackupSupervisor(
	store installationBackupStore,
	provisioner BackupProvisioner,
	aws aws.AWS,
	instanceID string,
	logger log.FieldLogger) *BackupSupervisor {
	return &BackupSupervisor{
		store:       store,
		provisioner: provisioner,
		aws:         aws,
		instanceID:  instanceID,
		logger:      logger,
	}
}

// ErrJobBackoffLimitReached indicates that job failed all possible attempts and there is no reason for retrying.
var ErrJobBackoffLimitReached = errors.New("job reached backoff limit")

// Shutdown performs graceful shutdown tasks for the backup supervisor.
func (s *BackupSupervisor) Shutdown() {
	s.logger.Debug("Shutting down backup supervisor")
}

// Do looks for work to be done on any pending backups and attempts to schedule the required work.
func (s *BackupSupervisor) Do() error {
	installations, err := s.store.GetUnlockedInstallationBackupPendingWork()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to query for backup pending work")
		return nil
	}

	for _, installation := range installations {
		s.Supervise(installation)
	}

	return nil
}

// Supervise schedules the required work on the given backup.
func (s *BackupSupervisor) Supervise(backup *model.InstallationBackup) {
	logger := s.logger.WithFields(log.Fields{
		"backup": backup.ID,
	})

	lock := newBackupLock(backup.ID, s.instanceID, s.store, logger)
	if !lock.TryLock() {
		return
	}
	defer lock.Unlock()

	// Before working on the backup, it is crucial that we ensure that it
	// was not updated to a new state by another provisioning server.
	originalState := backup.State
	backup, err := s.store.GetInstallationBackup(backup.ID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get refreshed backup")
		return
	}
	if backup.State != originalState {
		logger.WithField("oldBackupState", originalState).
			WithField("newBackupState", backup.State).
			Warn("Another provisioner has worked on this backup; skipping...")
		return
	}

	logger.Debugf("Supervising backup in state %s", backup.State)

	newState := s.transitionBackup(backup, s.instanceID, logger)

	backup, err = s.store.GetInstallationBackup(backup.ID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get backup and thus persist state %s", newState)
		return
	}

	if backup.State == newState {
		return
	}

	oldState := backup.State
	backup.State = newState

	err = s.store.UpdateInstallationBackupState(backup)
	if err != nil {
		logger.WithError(err).Errorf("Failed to set backup state to %s", newState)
		return
	}

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeInstallationBackup,
		ID:        backup.ID,
		NewState:  string(backup.State),
		OldState:  string(oldState),
		Timestamp: time.Now().UnixNano(),
		ExtraData: map[string]string{"Environment": s.aws.GetCloudEnvironmentName()},
	}
	err = webhook.SendToAllWebhooks(s.store, webhookPayload, logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		logger.WithError(err).Error("Unable to process and send webhooks")
	}

	logger.Debugf("Transitioned backup from %s to %s", oldState, backup.State)
}

// transitionBackup works with the given backup to transition it to a final state.
func (s *BackupSupervisor) transitionBackup(backup *model.InstallationBackup, instanceID string, logger log.FieldLogger) model.InstallationBackupState {
	switch backup.State {
	case model.InstallationBackupStateBackupRequested:
		return s.triggerBackup(backup, instanceID, logger)

	case model.InstallationBackupStateBackupInProgress:
		return s.monitorBackup(backup, instanceID, logger)

	case model.InstallationBackupStateDeletionRequested:
		return s.deleteBackup(backup, instanceID, logger)

	default:
		logger.Warnf("Found backup pending work in unexpected state %s", backup.State)
		return backup.State
	}
}

func (s *BackupSupervisor) triggerBackup(backup *model.InstallationBackup, instanceID string, logger log.FieldLogger) model.InstallationBackupState {
	installation, err := s.store.GetInstallation(backup.InstallationID, false, false)
	if err != nil {
		logger.WithError(err).Error("Failed to get installation")
		return backup.State
	}
	if installation == nil {
		logger.Errorf("Installation, with id %q not found, setting backup as failed", backup.InstallationID)
		return model.InstallationBackupStateBackupFailed
	}

	installationLock := newInstallationLock(installation.ID, instanceID, s.store, logger)
	if !installationLock.TryLock() {
		logger.Errorf("Failed to lock installation %s", installation.ID)
		return backup.State
	}
	defer installationLock.Unlock()

	err = model.EnsureInstallationReadyForBackup(installation)
	if err != nil {
		logger.WithError(err).Errorf("Installation is not backup compatible %s", installation.ID)
		return backup.State
	}

	backupCI, ciLock, err := claimClusterInstallation(s.store, installation, instanceID, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to claim Cluster Installation for backup")
		return backup.State
	}
	defer ciLock.Unlock()

	cluster, err := s.store.GetCluster(backupCI.ClusterID)
	if err != nil {
		logger.WithError(err).Error("Failed to get cluster")
		return backup.State
	}

	dataRes, err := s.provisioner.TriggerBackup(backup, cluster, installation)
	if err != nil {
		logger.WithError(err).Error("Failed to trigger backup")
		return backup.State
	}

	backup.DataResidence = dataRes
	backup.ClusterInstallationID = backupCI.ID

	err = s.store.UpdateInstallationBackupSchedulingData(backup)
	if err != nil {
		logger.Error("Failed to update backup data residency")
		return backup.State
	}

	return model.InstallationBackupStateBackupInProgress
}

func (s *BackupSupervisor) monitorBackup(backup *model.InstallationBackup, instanceID string, logger log.FieldLogger) model.InstallationBackupState {
	cluster, err := getClusterForClusterInstallation(s.store, backup.ClusterInstallationID)
	if err != nil {
		logger.WithError(err).Error("Failed to get cluster")
		return backup.State
	}

	startTime, err := s.provisioner.CheckBackupStatus(backup, cluster)
	if err != nil {
		if err == ErrJobBackoffLimitReached {
			logger.WithError(err).Error("Backup job backoff limit reached, backup failed")
			return model.InstallationBackupStateBackupFailed
		}
		logger.WithError(err).Error("Failed to check backup state")
		return backup.State
	}

	if startTime <= 0 {
		logger.Debugf("Backup in progress")
		return backup.State
	}

	backup.StartAt = startTime

	err = s.store.UpdateInstallationBackupStartTime(backup)
	if err != nil {
		logger.Error("Failed to update backup data start time")
		return backup.State
	}

	return model.InstallationBackupStateBackupSucceeded
}

func (s *BackupSupervisor) deleteBackup(backup *model.InstallationBackup, instanceID string, logger log.FieldLogger) model.InstallationBackupState {
	cluster, err := getClusterForClusterInstallation(s.store, backup.ClusterInstallationID)
	if err != nil {
		logger.WithError(err).Error("Failed to get cluster for backup")
		return backup.State
	}

	err = s.provisioner.CleanupBackupJob(backup, cluster)
	if err != nil {
		logger.WithError(err).Error("Failed to cleanup backup from cluster")
		return backup.State
	}

	if backup.DataResidence.URL != aws.S3URL {
		logger.WithError(err).Error("Only backups from S3 can be deleted")
		return model.InstallationBackupStateDeletionFailed
	}

	err = s.aws.S3EnsureObjectDeleted(backup.DataResidence.Bucket, backup.DataResidence.FullPath())
	if err != nil {
		logger.WithError(err).Error("Failed to delete backup from S3")
		return backup.State
	}

	err = s.store.DeleteInstallationBackup(backup.ID)
	if err != nil {
		logger.WithError(err).Error("Failed to mark backup as deleted")
		return backup.State
	}

	return model.InstallationBackupStateDeleted
}
