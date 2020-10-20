// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/utils"
	"github.com/mattermost/mattermost-cloud/internal/webhook"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	mmv1alpha1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1alpha1"
)

// installationStore abstracts the database operations required to query installations.
type installationStore interface {
	GetClusters(clusterFilter *model.ClusterFilter) ([]*model.Cluster, error)
	GetCluster(id string) (*model.Cluster, error)
	UpdateCluster(cluster *model.Cluster) error
	LockCluster(clusterID, lockerID string) (bool, error)
	UnlockCluster(clusterID string, lockerID string, force bool) (bool, error)

	GetInstallation(installationID string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.Installation, error)
	GetUnlockedInstallationsPendingWork() ([]*model.Installation, error)
	UpdateInstallation(installation *model.Installation) error
	UpdateInstallationGroupSequence(installation *model.Installation) error
	UpdateInstallationState(*model.Installation) error
	LockInstallation(installationID, lockerID string) (bool, error)
	UnlockInstallation(installationID, lockerID string, force bool) (bool, error)
	DeleteInstallation(installationID string) error

	CreateClusterInstallation(clusterInstallation *model.ClusterInstallation) error
	GetClusterInstallation(clusterInstallationID string) (*model.ClusterInstallation, error)
	GetClusterInstallations(*model.ClusterInstallationFilter) ([]*model.ClusterInstallation, error)
	LockClusterInstallations(clusterInstallationID []string, lockerID string) (bool, error)
	UnlockClusterInstallations(clusterInstallationID []string, lockerID string, force bool) (bool, error)
	UpdateClusterInstallation(clusterInstallation *model.ClusterInstallation) error

	GetMultitenantDatabase(multitenantdatabaseID string) (*model.MultitenantDatabase, error)
	GetMultitenantDatabases(filter *model.MultitenantDatabaseFilter) ([]*model.MultitenantDatabase, error)
	GetMultitenantDatabaseForInstallationID(installationID string) (*model.MultitenantDatabase, error)
	CreateMultitenantDatabase(multitenantDatabase *model.MultitenantDatabase) error
	UpdateMultitenantDatabase(multitenantDatabase *model.MultitenantDatabase) error
	LockMultitenantDatabase(multitenantdatabaseID, lockerID string) (bool, error)
	UnlockMultitenantDatabase(multitenantdatabaseID, lockerID string, force bool) (bool, error)

	GetAnnotationsForInstallation(installationID string) ([]*model.Annotation, error)

	GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error)
}

// provisioner abstracts the provisioning operations required by the installation supervisor.
type installationProvisioner interface {
	CreateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation, awsClient aws.AWS) error
	DeleteClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error
	UpdateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error
	HibernateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error
	GetClusterInstallationResource(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) (*mmv1alpha1.ClusterInstallation, error)
	GetClusterResources(cluster *model.Cluster, onlySchedulable bool) (*k8s.ClusterResources, error)
	GetPublicLoadBalancerEndpoint(cluster *model.Cluster, namespace string) (string, error)
}

// InstallationSupervisor finds installations pending work and effects the required changes.
//
// The degree of parallelism is controlled by a weighted semaphore, intended to be shared with
// other clients needing to coordinate background jobs.
type InstallationSupervisor struct {
	store                              installationStore
	provisioner                        installationProvisioner
	aws                                aws.AWS
	instanceID                         string
	clusterResourceThreshold           int
	clusterResourceThresholdScaleValue int
	keepDatabaseData                   bool
	keepFilestoreData                  bool
	resourceUtil                       *utils.ResourceUtil
	logger                             log.FieldLogger
}

// NewInstallationSupervisor creates a new InstallationSupervisor.
func NewInstallationSupervisor(store installationStore, installationProvisioner installationProvisioner, aws aws.AWS, instanceID string, threshold, thresholdScaleValue int, keepDatabaseData, keepFilestoreData bool, resourceUtil *utils.ResourceUtil, logger log.FieldLogger) *InstallationSupervisor {
	return &InstallationSupervisor{
		store:                              store,
		provisioner:                        installationProvisioner,
		aws:                                aws,
		instanceID:                         instanceID,
		clusterResourceThreshold:           threshold,
		clusterResourceThresholdScaleValue: thresholdScaleValue,
		keepDatabaseData:                   keepDatabaseData,
		keepFilestoreData:                  keepFilestoreData,
		resourceUtil:                       resourceUtil,
		logger:                             logger,
	}
}

// Shutdown performs graceful shutdown tasks for the installation supervisor.
func (s *InstallationSupervisor) Shutdown() {
	s.logger.Debug("Shutting down installation supervisor")
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
	logger := s.logger.WithFields(log.Fields{
		"installation": installation.ID,
	})

	lock := newInstallationLock(installation.ID, s.instanceID, s.store, logger)
	if !lock.TryLock() {
		return
	}
	defer lock.Unlock()

	// Before working on the installation, it is crucial that we ensure that it
	// was not updated to a new state by another provisioning server.
	originalState := installation.State
	installation, err := s.store.GetInstallation(installation.ID, true, false)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get refreshed installation")
		return
	}
	if installation.State != originalState {
		logger.WithField("oldInstallationState", originalState).
			WithField("newInstallationState", installation.State).
			Warn("Another provisioner has worked on this installation; skipping...")
		return
	}

	logger.Debugf("Supervising installation in state %s", installation.State)

	newState := s.transitionInstallation(installation, s.instanceID, logger)

	installation, err = s.store.GetInstallation(installation.ID, true, false)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get installation and thus persist state %s", newState)
		return
	}

	if installation.State == newState {
		return
	}

	oldState := installation.State
	installation.State = newState

	if installation.ConfigMergedWithGroup() && installation.State == model.InstallationStateStable {
		installation.SyncGroupAndInstallationSequence()
		err = s.store.UpdateInstallationGroupSequence(installation)
		if err != nil {
			logger.WithError(err).Warnf("Failed to set installation sequence to %s", newState)
			return
		}
	}

	err = s.store.UpdateInstallationState(installation)
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
		ExtraData: map[string]string{"DNS": installation.DNS},
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
		model.InstallationStateCreationNoCompatibleClusters:
		return s.createInstallation(installation, instanceID, logger)

	case model.InstallationStateCreationPreProvisioning:
		return s.preProvisionInstallation(installation, instanceID, logger)

	case model.InstallationStateCreationInProgress:
		return s.waitForCreationStable(installation, instanceID, logger)

	case model.InstallationStateCreationFinalTasks:
		return s.finalCreationTasks(installation, logger)

	case model.InstallationStateCreationDNS:
		return s.configureInstallationDNS(installation, instanceID, logger)

	case model.InstallationStateUpdateRequested:
		return s.updateInstallation(installation, instanceID, logger)

	case model.InstallationStateUpdateInProgress:
		return s.waitForUpdateStable(installation, instanceID, logger)

	case model.InstallationStateHibernationRequested:
		return s.hibernateInstallation(installation, instanceID, logger)

	case model.InstallationStateHibernationInProgress:
		return s.waitForHibernationStable(installation, instanceID, logger)

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

func (s *InstallationSupervisor) createInstallation(installation *model.Installation, instanceID string, logger log.FieldLogger) string {
	clusterInstallations, err := s.store.GetClusterInstallations(&model.ClusterInstallationFilter{
		InstallationID: installation.ID,
		PerPage:        model.AllPerPage,
	})
	if err != nil {
		logger.WithError(err).Warn("Failed to find cluster installations")
		return model.InstallationStateCreationRequested
	}

	if len(clusterInstallations) > 0 {
		logger.Warnf("Expected no cluster installations, but found %d", len(clusterInstallations))
		return s.preProvisionInstallation(installation, instanceID, logger)
	}

	clusterFilter := &model.ClusterFilter{
		PerPage:        model.AllPerPage,
		IncludeDeleted: false,
	}

	// Get only clusters that have all annotations present on the installation.
	// Clusters can have additional annotations not present on the installation.
	annotations, err := s.store.GetAnnotationsForInstallation(installation.ID)
	if err != nil {
		logger.WithError(err).Warn("Failed to get annotations for Installation")
		return model.InstallationStateCreationRequested
	}
	if len(annotations) > 0 {
		clusterFilter.Annotations = &model.AnnotationsFilter{MatchAllIDs: annotationsToIDs(annotations)}
	}

	// Proceed to requesting cluster installation creation on any available clusters.
	clusters, err := s.store.GetClusters(clusterFilter)
	if err != nil {
		logger.WithError(err).Warn("Failed to query clusters")
		return model.InstallationStateCreationRequested
	}

	for _, cluster := range clusters {
		clusterInstallation := s.createClusterInstallation(cluster, installation, instanceID, logger)
		if clusterInstallation != nil {
			return s.preProvisionInstallation(installation, instanceID, logger)
		}
	}

	// TODO: Support creating a cluster on demand if no existing cluster meets the criteria.
	logger.Warn("No compatible clusters available for installation scheduling")

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
	if !cluster.AllowInstallations {
		logger.Debugf("Cluster %s is set to not allow for new installation scheduling", cluster.ID)
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
			installation, err := s.store.GetInstallation(existingClusterInstallations[0].InstallationID, true, false)
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
		logger.WithError(err).Error("Invalid cluster installation size")
		return nil
	}
	clusterResources, err := s.provisioner.GetClusterResources(cluster, true)
	if err != nil {
		logger.WithError(err).Error("Failed to get cluster resources")
		return nil
	}

	installationCPURequirement := size.CalculateCPUMilliRequirement(
		installation.InternalDatabase(),
		installation.InternalFilestore(),
	)
	installationMemRequirement := size.CalculateMemoryMilliRequirement(
		installation.InternalDatabase(),
		installation.InternalFilestore(),
	)
	cpuPercent := clusterResources.CalculateCPUPercentUsed(installationCPURequirement)
	memoryPercent := clusterResources.CalculateMemoryPercentUsed(installationMemRequirement)

	if cpuPercent > s.clusterResourceThreshold || memoryPercent > s.clusterResourceThreshold {
		if s.clusterResourceThresholdScaleValue == 0 ||
			cluster.ProvisionerMetadataKops.NodeMinCount == cluster.ProvisionerMetadataKops.NodeMaxCount ||
			cluster.State != model.ClusterStateStable {
			logger.Debugf("Cluster %s would exceed the cluster load threshold (%d%%): CPU=%d%% (+%dm), Memory=%d%% (+%dMi)",
				cluster.ID,
				s.clusterResourceThreshold,
				cpuPercent, installationCPURequirement,
				memoryPercent, installationMemRequirement/1048576000, // Have to convert to Mi
			)
			return nil
		}

		// This cluster is ready to scale to meet increased resource demand.
		// TODO: if this ends up working well, build a safer interface for
		// updating the cluster. We should try to reuse some of the API flow
		// that already does this.

		newWorkerCount := cluster.ProvisionerMetadataKops.NodeMinCount + int64(s.clusterResourceThresholdScaleValue)
		if newWorkerCount > cluster.ProvisionerMetadataKops.NodeMaxCount {
			newWorkerCount = cluster.ProvisionerMetadataKops.NodeMaxCount
		}

		cluster.State = model.ClusterStateResizeRequested
		cluster.ProvisionerMetadataKops.ChangeRequest = &model.KopsMetadataRequestedState{
			NodeMinCount: newWorkerCount,
		}

		logger.WithField("cluster", cluster.ID).Infof("Scaling cluster worker nodes from %d to %d (max=%d)",
			cluster.ProvisionerMetadataKops.NodeMinCount,
			cluster.ProvisionerMetadataKops.ChangeRequest.NodeMinCount,
			cluster.ProvisionerMetadataKops.NodeMaxCount,
		)
		err = s.store.UpdateCluster(cluster)
		if err != nil {
			logger.WithError(err).Error("Failed to update cluster")
			return nil
		}

		webhookPayload := &model.WebhookPayload{
			Type:      model.TypeCluster,
			ID:        cluster.ID,
			NewState:  model.ClusterStateResizeRequested,
			OldState:  model.ClusterStateStable,
			Timestamp: time.Now().UnixNano(),
		}

		err = webhook.SendToAllWebhooks(s.store, webhookPayload, logger.WithField("webhookEvent", webhookPayload.NewState))
		if err != nil {
			logger.WithError(err).Error("Unable to process and send webhooks")
		}
	}

	// The cluster can support the cluster installation.

	clusterInstallation := &model.ClusterInstallation{
		ClusterID:      cluster.ID,
		InstallationID: installation.ID,
		Namespace:      installation.ID,
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

func (s *InstallationSupervisor) preProvisionInstallation(installation *model.Installation, instanceID string, logger log.FieldLogger) string {
	err := s.resourceUtil.GetDatabase(installation).Provision(s.store, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to provision installation database")
		return model.InstallationStateCreationPreProvisioning
	}

	err = s.resourceUtil.GetFilestore(installation).Provision(s.store, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to provision installation filestore")
		return model.InstallationStateCreationPreProvisioning
	}

	logger.Info("Installation pre-provisioning complete")

	return s.configureInstallationDNS(installation, instanceID, logger)
}

func (s *InstallationSupervisor) waitForCreationStable(installation *model.Installation, instanceID string, logger log.FieldLogger) string {
	stable, err := s.checkIfClusterInstallationsAreStable(installation, logger)
	if err != nil {
		logger.WithError(err).Error("Installation creation failed")
		return model.InstallationStateCreationFailed
	}
	if !stable {
		return model.InstallationStateCreationInProgress
	}

	logger.Info("Created cluster installations are now stable")

	return s.finalCreationTasks(installation, logger)
}

func (s *InstallationSupervisor) configureInstallationDNS(installation *model.Installation, instanceID string, logger log.FieldLogger) string {
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

		endpoint, err := s.provisioner.GetPublicLoadBalancerEndpoint(cluster, "nginx")
		if err != nil {
			logger.WithError(err).Error("Couldn't get the load balancer endpoint (nginx) for Cluster Installation")
			return model.InstallationStateCreationDNS
		}

		endpoints = append(endpoints, endpoint)
	}

	err = s.aws.CreatePublicCNAME(installation.DNS, endpoints, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to create DNS CNAME record")
		return model.InstallationStateCreationDNS
	}

	logger.Infof("Successfully configured DNS %s", installation.DNS)

	return s.waitForCreationStable(installation, instanceID, logger)
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

	logger.Info("Finished updating clusters installations")

	return s.waitForUpdateStable(installation, instanceID, logger)
}

func (s *InstallationSupervisor) waitForUpdateStable(installation *model.Installation, instanceID string, logger log.FieldLogger) string {
	stable, err := s.checkIfClusterInstallationsAreStable(installation, logger)
	if err != nil {
		logger.WithError(err).Error("Installation update failed")
		return model.InstallationStateUpdateFailed
	}
	if !stable {
		return model.InstallationStateUpdateInProgress
	}

	logger.Info("Finished updating installation")

	return model.InstallationStateStable
}

func (s *InstallationSupervisor) hibernateInstallation(installation *model.Installation, instanceID string, logger log.FieldLogger) string {
	clusterInstallations, err := s.store.GetClusterInstallations(&model.ClusterInstallationFilter{
		PerPage:        model.AllPerPage,
		InstallationID: installation.ID,
	})
	if err != nil {
		logger.WithError(err).Warn("Failed to find cluster installations")
		return installation.State
	}

	if len(clusterInstallations) == 0 {
		logger.Warn("Cluster installation list contained no results")
		return installation.State
	}

	var clusterInstallationIDs []string
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

		err = s.provisioner.HibernateClusterInstallation(cluster, installation, clusterInstallation)
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

	logger.Info("Finished updating clusters installations")

	return s.waitForHibernationStable(installation, instanceID, logger)
}

func (s *InstallationSupervisor) waitForHibernationStable(installation *model.Installation, instanceID string, logger log.FieldLogger) string {
	stable, err := s.checkIfClusterInstallationsAreStable(installation, logger)
	if err != nil {
		// TODO: there is no real failure state for hibernating so handle this
		// better in the future.
		logger.WithError(err).Warn("Installation hibernation failed")
		return model.InstallationStateHibernationInProgress
	}
	if !stable {
		return model.InstallationStateHibernationInProgress
	}

	logger.Info("Finished updating installation")

	return model.InstallationStateHibernating
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
	err := s.aws.DeletePublicCNAME(installation.DNS, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to delete installation DNS")
		return model.InstallationStateDeletionFinalCleanup
	}

	err = s.resourceUtil.GetDatabase(installation).Teardown(s.store, s.keepDatabaseData, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to delete database")
		return model.InstallationStateDeletionFinalCleanup
	}

	err = s.resourceUtil.GetFilestore(installation).Teardown(s.keepFilestoreData, s.store, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to delete filestore")
		return model.InstallationStateDeletionFinalCleanup
	}

	err = s.store.DeleteInstallation(installation.ID)
	if err != nil {
		logger.WithError(err).Warn("Failed to mark installation as deleted")
		return model.InstallationStateDeletionFinalCleanup
	}

	logger.Info("Finished deleting installation")

	return model.InstallationStateDeleted
}

func (s *InstallationSupervisor) finalCreationTasks(installation *model.Installation, logger log.FieldLogger) string {
	logger.Info("Finished final creation tasks")
	return model.InstallationStateStable
}

// Helper funcs

// checkIfClusterInstallationsAreStable returns if all cluster installations
// belonging to an installation are stable or not. Any errors that will likely
// not succeed on future retries will also be returned. Otherwise, the error will
// be logged and a nil error is returned. This will allow the caller to confidently
// retry until everything is stable.
func (s *InstallationSupervisor) checkIfClusterInstallationsAreStable(installation *model.Installation, logger log.FieldLogger) (bool, error) {
	clusterInstallations, err := s.store.GetClusterInstallations(&model.ClusterInstallationFilter{
		InstallationID: installation.ID,
		PerPage:        model.AllPerPage,
	})
	if err != nil {
		logger.WithError(err).Warn("Failed to find cluster installations")
		return false, nil
	}

	var stable, reconciling, failed, other int
	for _, clusterInstallation := range clusterInstallations {
		switch clusterInstallation.State {
		case model.ClusterInstallationStateStable:
			stable++
		case model.ClusterInstallationStateReconciling:
			reconciling++
		case model.ClusterInstallationStateCreationFailed:
			failed++
		default:
			other++
		}
	}

	logger.Debugf("Found %d cluster installations: %d stable, %d reconciling, %d failed, %d other", len(clusterInstallations), stable, reconciling, failed, other)

	if len(clusterInstallations) == stable {
		return true, nil
	}
	if failed > 0 {
		return false, errors.Errorf("found %d failed cluster installations", failed)
	}

	return false, nil
}

// annotationsToIDs parses slice of annotations to slice of strings containing annotations IDs.
func annotationsToIDs(annotations []*model.Annotation) []string {
	ids := make([]string, 0, len(annotations))
	for _, ann := range annotations {
		ids = append(ids, ann.ID)
	}
	return ids
}
