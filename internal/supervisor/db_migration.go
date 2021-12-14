// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	"strings"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/common"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-cloud/internal/provisioner"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/webhook"
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

// installationDBMigrationStore abstracts the database operations required by the supervisor.
type installationDBMigrationStore interface {
	GetUnlockedInstallationDBMigrationOperationsPendingWork() ([]*model.InstallationDBMigrationOperation, error)
	GetInstallationDBMigrationOperation(id string) (*model.InstallationDBMigrationOperation, error)
	UpdateInstallationDBMigrationOperationState(dbMigration *model.InstallationDBMigrationOperation) error
	UpdateInstallationDBMigrationOperation(dbMigration *model.InstallationDBMigrationOperation) error
	DeleteInstallationDBMigrationOperation(id string) error
	installationDBMigrationOperationLockStore

	TriggerInstallationRestoration(installation *model.Installation, backup *model.InstallationBackup) (*model.InstallationDBRestorationOperation, error)
	GetInstallationDBRestorationOperation(id string) (*model.InstallationDBRestorationOperation, error)
	UpdateInstallationDBRestorationOperationState(dbRestoration *model.InstallationDBRestorationOperation) error
	UpdateInstallationDBRestorationOperation(dbRestoration *model.InstallationDBRestorationOperation) error

	IsInstallationBackupRunning(installationID string) (bool, error)
	CreateInstallationBackup(backup *model.InstallationBackup) error
	GetInstallationBackup(id string) (*model.InstallationBackup, error)
	UpdateInstallationBackupState(backupMeta *model.InstallationBackup) error
	installationBackupLockStore

	GetInstallation(installationID string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.Installation, error)
	UpdateInstallation(installation *model.Installation) error
	installationLockStore

	GetClusterInstallations(*model.ClusterInstallationFilter) ([]*model.ClusterInstallation, error)
	GetClusterInstallation(clusterInstallationID string) (*model.ClusterInstallation, error)
	clusterInstallationLockStore

	GetCluster(id string) (*model.Cluster, error)

	GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error)

	model.InstallationDatabaseStoreInterface
}

type dbMigrationCIProvisioner interface {
	ClusterInstallationProvisioner(version string) provisioner.ClusterInstallationProvisioner
	ExecClusterInstallationJob(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) error
}

type databaseProvider interface {
	GetDatabase(installationID, dbType string) model.Database
}

// DBMigrationSupervisor finds pending work and effects the required changes.
//
// The degree of parallelism is controlled by a weighted semaphore, intended to be shared with
// other clients needing to coordinate background jobs.
type DBMigrationSupervisor struct {
	store                    installationDBMigrationStore
	aws                      aws.AWS
	dbProvider               databaseProvider
	instanceID               string
	environment              string
	logger                   log.FieldLogger
	dbMigrationCIProvisioner dbMigrationCIProvisioner
	eventsProducer           eventProducer
}

// NewInstallationDBMigrationSupervisor creates a new DBMigrationSupervisor.
func NewInstallationDBMigrationSupervisor(
	store installationDBMigrationStore,
	aws aws.AWS,
	dbProvider databaseProvider,
	instanceID string,
	provisioner dbMigrationCIProvisioner,
	eventsProducer eventProducer,
	logger log.FieldLogger) *DBMigrationSupervisor {
	return &DBMigrationSupervisor{
		store:                    store,
		aws:                      aws,
		dbProvider:               dbProvider,
		instanceID:               instanceID,
		environment:              aws.GetCloudEnvironmentName(),
		logger:                   logger,
		eventsProducer:           eventsProducer,
		dbMigrationCIProvisioner: provisioner,
	}
}

// Shutdown performs graceful shutdown tasks for the supervisor.
func (s *DBMigrationSupervisor) Shutdown() {
	s.logger.Debug("Shutting down installation db restoration supervisor")
}

// Do looks for work to be done on any pending backups and attempts to schedule the required work.
func (s *DBMigrationSupervisor) Do() error {
	installationDBMigrations, err := s.store.GetUnlockedInstallationDBMigrationOperationsPendingWork()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to query for pending work")
		return nil
	}

	for _, migration := range installationDBMigrations {
		s.Supervise(migration)
	}

	return nil
}

// Supervise schedules the required work on the given backup.
func (s *DBMigrationSupervisor) Supervise(migration *model.InstallationDBMigrationOperation) {
	logger := s.logger.WithFields(log.Fields{
		"dbMigrationOperation": migration.ID,
	})

	lock := newInstallationDBMigrationOperationLock(migration.ID, s.instanceID, s.store, logger)
	if !lock.TryLock() {
		return
	}
	defer lock.Unlock()

	// Before working on the migration operation, it is crucial that we ensure that it
	// was not updated to a new state by another provisioning server.
	originalState := migration.State
	migration, err := s.store.GetInstallationDBMigrationOperation(migration.ID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get refreshed migration")
		return
	}
	if migration.State != originalState {
		logger.WithField("oldRestorationState", originalState).
			WithField("newRestorationState", migration.State).
			Warn("Another provisioner has worked on this migration; skipping...")
		return
	}

	logger.Debugf("Supervising migration in state %s", migration.State)

	newState := s.transitionMigration(migration, s.instanceID, logger)

	migration, err = s.store.GetInstallationDBMigrationOperation(migration.ID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get migration and thus persist state %s", newState)
		return
	}

	if migration.State == newState {
		return
	}

	oldState := migration.State
	migration.State = newState

	err = s.store.UpdateInstallationDBMigrationOperationState(migration)
	if err != nil {
		logger.WithError(err).Errorf("Failed to set migration state to %s", newState)
		return
	}

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeInstallationDBMigration,
		ID:        migration.ID,
		NewState:  string(migration.State),
		OldState:  string(oldState),
		Timestamp: time.Now().UnixNano(),
		ExtraData: map[string]string{"Environment": s.aws.GetCloudEnvironmentName()},
	}
	err = webhook.SendToAllWebhooks(s.store, webhookPayload, logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		logger.WithError(err).Error("Unable to process and send webhooks")
	}

	logger.Debugf("Transitioned db migration from %s to %s", oldState, migration.State)
}

// transitionMigration works with the given db migration to transition it to a final state.
func (s *DBMigrationSupervisor) transitionMigration(dbMigration *model.InstallationDBMigrationOperation, instanceID string, logger log.FieldLogger) model.InstallationDBMigrationOperationState {
	switch dbMigration.State {
	case model.InstallationDBMigrationStateRequested:
		return s.triggerInstallationBackup(dbMigration, instanceID, logger)
	case model.InstallationDBMigrationStateBackupInProgress:
		return s.waitForInstallationBackup(dbMigration, instanceID, logger)
	case model.InstallationDBMigrationStateDatabaseSwitch:
		return s.switchDatabase(dbMigration, instanceID, logger)
	case model.InstallationDBMigrationStateRefreshSecrets:
		return s.refreshCredentials(dbMigration, instanceID, logger)
	case model.InstallationDBMigrationStateTriggerRestoration:
		return s.triggerInstallationRestoration(dbMigration, instanceID, logger)
	case model.InstallationDBMigrationStateRestorationInProgress:
		return s.waitForInstallationRestoration(dbMigration, instanceID, logger)
	case model.InstallationDBMigrationStateUpdatingInstallationConfig:
		return s.updateInstallationConfig(dbMigration, instanceID, logger)
	case model.InstallationDBMigrationStateFinalizing:
		return s.finalizeMigration(dbMigration, instanceID, logger)
	case model.InstallationDBMigrationStateFailing:
		return s.failMigration(dbMigration, instanceID, logger)
	case model.InstallationDBMigrationStateRollbackRequested:
		return s.rollbackMigration(dbMigration, instanceID, logger)
	case model.InstallationDBMigrationStateDeletionRequested:
		return s.cleanupMigration(dbMigration, instanceID, logger)
	default:
		logger.Warnf("Found migration pending work in unexpected state %s", dbMigration.State)
		return dbMigration.State
	}
}

// TODO: Possibly allow passing existing backupID to migrate from.
func (s *DBMigrationSupervisor) triggerInstallationBackup(dbMigration *model.InstallationDBMigrationOperation, instanceID string, logger log.FieldLogger) model.InstallationDBMigrationOperationState {
	installation, lock, err := getAndLockInstallation(s.store, dbMigration.InstallationID, instanceID, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to get and lock installation")
		return dbMigration.State
	}
	defer lock.Unlock()

	backup, err := common.TriggerInstallationBackup(s.store, installation, s.environment, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to trigger installation backup")
		return dbMigration.State
	}

	dbMigration.BackupID = backup.ID
	err = s.store.UpdateInstallationDBMigrationOperation(dbMigration)
	if err != nil {
		logger.WithError(err).Error("Failed to set backup ID for DB migration")
		return dbMigration.State
	}

	return model.InstallationDBMigrationStateBackupInProgress
}

func (s *DBMigrationSupervisor) waitForInstallationBackup(dbMigration *model.InstallationDBMigrationOperation, instanceID string, logger log.FieldLogger) model.InstallationDBMigrationOperationState {
	backup, err := s.store.GetInstallationBackup(dbMigration.BackupID)
	if err != nil {
		logger.WithError(err).Error("Failed to get installation backup")
		return dbMigration.State
	}

	switch backup.State {
	case model.InstallationBackupStateBackupSucceeded:
		logger.Info("Backup for migration finished successfully")
		return model.InstallationDBMigrationStateDatabaseSwitch
	case model.InstallationBackupStateBackupFailed:
		logger.Error("Backup for migration failed")
		return model.InstallationDBMigrationStateFailing
	case model.InstallationBackupStateBackupInProgress, model.InstallationBackupStateBackupRequested:
		logger.Debug("Backup for migration in progress")
		return dbMigration.State
	default:
		logger.Errorf("Unexpected state of installation backup for migration: %q", backup.State)
		return dbMigration.State
	}
}

func (s *DBMigrationSupervisor) switchDatabase(dbMigration *model.InstallationDBMigrationOperation, instanceID string, logger log.FieldLogger) model.InstallationDBMigrationOperationState {
	installation, lock, err := getAndLockInstallation(s.store, dbMigration.InstallationID, instanceID, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to get and lock installation")
		return dbMigration.State
	}
	defer lock.Unlock()

	sourceDB := s.dbProvider.GetDatabase(installation.ID, dbMigration.SourceDatabase)

	err = sourceDB.MigrateOut(s.store, dbMigration, logger)
	if err != nil {
		logger.WithError(err).Errorf("Failed to migrate installation out of database")
		return dbMigration.State
	}

	destinationDB := s.dbProvider.GetDatabase(installation.ID, dbMigration.DestinationDatabase)
	err = destinationDB.MigrateTo(s.store, dbMigration, logger)
	if err != nil {
		logger.WithError(err).Errorf("Failed to migrate installation to database")
		return dbMigration.State
	}

	installation.Database = dbMigration.DestinationDatabase
	err = s.store.UpdateInstallation(installation)
	if err != nil {
		logger.WithError(err).Errorf("Failed to switch database for installation")
		return dbMigration.State
	}

	return model.InstallationDBMigrationStateRefreshSecrets
}

func (s *DBMigrationSupervisor) refreshCredentials(dbMigration *model.InstallationDBMigrationOperation, instanceID string, logger log.FieldLogger) model.InstallationDBMigrationOperationState {
	installation, lock, err := getAndLockInstallation(s.store, dbMigration.InstallationID, instanceID, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to get and lock installation")
		return dbMigration.State
	}
	defer lock.Unlock()

	err = s.refreshSecrets(installation)
	if err != nil {
		logger.WithError(err).Error("Failed to refresh credentials for cluster installations")
		return dbMigration.State
	}

	return model.InstallationDBMigrationStateTriggerRestoration
}

func (s *DBMigrationSupervisor) triggerInstallationRestoration(dbMigration *model.InstallationDBMigrationOperation, instanceID string, logger log.FieldLogger) model.InstallationDBMigrationOperationState {
	installation, lock, err := getAndLockInstallation(s.store, dbMigration.InstallationID, instanceID, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to get and lock installation")
		return dbMigration.State
	}
	defer lock.Unlock()

	backup, err := s.store.GetInstallationBackup(dbMigration.BackupID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get backup")
		return dbMigration.State
	}
	if backup == nil {
		logger.Errorf("Backup not found on restoration phase")
		return model.InstallationDBMigrationStateFailing
	}

	dbRestoration, err := common.TriggerInstallationDBRestoration(s.store, installation, backup, s.eventsProducer, s.environment, logger)
	if err != nil {
		s.logger.WithError(err).Error("Failed to trigger installation db restoration")
		return dbMigration.State
	}

	dbMigration.InstallationDBRestorationOperationID = dbRestoration.ID
	err = s.store.UpdateInstallationDBMigrationOperation(dbMigration)
	if err != nil {
		logger.WithError(err).Error("Failed to set restoration operation ID for DB migration")
		return dbMigration.State
	}

	return model.InstallationDBMigrationStateRestorationInProgress
}

func (s *DBMigrationSupervisor) waitForInstallationRestoration(dbMigration *model.InstallationDBMigrationOperation, instanceID string, logger log.FieldLogger) model.InstallationDBMigrationOperationState {
	restoration, err := s.store.GetInstallationDBRestorationOperation(dbMigration.InstallationDBRestorationOperationID)
	if err != nil {
		logger.WithError(err).Error("Failed to get installation restoration")
		return dbMigration.State
	}

	switch restoration.State {
	case model.InstallationDBRestorationStateSucceeded:
		logger.Info("Restoration for migration finished successfully")
		return model.InstallationDBMigrationStateUpdatingInstallationConfig
	case model.InstallationDBRestorationStateFailed, model.InstallationDBRestorationStateInvalid:
		logger.Error("Restoration for migration failed or is invalid")
		return model.InstallationDBMigrationStateFailing
	default:
		logger.Debug("Restoration for migration in progress")
		return dbMigration.State
	}
}

func (s *DBMigrationSupervisor) updateInstallationConfig(dbMigration *model.InstallationDBMigrationOperation, instanceID string, logger log.FieldLogger) model.InstallationDBMigrationOperationState {
	installation, err := s.store.GetInstallation(dbMigration.InstallationID, false, false)
	if err != nil {
		logger.WithError(err).Error("Failed to get installation")
		return dbMigration.State
	}
	if installation == nil {
		logger.Error("Installation not found")
		return dbMigration.State
	}

	clusterInstallation, ciLock, err := claimClusterInstallation(s.store, installation, instanceID, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to claim cluster installation")
		return dbMigration.State
	}
	defer ciLock.Unlock()

	cluster, err := s.store.GetCluster(clusterInstallation.ClusterID)
	if err != nil {
		logger.WithError(err).Error("Failed to get cluster")
		return dbMigration.State
	}
	if cluster == nil {
		logger.Error("Cluster not found")
		return dbMigration.State
	}

	var command []string
	if strings.HasPrefix(installation.Version, "5.") {
		command = []string{"/bin/sh", "-c", "mattermost config set SqlSettings.DataSource $MM_CONFIG"}
	} else {
		// As `mattermost config` command was removed in v6 we need to work around performing config change without any other running server.
		// This command does the following:
		// - Start Mattermost server with disabled clustering as a background job.
		// - Waits for a successful ping.
		// - Executes config change with mmctl.
		// - Attempts to terminate Mattermost server (we want it to should down gracefully if possible).
		// WARNING: this should not be done if other MM pods are online as disabled clustering may lead to some issues.
		command = []string{"/bin/sh", "-c", "MM_CLUSTERSETTINGS_ENABLE=false mattermost & pid=$!; until $(curl --output /dev/null --silent --fail localhost:8065/api/v4/system/ping); do sleep 2; done; mmctl --local config set SqlSettings.DataSource $MM_CONFIG && kill $pid"}
	}

	err = s.dbMigrationCIProvisioner.ExecClusterInstallationJob(cluster, clusterInstallation, command...)
	if err != nil {
		logger.WithError(err).Error("Failed to execute command on cluster installation")
		return dbMigration.State
	}

	return model.InstallationDBMigrationStateFinalizing
}

func (s *DBMigrationSupervisor) finalizeMigration(dbMigration *model.InstallationDBMigrationOperation, instanceID string, logger log.FieldLogger) model.InstallationDBMigrationOperationState {
	installation, lock, err := getAndLockInstallation(s.store, dbMigration.InstallationID, instanceID, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to get and lock installation")
		return dbMigration.State
	}
	defer lock.Unlock()

	oldState := installation.State

	installation.State = model.InstallationStateHibernating
	err = s.store.UpdateInstallation(installation)
	if err != nil {
		logger.WithError(err).Errorf("Failed to set installation back to hibernating after migration")
		return dbMigration.State
	}

	err = s.eventsProducer.ProduceInstallationStateChangeEvent(installation, oldState)
	if err != nil {
		logger.WithError(err).Error("Failed to create installation state change event")
	}

	dbMigration.CompleteAt = model.GetMillis()
	err = s.store.UpdateInstallationDBMigrationOperation(dbMigration)
	if err != nil {
		logger.WithError(err).Errorf("Failed to set complete at for db migration")
		return dbMigration.State
	}

	return model.InstallationDBMigrationStateSucceeded
}

func (s *DBMigrationSupervisor) failMigration(dbMigration *model.InstallationDBMigrationOperation, instanceID string, logger log.FieldLogger) model.InstallationDBMigrationOperationState {
	installation, lock, err := getAndLockInstallation(s.store, dbMigration.InstallationID, instanceID, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to get and lock installation")
		return dbMigration.State
	}
	defer lock.Unlock()

	oldState := installation.State

	installation.State = model.InstallationStateDBMigrationFailed
	err = s.store.UpdateInstallation(installation)
	if err != nil {
		logger.WithError(err).Errorf("Failed to set installation back to hibernating after migration")
		return dbMigration.State
	}

	err = s.eventsProducer.ProduceInstallationStateChangeEvent(installation, oldState)
	if err != nil {
		logger.WithError(err).Error("Failed to create installation state change event")
	}

	return model.InstallationDBMigrationStateFailed
}

func (s *DBMigrationSupervisor) rollbackMigration(dbMigration *model.InstallationDBMigrationOperation, instanceID string, logger log.FieldLogger) model.InstallationDBMigrationOperationState {
	installation, lock, err := getAndLockInstallation(s.store, dbMigration.InstallationID, instanceID, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to get and lock installation")
		return dbMigration.State
	}
	defer lock.Unlock()

	// TODO: this approach is slightly simplified and will need to be split to 2 methods
	// when we want to support different database types.
	destinationDB := s.dbProvider.GetDatabase(installation.ID, dbMigration.DestinationDatabase)
	err = destinationDB.RollbackMigration(s.store, dbMigration, logger)
	if err != nil {
		logger.WithError(err).Errorf("Failed to migrate installation to database")
		return dbMigration.State
	}

	installation.Database = dbMigration.SourceDatabase
	err = s.store.UpdateInstallation(installation)
	if err != nil {
		logger.WithError(err).Error("Failed to switch database for installation")
		return dbMigration.State
	}

	err = s.refreshSecrets(installation)
	if err != nil {
		logger.WithError(err).Error("Failed to refresh secrets on cluster installations during rollback")
		return dbMigration.State
	}

	oldState := installation.State
	installation.State = model.InstallationStateHibernating
	err = s.store.UpdateInstallation(installation)
	if err != nil {
		logger.WithError(err).Error("Failed to set installation back to hibernating state")
		return dbMigration.State
	}

	err = s.eventsProducer.ProduceInstallationStateChangeEvent(installation, oldState)
	if err != nil {
		logger.WithError(err).Error("Failed to create installation state change event")
	}

	return model.InstallationDBMigrationStateRollbackFinished
}

func (s *DBMigrationSupervisor) refreshSecrets(installation *model.Installation) error {
	cis, err := s.store.GetClusterInstallations(&model.ClusterInstallationFilter{InstallationID: installation.ID, Paging: model.AllPagesNotDeleted()})
	if err != nil {
		return errors.Wrap(err, "failed to get cluster installations")
	}

	for _, ci := range cis {
		cluster, err := s.store.GetCluster(ci.ClusterID)
		if err != nil {
			return errors.Wrap(err, "failed to get cluster")
		}

		err = s.dbMigrationCIProvisioner.ClusterInstallationProvisioner(installation.CRVersion).
			RefreshSecrets(cluster, installation, cis[0])
		if err != nil {
			return errors.Wrap(err, "failed to refresh credentials of cluster installation")
		}
	}
	return nil
}

func (s *DBMigrationSupervisor) cleanupMigration(dbMigration *model.InstallationDBMigrationOperation, instanceID string, logger log.FieldLogger) model.InstallationDBMigrationOperationState {
	err := s.cleanupMigratedDBs(dbMigration, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to cleanup source database")
		return dbMigration.State
	}

	err = s.store.DeleteInstallationDBMigrationOperation(dbMigration.ID)
	if err != nil {
		logger.WithError(err).Error("Failed to mark migration operation as deleted")
		return dbMigration.State
	}

	return model.InstallationDBMigrationStateDeleted
}

func (s *DBMigrationSupervisor) cleanupMigratedDBs(dbMigration *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	sourceDB := s.dbProvider.GetDatabase(dbMigration.InstallationID, dbMigration.SourceDatabase)

	err := sourceDB.TeardownMigrated(s.store, dbMigration, logger)
	if err != nil {
		return errors.Wrap(err, "failed to tear down migrated database")
	}

	return nil
}
