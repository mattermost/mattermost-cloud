// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

// clusterStore abstracts the database operations required to query clusters.
type clusterStore interface {
	GetCluster(clusterID string) (*model.Cluster, error)
	GetUnlockedClustersPendingWork() ([]*model.Cluster, error)
	GetClusters(clusterFilter *model.ClusterFilter) ([]*model.Cluster, error)
	UpdateCluster(cluster *model.Cluster) error
	LockCluster(clusterID, lockerID string) (bool, error)
	UnlockCluster(clusterID string, lockerID string, force bool) (bool, error)
	DeleteCluster(clusterID string) error

	GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error)
}

// clusterProvisioner abstracts the provisioning operations required by the cluster supervisor.
type clusterProvisioner interface {
	PrepareCluster(cluster *model.Cluster) bool
	CreateCluster(cluster *model.Cluster, aws aws.AWS) error
	CheckClusterCreated(cluster *model.Cluster, awsClient aws.AWS) (bool, error)
	ProvisionCluster(cluster *model.Cluster, aws aws.AWS) error
	UpgradeCluster(cluster *model.Cluster, aws aws.AWS) error
	ResizeCluster(cluster *model.Cluster, aws aws.AWS) error
	DeleteCluster(cluster *model.Cluster, aws aws.AWS) (bool, error)
	RefreshKopsMetadata(cluster *model.Cluster) error
}

// ClusterSupervisor finds clusters pending work and effects the required changes.
//
// The degree of parallelism is controlled by a weighted semaphore, intended to be shared with
// other clients needing to coordinate background jobs.
type ClusterSupervisor struct {
	store          clusterStore
	provisioner    clusterProvisioner
	aws            aws.AWS
	eventsProducer eventProducer
	instanceID     string
	logger         log.FieldLogger
}

// NewClusterSupervisor creates a new ClusterSupervisor.
func NewClusterSupervisor(store clusterStore, clusterProvisioner clusterProvisioner, aws aws.AWS, eventProducer eventProducer, instanceID string, logger log.FieldLogger) *ClusterSupervisor {
	return &ClusterSupervisor{
		store:          store,
		provisioner:    clusterProvisioner,
		aws:            aws,
		eventsProducer: eventProducer,
		instanceID:     instanceID,
		logger:         logger,
	}
}

// Shutdown performs graceful shutdown tasks for the cluster supervisor.
func (s *ClusterSupervisor) Shutdown() {
	s.logger.Debug("Shutting down cluster supervisor")
}

// Do looks for work to be done on any pending clusters and attempts to schedule the required work.
func (s *ClusterSupervisor) Do() error {
	clusters, err := s.store.GetUnlockedClustersPendingWork()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to query for clusters pending work")
		return nil
	}

	for _, cluster := range clusters {
		s.Supervise(cluster)
	}

	return nil
}

// Supervise schedules the required work on the given cluster.
func (s *ClusterSupervisor) Supervise(cluster *model.Cluster) {
	logger := s.logger.WithFields(log.Fields{
		"cluster": cluster.ID,
	})

	lock := newClusterLock(cluster.ID, s.instanceID, s.store, logger)
	if !lock.TryLock() {
		return
	}
	defer lock.Unlock()

	// Before working on the cluster, it is crucial that we ensure that it was
	// not updated to a new state by another provisioning server.
	originalState := cluster.State
	cluster, err := s.store.GetCluster(cluster.ID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get refreshed cluster")
		return
	}
	if cluster.State != originalState {
		logger.WithField("oldClusterState", originalState).
			WithField("newClusterState", cluster.State).
			Warn("Another provisioner has worked on this cluster; skipping...")
		return
	}

	logger.Debugf("Supervising cluster in state %s", cluster.State)

	newState := s.transitionCluster(cluster, logger)

	cluster, err = s.store.GetCluster(cluster.ID)
	if err != nil {
		logger.WithError(err).Warnf("failed to get cluster and thus persist state %s", newState)
		return
	}

	if cluster.State == newState {
		return
	}

	oldState := cluster.State
	cluster.State = newState
	err = s.store.UpdateCluster(cluster)
	if err != nil {
		logger.WithError(err).Warnf("failed to set cluster state to %s", newState)
		return
	}

	err = s.eventsProducer.ProduceClusterStateChangeEvent(cluster, oldState)
	if err != nil {
		logger.WithError(err).Error("Failed to create cluster state change event")
	}

	logger.Debugf("Transitioned cluster from %s to %s", oldState, newState)
}

// Do works with the given cluster to transition it to a final state.
func (s *ClusterSupervisor) transitionCluster(cluster *model.Cluster, logger log.FieldLogger) string {
	switch cluster.State {
	case model.ClusterStateCreationRequested:
		return s.createCluster(cluster, logger)
	case model.ClusterStateCreationInProgress:
		return s.checkClusterCreated(cluster, logger)
	case model.ClusterStateProvisioningRequested:
		return s.provisionCluster(cluster, logger)
	case model.ClusterStateUpgradeRequested:
		return s.upgradeCluster(cluster, logger)
	case model.ClusterStateResizeRequested:
		return s.resizeCluster(cluster, logger)
	case model.ClusterStateRefreshMetadata:
		return s.refreshClusterMetadata(cluster, logger)
	case model.ClusterStateDeletionRequested:
		return s.deleteCluster(cluster, logger)
	default:
		logger.Warnf("Found cluster pending work in unexpected state %s", cluster.State)
		return cluster.State
	}
}

func (s *ClusterSupervisor) createCluster(cluster *model.Cluster, logger log.FieldLogger) string {
	var err error

	if s.provisioner.PrepareCluster(cluster) {
		err = s.store.UpdateCluster(cluster)
		if err != nil {
			logger.WithError(err).Error("Failed to record updated cluster after creation")
			return model.ClusterStateCreationFailed
		}
	}

	err = s.provisioner.CreateCluster(cluster, s.aws)
	if err != nil {
		logger.WithError(err).Error("Failed to create cluster")
		return model.ClusterStateCreationFailed
	}

	logger.Info("Finished creating cluster")
	return s.checkClusterCreated(cluster, logger)
}

func (s *ClusterSupervisor) provisionCluster(cluster *model.Cluster, logger log.FieldLogger) string {
	err := s.provisioner.ProvisionCluster(cluster, s.aws)
	if err != nil {
		logger.WithError(err).Error("Failed to provision cluster")
		return model.ClusterStateProvisioningFailed
	}

	logger.Info("Finished provisioning cluster")
	return s.refreshClusterMetadata(cluster, logger)
}

func (s *ClusterSupervisor) upgradeCluster(cluster *model.Cluster, logger log.FieldLogger) string {
	err := s.provisioner.UpgradeCluster(cluster, s.aws)
	if err != nil {
		logger.WithError(err).Error("Failed to upgrade cluster")
		logger.Info("Updating cluster store with latest cluster data")
		err = s.store.UpdateCluster(cluster)
		if err != nil {
			logger.WithError(err).Error("Failed to save updated cluster metadata")
			return model.ClusterStateRefreshMetadata
		}
		return model.ClusterStateUpgradeFailed
	}

	logger.Info("Finished upgrading cluster")
	return s.refreshClusterMetadata(cluster, logger)
}

func (s *ClusterSupervisor) resizeCluster(cluster *model.Cluster, logger log.FieldLogger) string {
	err := s.provisioner.ResizeCluster(cluster, s.aws)
	if err != nil {
		logger.WithError(err).Error("Failed to resize cluster")
		return model.ClusterStateResizeFailed
	}

	logger.Info("Finished resizing cluster")
	return s.refreshClusterMetadata(cluster, logger)
}

func (s *ClusterSupervisor) refreshClusterMetadata(cluster *model.Cluster, logger log.FieldLogger) string {
	if cluster.ProvisionerMetadataKops != nil {
		cluster.ProvisionerMetadataKops.ApplyChangeRequest()
		cluster.ProvisionerMetadataKops.ClearChangeRequest()
		cluster.ProvisionerMetadataKops.ClearRotatorRequest()
		cluster.ProvisionerMetadataKops.ClearWarnings()
	}

	err := s.provisioner.RefreshKopsMetadata(cluster)
	if err != nil {
		logger.WithError(err).Error("Failed to refresh cluster")
		return model.ClusterStateRefreshMetadata
	}
	err = s.store.UpdateCluster(cluster)
	if err != nil {
		logger.WithError(err).Error("Failed to save updated cluster metadata")
		return model.ClusterStateRefreshMetadata
	}

	return model.ClusterStateStable
}

func (s *ClusterSupervisor) deleteCluster(cluster *model.Cluster, logger log.FieldLogger) string {
	deleted, err := s.provisioner.DeleteCluster(cluster, s.aws)
	if err != nil {
		logger.WithError(err).Error("Failed to delete cluster")
		return model.ClusterStateDeletionFailed
	}
	if !deleted {
		logger.Info("Cluster still deleting")
		return model.ClusterInstallationStateDeletionRequested
	}

	err = s.store.DeleteCluster(cluster.ID)
	if err != nil {
		logger.WithError(err).Error("Failed to record updated cluster after deletion")
		return model.ClusterStateDeletionFailed
	}

	logger.Info("Finished deleting cluster")
	return model.ClusterStateDeleted
}

func (s *ClusterSupervisor) checkClusterCreated(cluster *model.Cluster, logger log.FieldLogger) string {
	ready, err := s.provisioner.CheckClusterCreated(cluster, s.aws)
	if err != nil {
		logger.WithError(err).Error("Failed to check if cluster creation finished")
		return model.ClusterStateCreationFailed
	}
	if !ready {
		logger.Info("Cluster not yet ready")
		return model.ClusterStateCreationInProgress
	}

	return s.provisionCluster(cluster, logger)
}
