package supervisor

import (
	"github.com/mattermost/mattermost-cloud/internal/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
)

// clusterStore abstracts the database operations required to query clusters.
type clusterStore interface {
	GetCluster(clusterID string) (*model.Cluster, error)
	GetUnlockedClusterPendingWork() (*model.Cluster, error)
	GetClusters(clusterFilter *model.ClusterFilter) ([]*model.Cluster, error)
	UpdateCluster(cluster *model.Cluster) error
	LockCluster(clusterID, lockerID string) (bool, error)
	UnlockCluster(clusterID string, lockerID string, force bool) (bool, error)
	DeleteCluster(clusterID string) error
}

// clusterProvisioner abstracts the provisioning operations required by the cluster supervisor.
type clusterProvisioner interface {
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
	workers     *semaphore.Weighted
	logger      log.FieldLogger
}

// NewClusterSupervisor creates a new ClusterSupervisor.
func NewClusterSupervisor(store clusterStore, clusterProvisioner clusterProvisioner, workers *semaphore.Weighted, logger log.FieldLogger) *ClusterSupervisor {
	return &ClusterSupervisor{
		store:       store,
		provisioner: clusterProvisioner,
		workers:     workers,
		logger:      logger,
	}
}

// Do looks for work to be done on any pending clusters and attempts to schedule the required work.
func (s *ClusterSupervisor) Do() error {
	for {
		more, err := s.DoOne()
		if err != nil {
			return errors.Wrap(err, "failed to work on cluster")
		}
		if !more {
			break
		}
	}

	return nil
}

// DoOne looks for work to be done on a single cluster, and does it.
func (s *ClusterSupervisor) DoOne() (bool, error) {
	// Reserve a worker for any necessary work.
	if ok := s.workers.TryAcquire(1); !ok {
		return false, nil
	}

	workerStarted := false
	defer func() {
		// Restore the worker if it hasn't actually been used.
		if !workerStarted {
			s.workers.Release(1)
		}
	}()

	// Look for an unlocked cluster in a state that needs to be transitioned.
	cluster, err := s.store.GetUnlockedClusterPendingWork()
	if err != nil {
		return false, errors.Wrap(err, "failed to query for cluster pending work")
	}
	if cluster == nil {
		return false, nil
	}

	workerID := model.NewID()
	logger := s.logger.WithFields(map[string]interface{}{
		"cluster": cluster.ID,
		"worker":  workerID,
	})

	lock := newClusterLock(cluster.ID, workerID, s.store, logger)
	if !lock.TryLock() {
		return false, nil
	}

	workerStarted = true
	go func() {
		defer func() {
			s.workers.Release(1)
			lock.Unlock()
		}()

		newState, err := s.transitionCluster(cluster, logger)
		if err != nil {
			logger.WithError(err).Error("transition cluster failed")
		}

		// Transition the state even if an error occurred, because failure is represented
		// in the states.
		if newState != "" {
			cluster, err := s.store.GetCluster(cluster.ID)
			if err != nil {
				logger.WithError(err).Error("failed to get cluster")
				return
			}

			if newState == cluster.State {
				return
			}

			cluster.State = newState
			err = s.store.UpdateCluster(cluster)
			if err != nil {
				logger.WithError(err).Errorf("failed to set cluster state to %s", newState)
				return
			}
		}
	}()

	return true, nil
}

// Do works with the given cluster to transition it to a final state.
func (s *ClusterSupervisor) transitionCluster(cluster *model.Cluster, logger log.FieldLogger) (string, error) {
	switch cluster.State {
	case model.ClusterStateCreationRequested:
		err := s.provisioner.CreateCluster(cluster)
		if err != nil {
			return model.ClusterStateCreationFailed, errors.Wrap(err, "failed to create cluster")
		}

		err = s.store.UpdateCluster(cluster)
		if err != nil {
			return model.ClusterStateCreationFailed, errors.Wrap(err, "failed to record updated cluster after creation")
		}

		return model.ClusterStateStable, nil

	case model.ClusterStateUpgradeRequested:
		err := s.provisioner.UpgradeCluster(cluster)
		if err != nil {
			return model.ClusterStateDeletionFailed, errors.Wrap(err, "failed to delete cluster")
		}

		return model.ClusterStateStable, nil

	case model.ClusterStateDeletionRequested:
		err := s.provisioner.DeleteCluster(cluster)
		if err != nil {
			return model.ClusterStateDeletionFailed, errors.Wrap(err, "failed to delete cluster")
		}

		err = s.store.DeleteCluster(cluster.ID)
		if err != nil {
			return model.ClusterStateDeletionFailed, errors.Wrap(err, "failed to mark cluster as deleted")
		}

		return model.ClusterStateDeleted, nil

	default:
		logger.Warnf("found cluster pending work in unexpected state %s", cluster.State)
		return "", nil
	}
}
