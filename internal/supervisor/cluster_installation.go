package supervisor

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/webhook"
	"github.com/mattermost/mattermost-cloud/model"

	mmv1alpha1 "github.com/mattermost/mattermost-operator/pkg/apis/mattermost/v1alpha1"
)

// clusterInstallationStore abstracts the database operations required to query installations.
type clusterInstallationStore interface {
	GetCluster(clusterID string) (*model.Cluster, error)

	GetInstallation(installationID string) (*model.Installation, error)

	GetClusterInstallation(clusterInstallationID string) (*model.ClusterInstallation, error)
	GetUnlockedClusterInstallationsPendingWork() ([]*model.ClusterInstallation, error)
	LockClusterInstallations(clusterInstallationID []string, lockerID string) (bool, error)
	UnlockClusterInstallations(clusterInstallationID []string, lockerID string, force bool) (bool, error)
	UpdateClusterInstallation(clusterInstallation *model.ClusterInstallation) error
	DeleteClusterInstallation(clusterInstallationID string) error

	GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error)
}

// provisioner abstracts the provisioning operations required by the cluster installation supervisor.
type clusterInstallationProvisioner interface {
	CreateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation, awsClient aws.AWS) error
	DeleteClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error
	UpdateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error
	GetClusterInstallationResource(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) (*mmv1alpha1.ClusterInstallation, error)
}

// ClusterInstallationSupervisor finds cluster installations pending work and effects the required changes.
//
// The degree of parallelism is controlled by a weighted semaphore, intended to be shared with
// other clients needing to coordinate background jobs.
type ClusterInstallationSupervisor struct {
	store       clusterInstallationStore
	provisioner clusterInstallationProvisioner
	aws         aws.AWS
	instanceID  string
	logger      log.FieldLogger
}

// NewClusterInstallationSupervisor creates a new ClusterInstallationSupervisor.
func NewClusterInstallationSupervisor(store clusterInstallationStore, clusterInstallationProvisioner clusterInstallationProvisioner, aws aws.AWS, instanceID string, logger log.FieldLogger) *ClusterInstallationSupervisor {
	return &ClusterInstallationSupervisor{
		store:       store,
		provisioner: clusterInstallationProvisioner,
		aws:         aws,
		instanceID:  instanceID,
		logger:      logger,
	}
}

// Do looks for work to be done on any pending cluster installations and attempts to schedule the required work.
func (s *ClusterInstallationSupervisor) Do() error {
	clusterInstallations, err := s.store.GetUnlockedClusterInstallationsPendingWork()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to query for cluster installations pending work")
		return nil
	}

	for _, clusterInstallation := range clusterInstallations {
		s.Supervise(clusterInstallation)
	}

	return nil
}

// Supervise schedules the required work on the given cluster installation.
func (s *ClusterInstallationSupervisor) Supervise(clusterInstallation *model.ClusterInstallation) {
	logger := s.logger.WithFields(log.Fields{
		"clusterInstallation": clusterInstallation.ID,
	})

	lock := newClusterInstallationLock(clusterInstallation.ID, s.instanceID, s.store, logger)
	if !lock.TryLock() {
		return
	}
	defer lock.Unlock()

	logger.Debugf("Supervising cluster installation in state %s", clusterInstallation.State)

	newState := s.transitionClusterInstallation(clusterInstallation, logger)

	clusterInstallation, err := s.store.GetClusterInstallation(clusterInstallation.ID)
	if err != nil {
		logger.WithError(err).Warnf("failed to get cluster installation and thus persist state %s", newState)
		return
	}

	if clusterInstallation.State == newState {
		return
	}

	oldState := clusterInstallation.State
	clusterInstallation.State = newState
	err = s.store.UpdateClusterInstallation(clusterInstallation)
	if err != nil {
		logger.WithError(err).Errorf("failed to set cluster installation state to %s", newState)
		return
	}

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeClusterInstallation,
		ID:        clusterInstallation.ID,
		NewState:  newState,
		OldState:  oldState,
		Timestamp: time.Now().UnixNano(),
		ExtraData: map[string]string{"ClusterID": clusterInstallation.ClusterID},
	}
	err = webhook.SendToAllWebhooks(s.store, webhookPayload, logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		logger.WithError(err).Error("Unable to process and send webhooks")
	}

	logger.Debugf("Transitioned cluster installation from %s to %s", oldState, newState)
}

func failedClusterInstallationState(state string) string {
	switch state {
	case model.ClusterInstallationStateCreationRequested:
		return model.ClusterInstallationStateCreationFailed
	case model.ClusterInstallationStateDeletionRequested:
		return model.ClusterInstallationStateDeletionFailed

	default:
		return state
	}
}

// transitionClusterInstallation works with the given cluster installation to transition it to a final state.
func (s *ClusterInstallationSupervisor) transitionClusterInstallation(clusterInstallation *model.ClusterInstallation, logger log.FieldLogger) string {
	cluster, err := s.store.GetCluster(clusterInstallation.ClusterID)
	if err != nil {
		logger.WithError(err).Warnf("Failed to query cluster %s", clusterInstallation.ClusterID)
		return clusterInstallation.State
	}
	if cluster == nil {
		logger.Errorf("Failed to find cluster %s", clusterInstallation.ClusterID)
		return failedClusterInstallationState(clusterInstallation.State)
	}

	installation, err := s.store.GetInstallation(clusterInstallation.InstallationID)
	if err != nil {
		logger.WithError(err).Warnf("Failed to query installation %s", clusterInstallation.InstallationID)
		return clusterInstallation.State
	}
	if installation == nil {
		logger.Errorf("Failed to find installation %s", clusterInstallation.InstallationID)
		return failedClusterInstallationState(clusterInstallation.State)
	}

	switch clusterInstallation.State {
	case model.ClusterInstallationStateCreationRequested:
		err = s.provisioner.CreateClusterInstallation(cluster, installation, clusterInstallation, s.aws)
		if err != nil {
			logger.WithError(err).Error("Failed to provision cluster installation")
			return model.ClusterInstallationStateCreationRequested
		}

		err = s.store.UpdateClusterInstallation(clusterInstallation)
		if err != nil {
			logger.WithError(err).Error("Failed to record updated cluster installation after provisioning")
			return model.ClusterInstallationStateCreationFailed
		}

		logger.Info("Finished creating cluster installation")
		return model.ClusterInstallationStateReconciling

	case model.ClusterInstallationStateDeletionRequested:
		err = s.provisioner.DeleteClusterInstallation(cluster, installation, clusterInstallation)
		if err != nil {
			logger.WithError(err).Error("Failed to delete cluster installation")
			return model.ClusterInstallationStateDeletionFailed
		}

		err = s.store.DeleteClusterInstallation(clusterInstallation.ID)
		if err != nil {
			logger.WithError(err).Error("Failed to record deleted cluster installation after deletion")
			return model.ClusterStateDeletionFailed
		}

		logger.Info("Finished deleting cluster installation")
		return model.ClusterInstallationStateDeleted

	case model.ClusterInstallationStateReconciling:
		cr, err := s.provisioner.GetClusterInstallationResource(cluster, installation, clusterInstallation)
		if err != nil {
			logger.WithError(err).Error("Failed to get cluster installation resource")
			return model.ClusterInstallationStateReconciling
		}

		if cr.Status.State != mmv1alpha1.Stable {
			logger.Info("Cluster installation is still reconciling")
			return model.ClusterInstallationStateReconciling
		}

		logger.Info("Cluster installation finished reconciling")
		return model.ClusterInstallationStateStable

	default:
		logger.Warnf("Found cluster installation pending work in unexpected state %s", clusterInstallation.State)
		return clusterInstallation.State
	}
}
