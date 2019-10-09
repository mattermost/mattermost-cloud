package supervisor

import (
	"time"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/k8s"
	"github.com/mattermost/mattermost-cloud/internal/webhook"
	"github.com/mattermost/mattermost-cloud/model"
	mmv1alpha1 "github.com/mattermost/mattermost-operator/pkg/apis/mattermost/v1alpha1"
	log "github.com/sirupsen/logrus"
)

// installationStore abstracts the database operations required to query installations.
type installationStore interface {
	GetClusters(clusterFilter *model.ClusterFilter) ([]*model.Cluster, error)
	GetCluster(id string) (*model.Cluster, error)
	LockCluster(clusterID, lockerID string) (bool, error)
	UnlockCluster(clusterID string, lockerID string, force bool) (bool, error)

	GetInstallation(installationID string) (*model.Installation, error)
	GetUnlockedInstallationsPendingWork() ([]*model.Installation, error)
	UpdateInstallation(installation *model.Installation) error
	LockInstallation(installationID, lockerID string) (bool, error)
	UnlockInstallation(installationID, lockerID string, force bool) (bool, error)
	DeleteInstallation(installationID string) error

	CreateClusterInstallation(clusterInstallation *model.ClusterInstallation) error
	GetClusterInstallation(clusterInstallationID string) (*model.ClusterInstallation, error)
	GetClusterInstallations(*model.ClusterInstallationFilter) ([]*model.ClusterInstallation, error)
	LockClusterInstallations(clusterInstallationID []string, lockerID string) (bool, error)
	UnlockClusterInstallations(clusterInstallationID []string, lockerID string, force bool) (bool, error)
	UpdateClusterInstallation(clusterInstallation *model.ClusterInstallation) error

	GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error)
}

// provisioner abstracts the provisioning operations required by the installation supervisor.
type installationProvisioner interface {
	CreateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation, awsClient aws.AWS) error
	DeleteClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error
	UpdateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error
	GetClusterInstallationResource(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) (*mmv1alpha1.ClusterInstallation, error)
	GetClusterResources(cluster *model.Cluster, onlySchedulable bool) (*k8s.ClusterResources, error)
}

// InstallationSupervisor finds installations pending work and effects the required changes.
//
// The degree of parallelism is controlled by a weighted semaphore, intended to be shared with
// other clients needing to coordinate background jobs.
type InstallationSupervisor struct {
	store                    installationStore
	provisioner              installationProvisioner
	aws                      aws.AWS
	instanceID               string
	clusterResourceThreshold int
	keepFilestoreData        bool
	logger                   log.FieldLogger
}

// NewInstallationSupervisor creates a new InstallationSupervisor.
func NewInstallationSupervisor(store installationStore, installationProvisioner installationProvisioner, aws aws.AWS, instanceID string, threshold int, keepFilestoreData bool, logger log.FieldLogger) *InstallationSupervisor {
	return &InstallationSupervisor{
		store:                    store,
		provisioner:              installationProvisioner,
		aws:                      aws,
		instanceID:               instanceID,
		clusterResourceThreshold: threshold,
		keepFilestoreData:        keepFilestoreData,
		logger:                   logger,
	}
}

// Do looks for work to be done on any pending installations and attempts to schedule the required work.
func (s *InstallationSupervisor) Do() error {
	installations, err := s.store.GetUnlockedInstallationsPendingWork()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to query for installation pending work")
		return nil
	}

	for _, installation := range installations {
		s.Supervise(installation)
	}

	return nil
}

// Supervise schedules the required work on the given installation.
func (s *InstallationSupervisor) Supervise(installation *model.Installation) {
	logger := s.logger.WithFields(map[string]interface{}{
		"installation": installation.ID,
	})

	lock := newInstallationLock(installation.ID, s.instanceID, s.store, logger)
	if !lock.TryLock() {
		return
	}
	defer lock.Unlock()

	logger.Debugf("Supervising installation in state %s", installation.State)

	newState := s.transitionInstallation(installation, s.instanceID, logger)

	installation, err := s.store.GetInstallation(installation.ID)
	if err != nil {
		logger.WithError(err).Warnf("failed to get installation and thus persist state %s", newState)
		return
	}

	if installation.State == newState {
		return
	}

	oldState := installation.State
	installation.State = newState
	err = s.store.UpdateInstallation(installation)
	if err != nil {
		logger.WithError(err).Warnf("Failed to set installation state to %s", newState)
		return
	}

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeInstallation,
		ID:        installation.ID,
		NewState:  newState,
		OldState:  oldState,
		Timestamp: time.Now().UnixNano(),
	}
	err = webhook.SendToAllWebhooks(s.store, webhookPayload, logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		logger.WithError(err).Error("Unable to process and send webhooks")
	}

	logger.Debugf("Transitioned installation from %s to %s", oldState, newState)
}

// transitionInstallation works with the given installation to transition it to a final state.
func (s *InstallationSupervisor) transitionInstallation(installation *model.Installation, instanceID string, logger log.FieldLogger) string {
	switch installation.State {
	case model.InstallationStateCreationRequested,
		model.InstallationStateCreationPreProvisioning:
		return s.preProvisionInstallation(installation, instanceID, logger)

	case model.InstallationStateCreationInProgress,
		model.InstallationStateCreationNoCompatibleClusters:
		return s.createInstallation(installation, instanceID, logger)

	case model.InstallationStateCreationDNS:
		return s.configureInstallationDNS(installation, logger)

	case model.InstallationStateUpgradeRequested:
		return s.updateInstallation(installation, instanceID, logger)

	case model.InstallationStateUpgradeInProgress:
		return s.waitForUpdateComplete(installation, instanceID, logger)

	case model.InstallationStateDeletionRequested,
		model.InstallationStateDeletionInProgress:
		return s.deleteInstallation(installation, instanceID, logger)

	case model.InstallationStateDeletionFinalCleanup:
		return s.finalDeletionCleanup(installation, logger)

	default:
		logger.Warnf("Found installation pending work in unexpected state %s", installation.State)
		return installation.State
	}
}

func (s *InstallationSupervisor) preProvisionInstallation(installation *model.Installation, instanceID string, logger log.FieldLogger) string {
	err := installation.GetFilestore().Provision(logger)
	if err != nil {
		logger.WithError(err).Warn("Failed to provision AWS S3 filestore")
		return model.InstallationStateCreationPreProvisioning
	}

	logger.Info("Installation pre-provisioning complete")

	return s.createInstallation(installation, instanceID, logger)
}

func (s *InstallationSupervisor) createInstallation(installation *model.Installation, instanceID string, logger log.FieldLogger) string {
	clusterInstallations, err := s.store.GetClusterInstallations(&model.ClusterInstallationFilter{
		InstallationID: installation.ID,
		PerPage:        model.AllPerPage,
	})
	if err != nil {
		logger.WithError(err).Warn("Failed to find cluster installations")
		return installation.State
	}

	// If we've previously created one or more cluster installations, consider the
	// installation to be stable once all cluster installations are stable. Or, if
	// some cluster installations have failed, mark the installation as failed.
	if len(clusterInstallations) > 0 {
		var stable, reconciling, failed int
		for _, clusterInstallation := range clusterInstallations {
			if clusterInstallation.State == model.ClusterInstallationStateStable {
				stable++
			}
			if clusterInstallation.State == model.ClusterInstallationStateReconciling {
				reconciling++
			}
			if clusterInstallation.State == model.ClusterInstallationStateCreationFailed {
				failed++
			}
		}

		logger.Debugf("Found %d cluster installations, %d stable, %d reconciling, %d failed", len(clusterInstallations), stable, reconciling, failed)

		if len(clusterInstallations) == stable {
			logger.Infof("Finished creating installation")
			return s.configureInstallationDNS(installation, logger)
		}
		if failed > 0 {
			logger.Infof("Found %d failed cluster installations", failed)
			return model.InstallationStateCreationFailed
		}

		return model.InstallationStateCreationInProgress
	}

	// Otherwise proceed to requesting cluster installation creation on any available clusters.
	clusters, err := s.store.GetClusters(&model.ClusterFilter{
		PerPage:        model.AllPerPage,
		IncludeDeleted: false,
	})
	if err != nil {
		logger.WithError(err).Warn("Failed to query clusters")
		return installation.State
	}

	for _, cluster := range clusters {
		clusterInstallation := s.createClusterInstallation(cluster, installation, instanceID, logger)
		if clusterInstallation != nil {
			// Once created, preserve the existing state until the cluster installation
			// stabilizes.
			return model.InstallationStateCreationInProgress
		}
	}

	// TODO: Support creating a cluster on demand if no existing cluster meets the criteria.
	logger.Debug("No compatible clusters available for installation")

	return model.InstallationStateCreationNoCompatibleClusters
}

// createClusterInstallation attempts to schedule a cluster installation onto the given cluster.
func (s *InstallationSupervisor) createClusterInstallation(cluster *model.Cluster, installation *model.Installation, instanceID string, logger log.FieldLogger) *model.ClusterInstallation {
	clusterLock := newClusterLock(cluster.ID, instanceID, s.store, logger)
	if !clusterLock.TryLock() {
		logger.Debugf("Failed to lock cluster %s", cluster.ID)
		return nil
	}
	defer clusterLock.Unlock()

	if cluster.State != model.ClusterStateStable {
		logger.Debugf("Cluster %s is not stable (currently %s)", cluster.ID, cluster.State)
		return nil
	}

	existingClusterInstallations, err := s.store.GetClusterInstallations(&model.ClusterInstallationFilter{
		PerPage:   model.AllPerPage,
		ClusterID: cluster.ID,
	})

	////////////////////////////////////////////////////////////////////////////
	//                              MULTI-TENANCY                             //
	////////////////////////////////////////////////////////////////////////////
	// Current model:                                                         //
	// - isolation=true  | 1 cluster installations                            //
	// - isolation=false | X cluster installations, where "X" is as many as   //
	//                     will fit with the given CPU and Memory threshold.  //
	////////////////////////////////////////////////////////////////////////////
	if installation.Affinity == model.InstallationAffinityIsolated {
		if len(existingClusterInstallations) > 0 {
			logger.Debugf("Cluster %s already has %d installations", cluster.ID, len(existingClusterInstallations))
			return nil
		}
	} else {
		if len(existingClusterInstallations) == 1 {
			// This should be the only scenario where we need to check if the
			// cluster installation running requires isolation or not.
			installation, err := s.store.GetInstallation(existingClusterInstallations[0].InstallationID)
			if err != nil {
				logger.WithError(err).Warn("Unable to find installation")
				return nil
			}
			if installation.Affinity == model.InstallationAffinityIsolated {
				logger.Debugf("Cluster %s already has an isolated installation %s", cluster.ID, installation.ID)
				return nil
			}
		}
	}

	// Begin final resource check.

	size, err := mmv1alpha1.GetClusterSize(installation.Size)
	if err != nil {
		logger.WithError(err).Error("invalid cluster installation size")
		return nil
	}
	clusterResources, err := s.provisioner.GetClusterResources(cluster, true)
	if err != nil {
		logger.WithError(err).Error("failed to get cluster resources")
		return nil
	}

	cpuPercent := clusterResources.CalculateCPUPercentUsed(
		size.CalculateCPUMilliRequirement(
			true,
			installation.InternalFilestore(),
		),
	)
	memoryPercent := clusterResources.CalculateMemoryPercentUsed(
		size.CalculateMemoryMilliRequirement(
			true,
			installation.InternalFilestore(),
		),
	)
	if cpuPercent > s.clusterResourceThreshold || memoryPercent > s.clusterResourceThreshold {
		logger.Debugf("Cluster %s would exceed the cluster load threshold (%d%%): CPU=%d%%, Memory=%d%%", cluster.ID, s.clusterResourceThreshold, cpuPercent, memoryPercent)
		return nil
	}

	// The cluster can support the cluster installation.

	clusterInstallation := &model.ClusterInstallation{
		ClusterID:      cluster.ID,
		InstallationID: installation.ID,
		Namespace:      model.NewID(),
		State:          model.ClusterInstallationStateCreationRequested,
	}

	err = s.store.CreateClusterInstallation(clusterInstallation)
	if err != nil {
		logger.WithError(err).Warn("Failed to create cluster installation")
		return nil
	}

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeClusterInstallation,
		ID:        clusterInstallation.ID,
		NewState:  model.ClusterInstallationStateCreationRequested,
		OldState:  "n/a",
		Timestamp: time.Now().UnixNano(),
	}
	err = webhook.SendToAllWebhooks(s.store, webhookPayload, logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		logger.WithError(err).Error("Unable to process and send webhooks")
	}

	logger.Infof("Requested creation of cluster installation on cluster %s. Expected resource load: CPU=%d%%, Memory=%d%%", cluster.ID, cpuPercent, memoryPercent)

	return clusterInstallation
}

func (s *InstallationSupervisor) configureInstallationDNS(installation *model.Installation, logger log.FieldLogger) string {
	clusterInstallations, err := s.store.GetClusterInstallations(&model.ClusterInstallationFilter{
		InstallationID: installation.ID,
		PerPage:        model.AllPerPage,
	})
	if err != nil {
		logger.WithError(err).Warn("Failed to find cluster installations")
		return model.InstallationStateCreationDNS
	}

	var endpoints []string
	for _, clusterInstallation := range clusterInstallations {
		cluster, err := s.store.GetCluster(clusterInstallation.ClusterID)
		if err != nil {
			logger.WithError(err).Warnf("Failed to query cluster %s", clusterInstallation.ClusterID)
			return model.InstallationStateCreationDNS
		}
		if cluster == nil {
			logger.Errorf("Failed to find cluster %s", clusterInstallation.ClusterID)
			return failedClusterInstallationState(clusterInstallation.State)
		}

		cr, err := s.provisioner.GetClusterInstallationResource(cluster, installation, clusterInstallation)
		if err != nil {
			logger.WithError(err).Error("Failed to get cluster installation resource")
			return model.InstallationStateCreationDNS
		}

		endpoints = append(endpoints, cr.Status.Endpoint)
	}

	err = s.aws.CreateCNAME(installation.DNS, endpoints, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to create DNS CNAME record")
		return model.InstallationStateCreationDNS
	}

	logger.Infof("Successfully configured DNS %s", installation.DNS)

	return model.InstallationStateStable
}

func (s *InstallationSupervisor) updateInstallation(installation *model.Installation, instanceID string, logger log.FieldLogger) string {
	clusterInstallations, err := s.store.GetClusterInstallations(&model.ClusterInstallationFilter{
		PerPage:        model.AllPerPage,
		InstallationID: installation.ID,
	})
	if err != nil {
		logger.WithError(err).Warn("Failed to find cluster installations")
		return installation.State
	}

	var clusterInstallationIDs []string
	if len(clusterInstallations) > 0 {
		for _, clusterInstallation := range clusterInstallations {
			clusterInstallationIDs = append(clusterInstallationIDs, clusterInstallation.ID)
		}

		clusterInstallationLocks := newClusterInstallationLocks(clusterInstallationIDs, instanceID, s.store, logger)
		if !clusterInstallationLocks.TryLock() {
			logger.Debugf("Failed to lock %d cluster installations", len(clusterInstallations))
			return installation.State
		}
		defer clusterInstallationLocks.Unlock()

		// Fetch the same cluster installations again, now that we have the locks.
		clusterInstallations, err = s.store.GetClusterInstallations(&model.ClusterInstallationFilter{
			PerPage: model.AllPerPage,
			IDs:     clusterInstallationIDs,
		})
		if err != nil {
			logger.WithError(err).Warnf("Failed to fetch %d cluster installations by ids", len(clusterInstallations))
			return installation.State
		}

		if len(clusterInstallations) != len(clusterInstallationIDs) {
			logger.Warnf("Found only %d cluster installations after locking, expected %d", len(clusterInstallations), len(clusterInstallationIDs))
		}
	}

	for _, clusterInstallation := range clusterInstallations {
		cluster, err := s.store.GetCluster(clusterInstallation.ClusterID)
		if err != nil {
			logger.WithError(err).Warnf("Failed to query cluster %s", clusterInstallation.ClusterID)
			return clusterInstallation.State
		}
		if cluster == nil {
			logger.Errorf("Failed to find cluster %s", clusterInstallation.ClusterID)
			return failedClusterInstallationState(clusterInstallation.State)
		}

		err = s.provisioner.UpdateClusterInstallation(cluster, installation, clusterInstallation)
		if err != nil {
			logger.Error("Failed to update cluster installation")
			return installation.State
		}

		clusterInstallation.State = model.ClusterInstallationStateReconciling
		err = s.store.UpdateClusterInstallation(clusterInstallation)
		if err != nil {
			logger.Errorf("Failed to change cluster installation state to %s", model.ClusterInstallationStateReconciling)
			return installation.State
		}
	}

	logger.Infof("Finished updating clusters installations")

	return model.InstallationStateUpgradeInProgress
}

func (s *InstallationSupervisor) waitForUpdateComplete(installation *model.Installation, instanceID string, logger log.FieldLogger) string {
	clusterInstallations, err := s.store.GetClusterInstallations(&model.ClusterInstallationFilter{
		InstallationID: installation.ID,
		PerPage:        model.AllPerPage,
	})
	if err != nil {
		logger.WithError(err).Warn("Failed to find cluster installations")
		return installation.State
	}

	var stable, reconciling, failed int
	for _, clusterInstallation := range clusterInstallations {
		if clusterInstallation.State == model.ClusterInstallationStateStable {
			stable++
		}
		if clusterInstallation.State == model.ClusterInstallationStateReconciling {
			reconciling++
		}
		if clusterInstallation.State == model.ClusterInstallationStateCreationFailed {
			failed++
		}
	}

	logger.Debugf("Found %d cluster installations, %d stable, %d reconciling, %d failed", len(clusterInstallations), stable, reconciling, failed)

	if len(clusterInstallations) == stable {
		logger.Infof("Finished updating installation")
		return model.InstallationStateStable
	}
	if failed > 0 {
		logger.Infof("Found %d failed cluster installations", failed)
		return model.InstallationStateUpgradeFailed
	}

	return installation.State
}

func (s *InstallationSupervisor) deleteInstallation(installation *model.Installation, instanceID string, logger log.FieldLogger) string {
	clusterInstallations, err := s.store.GetClusterInstallations(&model.ClusterInstallationFilter{
		PerPage:        model.AllPerPage,
		InstallationID: installation.ID,
		IncludeDeleted: true,
	})
	if err != nil {
		logger.WithError(err).Warn("Failed to find cluster installations")
		return installation.State
	}

	var clusterInstallationIDs []string
	if len(clusterInstallations) > 0 {
		for _, clusterInstallation := range clusterInstallations {
			clusterInstallationIDs = append(clusterInstallationIDs, clusterInstallation.ID)
		}

		clusterInstallationLocks := newClusterInstallationLocks(clusterInstallationIDs, instanceID, s.store, logger)
		if !clusterInstallationLocks.TryLock() {
			logger.Debugf("Failed to lock %d cluster installations", len(clusterInstallations))
			return installation.State
		}
		defer clusterInstallationLocks.Unlock()

		// Fetch the same cluster installations again, now that we have the locks.
		clusterInstallations, err = s.store.GetClusterInstallations(&model.ClusterInstallationFilter{
			PerPage:        model.AllPerPage,
			IDs:            clusterInstallationIDs,
			IncludeDeleted: true,
		})
		if err != nil {
			logger.WithError(err).Warnf("Failed to fetch %d cluster installations by ids", len(clusterInstallations))
			return installation.State
		}

		if len(clusterInstallations) != len(clusterInstallationIDs) {
			logger.Warnf("Found only %d cluster installations after locking, expected %d", len(clusterInstallations), len(clusterInstallationIDs))
		}
	}

	deletingClusterInstallations := 0
	deletedClusterInstallations := 0
	failedClusterInstallations := 0

	for _, clusterInstallation := range clusterInstallations {
		switch clusterInstallation.State {
		case model.ClusterInstallationStateCreationRequested:
		case model.ClusterInstallationStateCreationFailed:
		case model.ClusterInstallationStateReconciling:

		case model.ClusterInstallationStateDeletionRequested:
			deletingClusterInstallations++
			continue

		case model.ClusterInstallationStateDeletionFailed:
			// Only count failed cluster installations if the deletion is in
			// progress.
			if installation.State == model.InstallationStateDeletionInProgress {
				failedClusterInstallations++
				continue
			}

			// Otherwise, we try the deletion again below.

		case model.ClusterInstallationStateDeleted:
			deletedClusterInstallations++
			continue

		case model.ClusterInstallationStateStable:

		default:
			logger.Errorf("Cannot delete installation with cluster installation in state %s", clusterInstallation.State)
			return model.InstallationStateDeletionFailed
		}

		clusterInstallation.State = model.ClusterInstallationStateDeletionRequested
		err = s.store.UpdateClusterInstallation(clusterInstallation)
		if err != nil {
			logger.WithError(err).Warnf("Failed to mark cluster installation %s for deletion", clusterInstallation.ID)
			return installation.State
		}

		deletingClusterInstallations++
	}

	logger.Debugf(
		"Found %d cluster installations, %d deleting, %d deleted, %d failed",
		len(clusterInstallations),
		deletingClusterInstallations,
		deletedClusterInstallations,
		failedClusterInstallations,
	)

	if failedClusterInstallations > 0 {
		logger.Infof("Found %d failed cluster installations", failedClusterInstallations)
		return model.InstallationStateDeletionFailed
	}

	if deletedClusterInstallations < len(clusterInstallations) {
		return model.InstallationStateDeletionInProgress
	}

	return s.finalDeletionCleanup(installation, logger)
}

func (s *InstallationSupervisor) finalDeletionCleanup(installation *model.Installation, logger log.FieldLogger) string {
	err := s.aws.DeleteCNAME(installation.DNS, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to delete installation DNS")
		return model.InstallationStateDeletionFinalCleanup
	}

	err = installation.GetFilestore().Teardown(s.keepFilestoreData, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to delete installation AWS S3 filestore")
		return model.InstallationStateDeletionFinalCleanup
	}

	err = s.store.DeleteInstallation(installation.ID)
	if err != nil {
		logger.WithError(err).Warn("Failed to mark installation as deleted")
		return model.InstallationStateDeletionFinalCleanup
	}

	logger.Infof("Finished deleting cluster installation")

	return model.InstallationStateDeleted
}
