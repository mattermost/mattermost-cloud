// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	"time"

	"github.com/mattermost/mattermost-cloud/internal/provisioner"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/webhook"
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

// installationDBRestorationStore abstracts the database operations required by the supervisor.
type installationDBRestorationStore interface {
	GetUnlockedInstallationDBRestorationOperationsPendingWork() ([]*model.InstallationDBRestorationOperation, error)
	GetInstallationDBRestorationOperation(id string) (*model.InstallationDBRestorationOperation, error)
	UpdateInstallationDBRestorationOperationState(dbRestoration *model.InstallationDBRestorationOperation) error
	UpdateInstallationDBRestorationOperation(dbRestoration *model.InstallationDBRestorationOperation) error
	installationDBRestorationLockStore

	GetInstallationBackup(id string) (*model.InstallationBackup, error)
	installationBackupLockStore

	GetInstallation(installationID string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.Installation, error)
	UpdateInstallation(installation *model.Installation) error
	installationLockStore

	GetClusterInstallations(*model.ClusterInstallationFilter) ([]*model.ClusterInstallation, error)
	GetClusterInstallation(clusterInstallationID string) (*model.ClusterInstallation, error)
	clusterInstallationLockStore

	GetCluster(id string) (*model.Cluster, error)

	GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error)
}

// restoreOperator abstracts different restoration operations required by the installation db restoration supervisor.
type restoreOperator interface {
	TriggerRestore(installation *model.Installation, backup *model.InstallationBackup, cluster *model.Cluster) error
	CheckRestoreStatus(backupMeta *model.InstallationBackup, cluster *model.Cluster) (int64, error)
	CleanupRestoreJob(backup *model.InstallationBackup, cluster *model.Cluster) error
}

// InstallationDBRestorationSupervisor finds pending work and effects the required changes.
//
// The degree of parallelism is controlled by a weighted semaphore, intended to be shared with
// other clients needing to coordinate background jobs.
type InstallationDBRestorationSupervisor struct {
	store       installationDBRestorationStore
	aws         aws.AWS
	instanceID  string
	environment string
	logger      log.FieldLogger

	restoreOperator restoreOperator
}

// NewInstallationDBRestorationSupervisor creates a new InstallationDBRestorationSupervisor.
func NewInstallationDBRestorationSupervisor(
	store installationDBRestorationStore,
	aws aws.AWS,
	restoreOperator restoreOperator,
	instanceID string,
	logger log.FieldLogger) *InstallationDBRestorationSupervisor {
	return &InstallationDBRestorationSupervisor{
		store:           store,
		aws:             aws,
		restoreOperator: restoreOperator,
		instanceID:      instanceID,
		environment:     aws.GetCloudEnvironmentName(),
		logger:          logger,
	}
}

// Shutdown performs graceful shutdown tasks for the supervisor.
func (s *InstallationDBRestorationSupervisor) Shutdown() {
	s.logger.Debug("Shutting down installation db restoration supervisor")
}

// Do looks for work to be done on any pending restoration operations and attempts to schedule the required work.
func (s *InstallationDBRestorationSupervisor) Do() error {
	installationDBRestorations, err := s.store.GetUnlockedInstallationDBRestorationOperationsPendingWork()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to query for pending work")
		return nil
	}

	for _, restoration := range installationDBRestorations {
		s.Supervise(restoration)
	}

	return nil
}

// Supervise schedules the required work on the given restoration.
func (s *InstallationDBRestorationSupervisor) Supervise(restoration *model.InstallationDBRestorationOperation) {
	logger := s.logger.WithFields(log.Fields{
		"restorationOperation": restoration.ID,
	})

	lock := newInstallationDBRestorationLock(restoration.ID, s.instanceID, s.store, logger)
	if !lock.TryLock() {
		return
	}
	defer lock.Unlock()

	// Before working on the restoration, it is crucial that we ensure that it
	// was not updated to a new state by another provisioning server.
	originalState := restoration.State
	restoration, err := s.store.GetInstallationDBRestorationOperation(restoration.ID)
	if err != nil {
		logger.WithError(err).Error("Failed to get refreshed restoration")
		return
	}
	if restoration.State != originalState {
		logger.WithField("oldRestorationState", originalState).
			WithField("newRestorationState", restoration.State).
			Warn("Another provisioner has worked on this restoration; skipping...")
		return
	}

	logger.Debugf("Supervising restoration in state %s", restoration.State)

	newState := s.transitionRestoration(restoration, s.instanceID, logger)

	restoration, err = s.store.GetInstallationDBRestorationOperation(restoration.ID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get restoration and thus persist state %s", newState)
		return
	}

	if restoration.State == newState {
		return
	}

	oldState := restoration.State
	restoration.State = newState

	err = s.store.UpdateInstallationDBRestorationOperationState(restoration)
	if err != nil {
		logger.WithError(err).Errorf("Failed to set restoration state to %s", newState)
		return
	}

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeInstallationDBRestoration,
		ID:        restoration.ID,
		NewState:  string(restoration.State),
		OldState:  string(oldState),
		Timestamp: time.Now().UnixNano(),
		ExtraData: map[string]string{"Environment": s.environment},
	}
	err = webhook.SendToAllWebhooks(s.store, webhookPayload, logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		logger.WithError(err).Error("Unable to process and send webhooks")
	}

	logger.Debugf("Transitioned restoration from %s to %s", oldState, restoration.State)
}

// transitionRestoration works with the given restoration to transition it to a final state.
func (s *InstallationDBRestorationSupervisor) transitionRestoration(restoration *model.InstallationDBRestorationOperation, instanceID string, logger log.FieldLogger) model.InstallationDBRestorationState {
	switch restoration.State {
	case model.InstallationDBRestorationStateRequested:
		return s.triggerRestoration(restoration, instanceID, logger)

	case model.InstallationDBRestorationStateInProgress:
		return s.checkRestorationStatus(restoration, instanceID, logger)

	case model.InstallationDBRestorationStateFinalizing:
		return s.finalizeRestoration(restoration, instanceID, logger)

	case model.InstallationDBRestorationStateFailing:
		return s.failRestoration(restoration, instanceID, logger)

	default:
		logger.Warnf("Found restoration pending work in unexpected state %s", restoration.State)
		return restoration.State
	}
}

func (s *InstallationDBRestorationSupervisor) triggerRestoration(restoration *model.InstallationDBRestorationOperation, instanceID string, logger log.FieldLogger) model.InstallationDBRestorationState {
	installation, lock, err := getAndLockInstallation(s.store, restoration.InstallationID, instanceID, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to get and lock installation")
		return restoration.State
	}
	defer lock.Unlock()

	backup, err := s.store.GetInstallationBackup(restoration.BackupID)
	if err != nil {
		logger.WithError(err).Error("Failed to get backup")
		return restoration.State
	}

	if restoration.ClusterInstallationID == "" {
		restoreCI, ciLock, err := claimClusterInstallation(s.store, installation, instanceID, logger)
		if err != nil {
			logger.WithError(err).Error("Failed to claim Cluster Installation for restoration")
			return restoration.State
		}
		defer ciLock.Unlock()
		restoration.ClusterInstallationID = restoreCI.ID
		err = s.store.UpdateInstallationDBRestorationOperation(restoration)
		if err != nil {
			logger.WithError(err).Error("Failed to assign cluster installation to restoration")
			return restoration.State
		}
	}

	cluster, err := getClusterForClusterInstallation(s.store, restoration.ClusterInstallationID)
	if err != nil {
		logger.WithError(err).Error("Failed to get cluster for restoration")
		return restoration.State
	}

	err = s.restoreOperator.TriggerRestore(installation, backup, cluster)
	if err != nil {
		logger.WithError(err).Error("Failed to trigger restoration job")
		return restoration.State
	}

	return model.InstallationDBRestorationStateInProgress
}

func (s *InstallationDBRestorationSupervisor) checkRestorationStatus(restoration *model.InstallationDBRestorationOperation, instanceID string, logger log.FieldLogger) model.InstallationDBRestorationState {
	backup, err := s.store.GetInstallationBackup(restoration.BackupID)
	if err != nil {
		logger.WithError(err).Error("Failed to get backup")
		return restoration.State
	}

	cluster, err := getClusterForClusterInstallation(s.store, restoration.ClusterInstallationID)
	if err != nil {
		logger.WithError(err).Error("Failed to get cluster for restoration")
		return restoration.State
	}

	completeAt, err := s.restoreOperator.CheckRestoreStatus(backup, cluster)
	if err != nil {
		if err == provisioner.ErrJobBackoffLimitReached {
			logger.WithError(err).Error("Installation db restoration failed")
			return model.InstallationDBRestorationStateFailing
		}
		logger.WithError(err).Error("Failed to check restoration status")
		return restoration.State
	}
	if completeAt <= 0 {
		logger.Info("Database restoration still in progress")
		return restoration.State
	}

	restoration.CompleteAt = completeAt
	err = s.store.UpdateInstallationDBRestorationOperation(restoration)
	if err != nil {
		logger.WithError(err).Error("Failed to update restoration")
		return restoration.State
	}

	return model.InstallationDBRestorationStateFinalizing
}

func (s *InstallationDBRestorationSupervisor) finalizeRestoration(restoration *model.InstallationDBRestorationOperation, instanceID string, logger log.FieldLogger) model.InstallationDBRestorationState {
	installation, lock, err := getAndLockInstallation(s.store, restoration.InstallationID, instanceID, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to get and lock installation")
		return restoration.State
	}
	defer lock.Unlock()

	oldState := installation.State
	installation.State = restoration.TargetInstallationState
	err = s.store.UpdateInstallation(installation)
	if err != nil {
		logger.WithError(err).Error("Failed to set installation to target state after restore")
		return restoration.State
	}

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeInstallation,
		ID:        installation.ID,
		NewState:  installation.State,
		OldState:  oldState,
		Timestamp: time.Now().UnixNano(),
		ExtraData: map[string]string{"DNS": installation.DNS, "Environment": s.environment},
	}

	err = webhook.SendToAllWebhooks(s.store, webhookPayload, logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		logger.WithError(err).Error("Unable to process and send webhooks")
	}

	return model.InstallationDBRestorationStateSucceeded
}

func (s *InstallationDBRestorationSupervisor) failRestoration(restoration *model.InstallationDBRestorationOperation, instanceID string, logger log.FieldLogger) model.InstallationDBRestorationState {
	installation, lock, err := getAndLockInstallation(s.store, restoration.InstallationID, instanceID, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to get and lock installation")
		return restoration.State
	}
	defer lock.Unlock()

	oldState := installation.State
	installation.State = model.InstallationStateDBRestorationFailed
	err = s.store.UpdateInstallation(installation)
	if err != nil {
		logger.WithError(err).Error("Failed to set installation to failed DB restoration state")
		return restoration.State
	}

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeInstallation,
		ID:        installation.ID,
		NewState:  installation.State,
		OldState:  oldState,
		Timestamp: time.Now().UnixNano(),
		ExtraData: map[string]string{"DNS": installation.DNS, "Environment": s.environment},
	}

	err = webhook.SendToAllWebhooks(s.store, webhookPayload, logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		logger.WithError(err).Error("Unable to process and send webhooks")
	}

	return model.InstallationDBRestorationStateFailed
}
