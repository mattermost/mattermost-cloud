package provisioner

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
	UpdateCluster(cluster *model.Cluster) error
	LockCluster(clusterID, lockerID string) (bool, error)
	UnlockCluster(clusterID string, lockerID string, force bool) (bool, error)
	DeleteCluster(clusterID string) error
}

// provisioner abstracts the provisioning operations required by the cluster supervisor.
type provisioner interface {
	CreateCluster(cluster *model.Cluster) error
	UpgradeCluster(cluster *model.Cluster) error
	DeleteCluster(cluster *model.Cluster) error
}

// Supervisor finds clusters pending work and effects the required changes.
//
// The degree of parallelism is controlled by a weighted semaphore, intended to be shared with
// other clients needing to coordinate background jobs.
type Supervisor struct {
	clusterStore clusterStore
	provisioner  provisioner
	workers      *semaphore.Weighted
	logger       log.FieldLogger
}

// NewSupervisor creates a new Supervisor.
func NewSupervisor(clusterStore clusterStore, provisioner provisioner, workers *semaphore.Weighted, logger log.FieldLogger) *Supervisor {
	return &Supervisor{
		clusterStore: clusterStore,
		provisioner:  provisioner,
		workers:      workers,
		logger:       logger,
	}
}

// Do looks for work to be done on any pending clusters and attempts to schedule the required work.
func (s *Supervisor) Do() error {
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
func (s *Supervisor) DoOne() (bool, error) {
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
	cluster, err := s.clusterStore.GetUnlockedClusterPendingWork()
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

	// Attempt to lock the cluster. There's a chance another provisioning server will find
	// the same cluster needing work and lock it first.
	locked, err := s.clusterStore.LockCluster(cluster.ID, workerID)
	if err != nil {
		return false, errors.Wrap(err, "failed to lock cluster")
	}
	if !locked {
		return false, nil
	}

	workerStarted = true
	go func() {
		defer func() {
			s.workers.Release(1)

			unlocked, err := s.clusterStore.UnlockCluster(cluster.ID, workerID, false)
			if err != nil {
				logger.WithError(err).Error("failed to unlock cluster")
			} else if unlocked != true {
				logger.Error("failed to release lock for cluster")
			}
		}()

		newState, err := s.transitionCluster(cluster, logger)
		if err != nil {
			logger.WithError(err).Error("transition cluster failed")
		}

		// Transition the state even if an error occurred.
		if newState != "" {
			cluster, err := s.clusterStore.GetCluster(cluster.ID)
			if err != nil {
				logger.WithError(err).Error("failed to get cluster")
				return
			}

			cluster.State = newState
			err = s.clusterStore.UpdateCluster(cluster)
			if err != nil {
				logger.WithError(err).Errorf("failed to set cluster state to %s", newState)
				return
			}
		}
	}()

	return true, nil
}

// Do works with the given cluster to transition it to a final state.
func (s *Supervisor) transitionCluster(cluster *model.Cluster, logger log.FieldLogger) (string, error) {
	switch cluster.State {
	case model.ClusterStateCreationRequested:
		err := s.provisioner.CreateCluster(cluster)
		if err != nil {
			return model.ClusterStateCreationFailed, errors.Wrap(err, "failed to create cluster")
		}

		err = s.clusterStore.UpdateCluster(cluster)
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

		err = s.clusterStore.DeleteCluster(cluster.ID)
		if err != nil {
			return model.ClusterStateDeletionFailed, errors.Wrap(err, "failed to mark cluster as deleted")
		}

		return model.ClusterStateDeleted, nil
	}

	logger.Warnf("found cluster pending work in unexpected state %s", cluster.State)
	return "", nil
}
