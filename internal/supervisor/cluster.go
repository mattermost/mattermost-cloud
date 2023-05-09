// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	"sort"

	"github.com/mattermost/mattermost-cloud/internal/metrics"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
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

	GetStateChangeEvents(filter *model.StateChangeEventFilter) ([]*model.StateChangeEventData, error)
}

// ClusterProvisioner abstracts the provisioning operations required by the cluster supervisor.
type ClusterProvisioner interface {
	PrepareCluster(cluster *model.Cluster) bool
	CreateCluster(cluster *model.Cluster) error
	CheckClusterCreated(cluster *model.Cluster) (bool, error)
	CreateNodegroups(cluster *model.Cluster) error
	CheckNodegroupsCreated(cluster *model.Cluster) (bool, error)
	DeleteNodegroups(cluster *model.Cluster) error
	ProvisionCluster(cluster *model.Cluster) error
	UpgradeCluster(cluster *model.Cluster) error
	ResizeCluster(cluster *model.Cluster) error
	DeleteCluster(cluster *model.Cluster) (bool, error)
	RefreshClusterMetadata(cluster *model.Cluster) error
}

type ClusterProvisionerOption interface {
	GetClusterProvisioner(provisioner string) ClusterProvisioner
}

// ClusterSupervisor finds clusters pending work and effects the required changes.
//
// The degree of parallelism is controlled by a weighted semaphore, intended to be shared with
// other clients needing to coordinate background jobs.
type ClusterSupervisor struct {
	store          clusterStore
	provisioner    ClusterProvisionerOption
	eventsProducer eventProducer
	instanceID     string
	metrics        *metrics.CloudMetrics
	logger         log.FieldLogger
}

// NewClusterSupervisor creates a new ClusterSupervisor.
func NewClusterSupervisor(store clusterStore, provisioner ClusterProvisionerOption, eventProducer eventProducer, instanceID string, logger log.FieldLogger, metrics *metrics.CloudMetrics) *ClusterSupervisor {
	return &ClusterSupervisor{
		store:          store,
		provisioner:    provisioner,
		eventsProducer: eventProducer,
		instanceID:     instanceID,
		metrics:        metrics,
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

	// Sort the clusters by state preference. Relative order is preserved.
	sort.SliceStable(clusters, func(i, j int) bool {
		return model.ClusterStateWorkPriority[clusters[i].State] > model.ClusterStateWorkPriority[clusters[j].State]
	})

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
		logger.WithError(err).Warnf("Failed to get cluster and thus persist state %s", newState)
		return
	}

	if cluster.State == newState {
		return
	}

	oldState := cluster.State
	cluster.State = newState
	err = s.store.UpdateCluster(cluster)
	if err != nil {
		logger.WithError(err).Warnf("Failed to set cluster state to %s", newState)
		return
	}

	err = s.processClusterMetrics(cluster, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to process cluster metrics")
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
	case model.ClusterStateWaitingForNodes:
		return s.checkNodesCreated(cluster, logger)
	case model.ClusterStateProvisionInProgress:
		return s.provisionCluster(cluster, logger)
	case model.ClusterStateProvisioningRequested:
		return s.provisionCluster(cluster, logger)
	case model.ClusterStateUpgradeRequested:
		return s.upgradeCluster(cluster, logger)
	case model.ClusterStateResizeRequested:
		return s.resizeCluster(cluster, logger)
	case model.ClusterStateNodegroupsCreationRequested:
		return s.createNodegroups(cluster, logger)
	case model.ClusterStateNodegroupsDeletionRequested:
		return s.deleteNodegroups(cluster, logger)
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

	if s.provisioner.GetClusterProvisioner(cluster.Provisioner).PrepareCluster(cluster) {
		err = s.store.UpdateCluster(cluster)
		if err != nil {
			logger.WithError(err).Error("Failed to record updated cluster after creation")
			return model.ClusterStateCreationFailed
		}
	}

	err = s.provisioner.GetClusterProvisioner(cluster.Provisioner).CreateCluster(cluster)
	if err != nil {
		logger.WithError(err).Error("Failed to create cluster")
		return model.ClusterStateCreationFailed
	}

	logger.Info("Finished creating cluster")
	return s.checkClusterCreated(cluster, logger)
}

func (s *ClusterSupervisor) provisionCluster(cluster *model.Cluster, logger log.FieldLogger) string {
	err := s.provisioner.GetClusterProvisioner(cluster.Provisioner).ProvisionCluster(cluster)
	if err != nil {
		logger.WithError(err).Error("Failed to provision cluster")
		return model.ClusterStateProvisioningFailed
	}

	logger.Info("Finished provisioning cluster")
	return s.refreshClusterMetadata(cluster, logger)
}

func (s *ClusterSupervisor) upgradeCluster(cluster *model.Cluster, logger log.FieldLogger) string {
	err := s.provisioner.GetClusterProvisioner(cluster.Provisioner).UpgradeCluster(cluster)
	if err != nil {
		logger.WithError(err).Error("Failed to upgrade cluster")
		logger.Info("Updating cluster metadata to reflect upgrade failure")
		err = s.store.UpdateCluster(cluster)
		if err != nil {
			logger.WithError(err).Error("Failed to update cluster metadata to reflect upgrade failure")
			return model.ClusterStateRefreshMetadata
		}
		return model.ClusterStateUpgradeFailed
	}

	logger.Info("Finished upgrading cluster")
	return s.refreshClusterMetadata(cluster, logger)
}

func (s *ClusterSupervisor) resizeCluster(cluster *model.Cluster, logger log.FieldLogger) string {
	err := s.provisioner.GetClusterProvisioner(cluster.Provisioner).ResizeCluster(cluster)
	if err != nil {
		logger.WithError(err).Error("Failed to resize cluster")
		return model.ClusterStateResizeFailed
	}

	logger.Info("Finished resizing cluster")
	return s.refreshClusterMetadata(cluster, logger)
}

func (s *ClusterSupervisor) createNodegroups(cluster *model.Cluster, logger log.FieldLogger) string {
	err := s.provisioner.GetClusterProvisioner(cluster.Provisioner).CreateNodegroups(cluster)
	if err != nil {
		logger.WithError(err).Error("Failed to create nodegroups")
		return model.ClusterStateNodegroupsCreationFailed
	}

	_, err = s.provisioner.GetClusterProvisioner(cluster.Provisioner).CheckNodegroupsCreated(cluster)
	if err != nil {
		logger.WithError(err).Error("Failed to create nodegroups")
		return model.ClusterStateNodegroupsCreationFailed
	}

	logger.Info("Finished creating nodegroups")
	return s.refreshClusterMetadata(cluster, logger)
}

func (s *ClusterSupervisor) deleteNodegroups(cluster *model.Cluster, logger log.FieldLogger) string {
	err := s.provisioner.GetClusterProvisioner(cluster.Provisioner).DeleteNodegroups(cluster)
	if err != nil {
		logger.WithError(err).Error("Failed to delete nodegroups")
		return model.ClusterStateNodegroupsDeletionFailed
	}

	logger.Info("Finished deleting nodegroups")
	return s.refreshClusterMetadata(cluster, logger)
}

func (s *ClusterSupervisor) refreshClusterMetadata(cluster *model.Cluster, logger log.FieldLogger) string {

	err := s.provisioner.GetClusterProvisioner(cluster.Provisioner).RefreshClusterMetadata(cluster)
	if err != nil {
		logger.WithError(err).Error("Failed to refresh cluster metadata")
		return model.ClusterStateRefreshMetadata
	}

	err = s.store.UpdateCluster(cluster)
	if err != nil {
		logger.WithError(err).Error("Failed to update cluster metadata in store")
		return model.ClusterStateRefreshMetadata
	}

	return model.ClusterStateStable
}

func (s *ClusterSupervisor) deleteCluster(cluster *model.Cluster, logger log.FieldLogger) string {
	deleted, err := s.provisioner.GetClusterProvisioner(cluster.Provisioner).DeleteCluster(cluster)
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
		logger.WithError(err).Error("Failed to delete cluster from store")
		return model.ClusterStateDeletionFailed
	}

	logger.Info("Finished deleting cluster")
	return model.ClusterStateDeleted
}

func (s *ClusterSupervisor) checkClusterCreated(cluster *model.Cluster, logger log.FieldLogger) string {
	ready, err := s.provisioner.GetClusterProvisioner(cluster.Provisioner).CheckClusterCreated(cluster)
	if err != nil {
		logger.WithError(err).Error("Failed to check if cluster creation finished")
		return model.ClusterStateCreationFailed
	}
	if !ready {
		logger.Debug("Cluster is not yet ready")
		return model.ClusterStateCreationInProgress
	}

	err = s.provisioner.GetClusterProvisioner(cluster.Provisioner).CreateNodegroups(cluster)
	if err != nil {
		logger.WithError(err).Error("Failed to create cluster nodegroups")
		return model.ClusterStateCreationFailed
	}

	return s.checkNodesCreated(cluster, logger)
}

func (s *ClusterSupervisor) checkNodesCreated(cluster *model.Cluster, logger log.FieldLogger) string {

	ready, err := s.provisioner.GetClusterProvisioner(cluster.Provisioner).CheckNodegroupsCreated(cluster)
	if err != nil {
		logger.WithError(err).Error("Failed to check if nodegroups creation finished")
		return model.ClusterStateCreationFailed
	}
	if !ready {
		logger.Debug("Cluster nodegroups are not ready yet")
		return model.ClusterStateWaitingForNodes
	}

	return model.ClusterStateProvisionInProgress
}

func (s *ClusterSupervisor) processClusterMetrics(cluster *model.Cluster, logger log.FieldLogger) error {

	if cluster.State != model.ClusterStateStable && cluster.State != model.ClusterStateDeleted {
		return nil
	}

	// Get the latest event of a 'requested' type to emit the correct metrics.
	events, err := s.store.GetStateChangeEvents(&model.StateChangeEventFilter{
		ResourceID:   cluster.ID,
		ResourceType: model.TypeCluster,
		NewStates:    model.AllClusterRequestStates,
		Paging:       model.Paging{Page: 0, PerPage: 1, IncludeDeleted: false},
	})
	if err != nil {
		return errors.Wrap(err, "failed to get state change events")
	}
	if len(events) != 1 {
		return errors.Errorf("expected 1 state change event, but got %d", len(events))
	}

	event := events[0]
	elapsedSeconds := model.ElapsedTimeInSeconds(event.Event.Timestamp)

	switch event.StateChange.NewState {
	case model.ClusterStateCreationRequested:
		s.metrics.ClusterCreationDurationHist.WithLabelValues().Observe(elapsedSeconds)
		logger.Debugf("Cluster was created in %d seconds", int(elapsedSeconds))
	case model.ClusterStateUpgradeRequested:
		s.metrics.ClusterUpgradeDurationHist.WithLabelValues().Observe(elapsedSeconds)
		logger.Debugf("Cluster was upgraded in %d seconds", int(elapsedSeconds))
	case model.ClusterStateProvisioningRequested:
		s.metrics.ClusterProvisioningDurationHist.WithLabelValues().Observe(elapsedSeconds)
		logger.Debugf("Cluster was provisioned in %d seconds", int(elapsedSeconds))
	case model.ClusterStateResizeRequested:
		s.metrics.ClusterResizeDurationHist.WithLabelValues().Observe(elapsedSeconds)
		logger.Debugf("Cluster was resized in %d seconds", int(elapsedSeconds))
	case model.ClusterStateDeletionRequested:
		s.metrics.ClusterDeletionDurationHist.WithLabelValues().Observe(elapsedSeconds)
		logger.Debugf("Cluster was deleted in %d seconds", int(elapsedSeconds))
	case model.ClusterStateNodegroupsCreationRequested:
		s.metrics.ClusterNodegroupsCreationDurationHist.WithLabelValues().Observe(elapsedSeconds)
		logger.Debugf("Cluster nodegroups were created in %d seconds", int(elapsedSeconds))
	case model.ClusterStateNodegroupsDeletionRequested:
		s.metrics.ClusterNodegroupsDeletionDurationHist.WithLabelValues().Observe(elapsedSeconds)
		logger.Debugf("Cluster nodegroups were deleted in %d seconds", int(elapsedSeconds))
	default:
		return errors.Errorf("failed to handle event %s with new state %s", event.Event.ID, event.StateChange.NewState)
	}

	return nil
}
