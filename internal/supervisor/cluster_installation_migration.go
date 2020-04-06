package supervisor

import (
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/utils"
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

// clusterInstallationMigrationStore abstracts the database operations required to query cluster installation migrations.
type clusterInstallationMigrationStore interface {
	GetClusterInstallationMigration(migrationID string) (*model.ClusterInstallationMigration, error)
	GetUnlockedClusterInstallationMigrationsPendingWork() ([]*model.ClusterInstallationMigration, error)
	LockClusterInstallationMigration(migrationID, lockerID string) (bool, error)
	UnlockClusterInstallationMigration(migrationID, lockerID string, force bool) (bool, error)
	DeleteClusterInstallationMigration(migrationID string) error
	CreateClusterInstallationMigration(migration *model.ClusterInstallationMigration) error
	UpdateClusterInstallationMigration(migration *model.ClusterInstallationMigration) error
}

// CIMSupervisorConfig configures a ClusterInstallationMigrationSupervisor instance.
type CIMSupervisorConfig struct {
	Store                         clusterInstallationMigrationStore
	ClusterSupervisorInstance     *ClusterSupervisor
	InstallationSupervisor        *InstallationSupervisor
	ClusterInstallationSupervisor *ClusterInstallationSupervisor
	ResourceUtil                  *utils.ResourceUtil
	AWSClient                     *aws.Client
	InstanceID                    string
	Logger                        log.FieldLogger
}

// ClusterInstallationMigrationSupervisor finds migrations pending work and effects the required changes.
//
// The degree of parallelism is controlled by a weighted semaphore, intended to be shared with
// other clients needing to coordinate background jobs.
type ClusterInstallationMigrationSupervisor struct {
	instanceID                    string
	store                         clusterInstallationMigrationStore
	logger                        log.FieldLogger
	resourceUtil                  *utils.ResourceUtil
	awsClient                     *aws.Client
	clusterSupervisor             *ClusterSupervisor
	installationSupervisor        *InstallationSupervisor
	clusterInstallationSupervisor *ClusterInstallationSupervisor
}

// NewClusterInstallationMigrationSupervisor creates a new ClusterInstallationMigrationSupervisor.
func NewClusterInstallationMigrationSupervisor(config *CIMSupervisorConfig) *ClusterInstallationMigrationSupervisor {
	return &ClusterInstallationMigrationSupervisor{
		instanceID:                    config.InstanceID,
		logger:                        config.Logger,
		store:                         config.Store,
		awsClient:                     config.AWSClient,
		resourceUtil:                  config.ResourceUtil,
		clusterSupervisor:             config.ClusterSupervisorInstance,
		installationSupervisor:        config.InstallationSupervisor,
		clusterInstallationSupervisor: config.ClusterInstallationSupervisor,
	}
}

// Do looks for work to be done on any pending installations and attempts to schedule the required work.
func (s *ClusterInstallationMigrationSupervisor) Do() error {
	migrations, err := s.store.GetUnlockedClusterInstallationMigrationsPendingWork()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to query for cluster installation migration pending work")
		return nil
	}

	for _, migration := range migrations {
		s.Supervise(migration)
	}

	return nil
}

// Supervise schedules the required work on the given migration.
func (s *ClusterInstallationMigrationSupervisor) Supervise(migration *model.ClusterInstallationMigration) {
	logger := s.logger.WithFields(log.Fields{
		"cluster-installation-migration": migration.ID,
	})

	lock := newClusterInstallationMigrationLock(migration.ID, s.instanceID, s.store, logger)
	if !lock.TryLock() {
		return
	}
	defer lock.Unlock()

	logger.Debugf("Supervising cluster installation migration in state %s", migration.State)

	newState := s.transitionClusterInstallationMigration(migration, logger)
	migration, err := s.store.GetClusterInstallationMigration(migration.ID)
	if err != nil {
		logger.WithError(err).Warnf("failed to get migration and thus persist state %s", newState)
		return
	}

	if migration.State == newState {
		return
	}

	oldState := migration.State
	migration.State = newState
	err = s.store.UpdateClusterInstallationMigration(migration)
	if err != nil {
		logger.WithError(err).Warnf("Failed to set migration state to %s", newState)
		return
	}

	logger.Debugf("Transitioned installation from %s to %s", oldState, newState)
}

// transitionMigration works with the given migration to migration it to a final state.
func (s *ClusterInstallationMigrationSupervisor) transitionClusterInstallationMigration(migration *model.ClusterInstallationMigration, logger log.FieldLogger) string {
	switch migration.State {
	case model.CIMigrationCreationRequested:
		return s.createClusterInstallationMigration(migration, logger)
	case model.CIMigrationCreationComplete:
		return s.createClusterInstallationSnapshot(migration, logger)
	case model.CIMigrationSnapshotCreationComplete:
		return s.restoreDatabase(migration, logger)
	case model.CIMigrationRestoreDatabaseComplete:
		return s.setupDatabase(migration, logger)
	case model.CIMigrationSetupDatabaseComplete:
		return s.createClusterInstallation(migration, logger)

	// case model.CIMigrationSnapshotCreationIP:
	// 	return s.createClusterInstallation(migration, logger)
	// case model.CIMigrationClusterInstallationCreationIP:
	// 	return s.waitForClusterInstallation(migration, logger)

	default:
		logger.Warnf("Found installation pending work in unexpected state %s", migration.State)
		return migration.State
	}
}

func (s *ClusterInstallationMigrationSupervisor) createClusterInstallationMigration(migration *model.ClusterInstallationMigration, logger log.FieldLogger) string {
	clusterInstallationMigration, err := s.store.GetClusterInstallationMigration(migration.ID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get cluster installation migration ID %s", clusterInstallationMigration.ID)
		return model.CIMigrationCreationFailed
	}
	if clusterInstallationMigration == nil {
		logger.Errorf("Cannot find any cluster installation migration with ID %s", clusterInstallationMigration.ID)
		return model.CIMigrationCreationFailed
	}

	clusterInstallation, err := s.clusterInstallationSupervisor.store.GetClusterInstallation(migration.ClusterInstallationID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get cluster installation ID %s", migration.ClusterInstallationID)
		return model.CIMigrationCreationFailed
	}
	if clusterInstallation == nil {
		logger.Errorf("Cannot find any cluster installation with ID %s", migration.ClusterInstallationID)
		return model.CIMigrationCreationFailed
	}

	if clusterInstallation.LockAcquiredBy == nil {
		clusterInstallationLocked, err := s.clusterInstallationSupervisor.store.LockClusterInstallations([]string{clusterInstallation.ID}, clusterInstallationMigration.ID)
		if err != nil {
			logger.WithError(err).Errorf("Unable to lock cluster installation: %s", clusterInstallation.ID)
			return model.CIMigrationCreationFailed
		}
		if clusterInstallationLocked {
			logger.Debugf("Still locking cluster installation id: %s", clusterInstallation.ID)
			return model.CIMigrationCreationRequested
		}
	}

	installation, err := s.installationSupervisor.store.GetInstallation(clusterInstallation.InstallationID, false, false)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get installation ID %s", clusterInstallation.InstallationID)
		return model.CIMigrationCreationFailed
	}
	if installation.Database != model.InstallationDatabaseAwsRDS {
		logger.Errorf("Database type %s is not supported", installation.Database)
		return model.CIMigrationCreationFailed
	}

	if installation.LockAcquiredBy == nil {
		installationLocked, err := s.installationSupervisor.store.LockInstallation(installation.ID, clusterInstallationMigration.ID)
		if err != nil {
			logger.WithError(err).Errorf("Unable to lock installation ID %s ", installation.ID)
			return model.CIMigrationCreationFailed
		}
		if !installationLocked {
			logger.Debugf("Still locking installation ID %s", installation.ID)
			return migration.State
		}
	}

	return model.CIMigrationCreationComplete
}

func (s *ClusterInstallationMigrationSupervisor) createClusterInstallationSnapshot(migration *model.ClusterInstallationMigration, logger log.FieldLogger) string {
	clusterInstallation, err := s.clusterInstallationSupervisor.store.GetClusterInstallation(migration.ClusterInstallationID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get cluster installation ID %s", migration.ClusterInstallationID)
		return model.CIMigrationCreationFailed
	}
	if clusterInstallation == nil {
		logger.Errorf("Cannot find any cluster installation with ID %s", migration.ClusterInstallationID)
		return model.CIMigrationCreationFailed
	}

	installation, err := s.installationSupervisor.store.GetInstallation(clusterInstallation.InstallationID, false, false)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get installation ID %s", clusterInstallation.InstallationID)
		return model.CIMigrationCreationFailed
	}
	if installation.Database != model.InstallationDatabaseAwsRDS {
		logger.Errorf("Database type %s is not supported", installation.Database)
		return model.CIMigrationCreationFailed
	}

	err = s.resourceUtil.GetDatabase(installation).Snapshot(logger)
	if err != nil {
		logger.WithError(err).Errorf("Failed to snapshot database for installation ID %s", installation.ID)
		return model.CIMigrationCreationFailed
	}

	return model.CIMigrationSnapshotCreationComplete
}

func (s *ClusterInstallationMigrationSupervisor) restoreDatabase(migration *model.ClusterInstallationMigration, logger log.FieldLogger) string {
	cluster, err := s.clusterSupervisor.store.GetCluster(migration.ClusterID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get cluster ID %s", migration.ClusterID)
		return model.CIMigrationCreationFailed
	}
	if cluster == nil {
		logger.Errorf("Cannot find any cluster with ID %s", migration.ClusterID)
		return model.CIMigrationCreationFailed
	}
	if cluster.State != model.ClusterStateStable {
		logger.Errorf("Cluster %s is not stable (currently %s)", cluster.ID, cluster.State)
		return model.CIMigrationCreationFailed
	}

	clusterInstallation, err := s.clusterInstallationSupervisor.store.GetClusterInstallation(migration.ClusterInstallationID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get cluster installation ID %s", migration.ClusterInstallationID)
		return model.CIMigrationCreationFailed
	}
	if clusterInstallation == nil {
		logger.Errorf("Cannot find any cluster installation with ID %s", migration.ClusterInstallationID)
		return model.CIMigrationCreationFailed
	}

	installation, err := s.installationSupervisor.store.GetInstallation(clusterInstallation.InstallationID, false, false)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get installation ID %s", clusterInstallation.InstallationID)
		return model.CIMigrationCreationFailed
	}
	if installation.Database != model.InstallationDatabaseAwsRDS {
		logger.Errorf("Database type %s is not supported", installation.Database)
		return model.CIMigrationCreationFailed
	}

	status, err := s.resourceUtil.GetDatabaseMigration(installation, cluster).Restore(logger)
	if err != nil {
		logger.Errorf("Failed to restore installation ID %s database in cluster ID %s", installation.ID, cluster.ID)
		return model.CIMigrationCreationFailed
	}

	switch status {
	case model.DatabaseMigrationStatusRestoreIP:
		return migration.State
	case model.DatabaseMigrationStatusRestoreComplete:
		return model.CIMigrationRestoreDatabaseComplete
	}

	return model.CIMigrationCreationFailed
}

func (s *ClusterInstallationMigrationSupervisor) setupDatabase(migration *model.ClusterInstallationMigration, logger log.FieldLogger) string {
	cluster, err := s.clusterSupervisor.store.GetCluster(migration.ClusterID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get cluster ID %s", migration.ClusterID)
		return model.CIMigrationCreationFailed
	}
	if cluster == nil {
		logger.Errorf("Cannot find any cluster with ID %s", migration.ClusterID)
		return model.CIMigrationCreationFailed
	}
	if cluster.State != model.ClusterStateStable {
		logger.Errorf("Cluster %s is not stable (currently %s)", cluster.ID, cluster.State)
		return model.CIMigrationCreationFailed
	}

	clusterInstallation, err := s.clusterInstallationSupervisor.store.GetClusterInstallation(migration.ClusterInstallationID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get cluster installation ID %s", migration.ClusterInstallationID)
		return model.CIMigrationCreationFailed
	}
	if clusterInstallation == nil {
		logger.Errorf("Cannot find any cluster installation with ID %s", migration.ClusterInstallationID)
		return model.CIMigrationCreationFailed
	}

	installation, err := s.installationSupervisor.store.GetInstallation(clusterInstallation.InstallationID, false, false)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get installation ID %s", clusterInstallation.InstallationID)
		return model.CIMigrationCreationFailed
	}
	if installation.Database != model.InstallationDatabaseAwsRDS {
		logger.Errorf("Database type %s is not supported", installation.Database)
		return model.CIMigrationCreationFailed
	}

	status, err := s.resourceUtil.GetDatabaseMigration(installation, cluster).Setup(logger)
	if err != nil {
		logger.Errorf("Failed to setup installation ID %s database in cluster ID %s", installation.ID, cluster.ID)
		return model.CIMigrationCreationFailed
	}

	switch status {
	case model.DatabaseMigrationStatusSetupIP:
		return migration.State
	case model.DatabaseMigrationStatusSetupComplete:
		return model.CIMigrationSetupDatabaseComplete
	}

	return model.CIMigrationCreationFailed
}

func (s *ClusterInstallationMigrationSupervisor) createClusterInstallation(migration *model.ClusterInstallationMigration, logger log.FieldLogger) string {
	clusterInstallationMigration, err := s.store.GetClusterInstallationMigration(migration.ID)
	if err != nil {
		logger.WithError(err).Errorf("Gailed to get cluster installation migration ID %s", clusterInstallationMigration.ID)
		return model.CIMigrationCreationFailed
	}
	if clusterInstallationMigration == nil {
		logger.Errorf("Cannot find any cluster installation migration with ID %s", clusterInstallationMigration.ID)
		return model.CIMigrationCreationFailed
	}

	cluster, err := s.clusterSupervisor.store.GetCluster(migration.ClusterID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get cluster ID %s", migration.ClusterID)
		return model.CIMigrationCreationFailed
	}
	if cluster == nil {
		logger.Errorf("Cannot find any cluster with ID %s", migration.ClusterID)
		return model.CIMigrationCreationFailed
	}
	if cluster.State != model.ClusterStateStable {
		logger.Errorf("Cluster %s is not stable (currently %s)", cluster.ID, cluster.State)
		return model.CIMigrationCreationFailed
	}

	clusterInstallation, err := s.clusterInstallationSupervisor.store.GetClusterInstallation(migration.ClusterInstallationID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get cluster installation ID %s", migration.ClusterInstallationID)
		return model.CIMigrationCreationFailed
	}
	if clusterInstallation == nil {
		logger.Errorf("Cannot find any cluster installation with ID %s", migration.ClusterInstallationID)
		return model.CIMigrationCreationFailed
	}

	installation, err := s.installationSupervisor.store.GetInstallation(clusterInstallation.InstallationID, false, false)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get installation ID %s", clusterInstallation.InstallationID)
		return model.CIMigrationCreationFailed
	}
	if installation.Database != model.InstallationDatabaseAwsRDS {
		logger.Errorf("Database type %s is not supported", installation.Database)
		return model.CIMigrationCreationFailed
	}

	clusterInstallationRequest := s.installationSupervisor.createClusterInstallation(cluster, installation, s.instanceID, logger)
	if clusterInstallationRequest != nil && clusterInstallationRequest.State != model.ClusterStateCreationRequested {
		logger.Errorf("Unexpected cluster installation state: %s", clusterInstallationRequest.State)
		return model.CIMigrationCreationFailed
	}

	err = s.installationSupervisor.Do()
	if err != nil {
		logger.WithError(err).Errorf("Failed when running the scheduler for creating new cluster installation")
		return model.CIMigrationCreationFailed
	}
	s.installationSupervisor.Supervise(installation)

	return model.CIMigrationClusterInstallationCreationComplete
}

// func (s *ClusterInstallationMigrationSupervisor) waitForDatabase(migration *model.ClusterInstallationMigration, logger log.FieldLogger) string {
// 	cluster, err := s.clusterSupervisor.store.GetCluster(migration.ClusterID)
// 	if err != nil {
// 		logger.WithError(err).Errorf("Failed to get cluster ID %s", migration.ClusterID)
// 		return model.CIMigrationCreationFailed
// 	}
// 	if cluster == nil {
// 		logger.Errorf("Cannot find any cluster with ID %s", migration.ClusterID)
// 		return model.CIMigrationCreationFailed
// 	}
// 	if cluster.State != model.ClusterStateStable {
// 		logger.Errorf("Cluster %s is not stable (currently %s)", cluster.ID, cluster.State)
// 		return model.CIMigrationCreationFailed
// 	}

// 	clusterInstallation, err := s.clusterInstallationSupervisor.store.GetClusterInstallation(migration.ClusterInstallationID)
// 	if err != nil {
// 		logger.WithError(err).Errorf("Failed to get cluster installation ID %s", migration.ClusterInstallationID)
// 		return model.CIMigrationCreationFailed
// 	}
// 	if clusterInstallation == nil {
// 		logger.Errorf("Cannot find any cluster installation with ID %s", migration.ClusterInstallationID)
// 		return model.CIMigrationCreationFailed
// 	}

// 	installation, err := s.installationSupervisor.store.GetInstallation(clusterInstallation.InstallationID, false, false)
// 	if err != nil {
// 		logger.WithError(err).Errorf("Failed to get installation ID %s", clusterInstallation.InstallationID)
// 		return model.CIMigrationCreationFailed
// 	}
// 	if installation.Database != model.InstallationDatabaseAwsRDS {
// 		logger.Errorf("Database type %s is not supported", installation.Database)
// 		return model.CIMigrationCreationFailed
// 	}

// 	databaseStatus, err := s.resourceUtil.GetDatabaseMigration(installation, cluster).Status(logger)
// 	if err != nil {
// 		logger.Errorf("Failed to get cluster installation ID %s database status", migration.ClusterInstallationID)
// 		return model.CIMigrationCreationFailed
// 	}

// 	switch databaseStatus {
// 	case model.DatabaseMigrationReplicaProvisionComplete:
// 		logger.Debug("database creation complete")
// 	case model.DatabaseMigrationReplicaProvisionIP:
// 		logger.Debug("database creation is still in progress")
// 		return migration.State
// 	}

// 	return model.CIMigrationRestoreDatabaseComplete

// }

// func (s *ClusterInstallationMigrationSupervisor) waitForClusterInstallation(migration *model.ClusterInstallationMigration, logger log.FieldLogger) string {

// 	clusterInstallationMigration, err := s.store.GetClusterInstallationMigration(migration.ID)
// 	if err != nil {
// 		logger.Errorf("failed to retrieve cluster installation migration: %s", err.Error())
// 		return model.CIMigrationCreationFailed
// 	}
// 	if clusterInstallationMigration.State != model.CIMigrationSnapshotCreationIP {
// 		return model.CIMigrationCreationFailed
// 	}

// 	clusterInstallation, err := s.clusterInstallationSupervisor.store.GetClusterInstallation(migration.ClusterInstallationID)
// 	if err != nil {
// 		logger.Errorf("failed to retrieve cluster installation migration: %s", err.Error())
// 		return model.CIMigrationCreationFailed
// 	}

// 	installation, err := s.installationSupervisor.store.GetInstallation(clusterInstallation.InstallationID)
// 	if err != nil {
// 		logger.Errorf("failed to retrieve installation: %s", err.Error())
// 		return model.CIMigrationCreationFailed
// 	}

// 	clusterInstallations, err := s.installationSupervisor.store.GetClusterInstallations(&model.ClusterInstallationFilter{
// 		ClusterID:      migration.ClusterID,
// 		InstallationID: installation.ID,
// 		IncludeDeleted: false,
// 	})
// 	if err != nil || len(clusterInstallations) != 1 {
// 		return model.CIMigrationCreationFailed
// 	}

// 	if clusterInstallations[0].State != model.ClusterInstallationStateStable {
// 		logger.Debug("still waiting on cluster installation to become stable")
// 		return migration.State
// 	}

// 	return model.CIMigrationClusterInstallationCreationComplete
// }

// func (s *ClusterInstallationMigrationSupervisor) waitForSnapshot(migration *model.ClusterInstallationMigration, logger log.FieldLogger) string {
// 	clusterInstallation, err := s.clusterInstallationSupervisor.store.GetClusterInstallation(migration.ClusterInstallationID)
// 	if err != nil {
// 		return model.CIMigrationCreationFailed
// 	}

// 	installation, err := s.installationSupervisor.store.GetInstallation(clusterInstallation.InstallationID)
// 	if err != nil {
// 		return model.CIMigrationCreationFailed
// 	}

// 	snapshotStatus, err := utils.GetDatabaseMigration(installation, clusterInstallation).SnapshotStatus(logger)
// 	if err != nil {
// 		logger.Errorf("failed to restore database: %s", err.Error())
// 		return model.CIMigrationCreationFailed
// 	}

// 	switch snapshotStatus {
// 	case model.DatabaseMigrationSnapshotCreationComplete:
// 		logger.Debug("snapshot creation is completed")
// 		return model.CIMigrationSnapshotCreationComplete
// 	case model.DatabaseMigrationSnapshotModifying:
// 		logger.Errorf("snapshot is being modified")
// 		return model.CIMigrationCreationFailed
// 	case model.DatabaseMigrationSnapshotCreationIP:
// 		logger.Debug("snapshot creation is still in progress")
// 	}

// 	return migration.State
// }
