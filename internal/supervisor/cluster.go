package supervisor

import (
	"github.com/mattermost/mattermost-cloud/internal/model"
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
}

// clusterProvisioner abstracts the provisioning operations required by the cluster supervisor.
type clusterProvisioner interface {
	PrepareCluster(cluster *model.Cluster) (bool, error)
	CreateCluster(cluster *model.Cluster) error
	UpgradeCluster(cluster *model.Cluster) error
	DeleteCluster(cluster *model.Cluster) error
}

// ClusterSupervisor finds clusters pending work and effects the required changes.
//
// The degree of parallelism is controlled by a weighted semaphore, intended to be shared with
// other clients needing to coordinate background jobs.
type ClusterSupervisor struct {
	store       clusterStore
	provisioner clusterProvisioner
	instanceID  string
	logger      log.FieldLogger
}

// NewClusterSupervisor creates a new ClusterSupervisor.
func NewClusterSupervisor(store clusterStore, clusterProvisioner clusterProvisioner, instanceID string, logger log.FieldLogger) *ClusterSupervisor {
	return &ClusterSupervisor{
		store:       store,
		provisioner: clusterProvisioner,
		instanceID:  instanceID,
		logger:      logger,
	}
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
	logger := s.logger.WithFields(map[string]interface{}{
		"cluster": cluster.ID,
	})

	lock := newClusterLock(cluster.ID, s.instanceID, s.store, logger)
	if !lock.TryLock() {
		return
	}
	defer lock.Unlock()

	logger.Debugf("Supervising cluster in state %s", cluster.State)

	newState := s.transitionCluster(cluster, logger)

	cluster, err := s.store.GetCluster(cluster.ID)
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

	logger.Debugf("Transitioned cluster from %s to %s", oldState, newState)
}

// Do works with the given cluster to transition it to a final state.
func (s *ClusterSupervisor) transitionCluster(cluster *model.Cluster, logger log.FieldLogger) string {
	switch cluster.State {
	case model.ClusterStateCreationRequested:
		changed, err := s.provisioner.PrepareCluster(cluster)
		if err != nil {
			logger.WithError(err).Error("Failed to prepare cluster")
			return model.ClusterStateCreationFailed
		}

		if changed {
			err = s.store.UpdateCluster(cluster)
			if err != nil {
				logger.WithError(err).Error("Failed to record updated cluster after creation")
				return model.ClusterStateCreationFailed
			}
		}

		err = s.provisioner.CreateCluster(cluster)
		if err != nil {
			logger.WithError(err).Error("Failed to create cluster")
			return model.ClusterStateCreationFailed
		}

		logger.Info("Finished creating cluster")
		return model.ClusterStateStable

	case model.ClusterStateUpgradeRequested:
		err := s.provisioner.UpgradeCluster(cluster)
		if err != nil {
			logger.WithError(err).Error("Failed to upgrade cluster")
			return model.ClusterStateUpgradeFailed
		}

		logger.Info("Finished upgrading cluster")
		return model.ClusterStateStable

	case model.ClusterStateDeletionRequested:
		err := s.provisioner.DeleteCluster(cluster)
		if err != nil {
			logger.WithError(err).Error("Failed to delete cluster")
			return model.ClusterStateDeletionFailed
		}

		err = s.store.DeleteCluster(cluster.ID)
		if err != nil {
			logger.WithError(err).Error("Failed to record updated cluster after deletion")
			return model.ClusterStateDeletionFailed
		}

		logger.Info("Finished deleting cluster")
		return model.ClusterStateDeleted

	default:
		logger.Warnf("Found cluster pending work in unexpected state %s", cluster.State)
		return cluster.State
	}
}
