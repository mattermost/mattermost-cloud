// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/events"

	"github.com/mattermost/mattermost-cloud/internal/provisioner"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/mattermost/mattermost-cloud/internal/metrics"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/utils"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	mmv1alpha1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1alpha1"
)

const (
	latestCRVersion = model.V1betaCRVersion
)

// installationStore abstracts the database operations required to query installations.
type installationStore interface {
	GetClusters(clusterFilter *model.ClusterFilter) ([]*model.Cluster, error)
	GetCluster(id string) (*model.Cluster, error)
	UpdateCluster(cluster *model.Cluster) error
	clusterLockStore

	GetInstallation(installationID string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.Installation, error)
	GetUnlockedInstallationsPendingWork() ([]*model.Installation, error)
	UpdateInstallation(installation *model.Installation) error
	UpdateInstallationGroupSequence(installation *model.Installation) error
	UpdateInstallationState(*model.Installation) error
	UpdateInstallationCRVersion(installationID, crVersion string) error
	DeleteInstallation(installationID string) error
	installationLockStore

	GetSingleTenantDatabaseConfigForInstallation(installationID string) (*model.SingleTenantDatabaseConfig, error)
	GetAnnotationsForInstallation(installationID string) ([]*model.Annotation, error)

	CreateClusterInstallation(clusterInstallation *model.ClusterInstallation) error
	GetClusterInstallation(clusterInstallationID string) (*model.ClusterInstallation, error)
	GetClusterInstallations(*model.ClusterInstallationFilter) ([]*model.ClusterInstallation, error)
	UpdateClusterInstallation(clusterInstallation *model.ClusterInstallation) error
	clusterInstallationLockStore

	GetGroup(id string) (*model.Group, error)
	LockGroup(groupID, lockerID string) (bool, error)
	UnlockGroup(groupID, lockerID string, force bool) (bool, error)

	GetInstallationBackups(filter *model.InstallationBackupFilter) ([]*model.InstallationBackup, error)
	UpdateInstallationBackupState(backup *model.InstallationBackup) error
	installationBackupLockStore

	GetInstallationDBMigrationOperations(filter *model.InstallationDBMigrationFilter) ([]*model.InstallationDBMigrationOperation, error)
	UpdateInstallationDBMigrationOperationState(operation *model.InstallationDBMigrationOperation) error
	installationDBMigrationOperationLockStore

	GetInstallationDBRestorationOperations(filter *model.InstallationDBRestorationFilter) ([]*model.InstallationDBRestorationOperation, error)
	UpdateInstallationDBRestorationOperationState(operation *model.InstallationDBRestorationOperation) error
	installationDBRestorationLockStore

	GetStateChangeEvents(filter *model.StateChangeEventFilter) ([]*model.StateChangeEventData, error)

	GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error)

	model.InstallationDatabaseStoreInterface
}

type eventProducer interface {
	ProduceInstallationStateChangeEvent(installation *model.Installation, oldState string, extraDataFields ...events.DataField) error
	ProduceClusterStateChangeEvent(cluster *model.Cluster, oldState string, extraDataFields ...events.DataField) error
	ProduceClusterInstallationStateChangeEvent(clusterInstallation *model.ClusterInstallation, oldState string, extraDataFields ...events.DataField) error
}

// provisioner abstracts the provisioning operations required by the installation supervisor.
type installationProvisioner interface {
	ClusterInstallationProvisioner(version string) provisioner.ClusterInstallationProvisioner
	GetClusterResources(cluster *model.Cluster, onlySchedulable bool) (*k8s.ClusterResources, error)
	GetPublicLoadBalancerEndpoint(cluster *model.Cluster, namespace string) (string, error)
}

// InstallationSupervisor finds installations pending work and effects the required changes.
//
// The degree of parallelism is controlled by a weighted semaphore, intended to be shared with
// other clients needing to coordinate background jobs.
type InstallationSupervisor struct {
	store             installationStore
	provisioner       installationProvisioner
	aws               aws.AWS
	instanceID        string
	keepDatabaseData  bool
	keepFilestoreData bool
	scheduling        InstallationSupervisorSchedulingOptions
	resourceUtil      *utils.ResourceUtil
	logger            log.FieldLogger
	metrics           *metrics.CloudMetrics
	eventsProducer    eventProducer
	forceCRUpgrade    bool
}

// InstallationSupervisorSchedulingOptions are the various options that control
// how installation scheduling occurs.
type InstallationSupervisorSchedulingOptions struct {
	balanceInstallations               bool
	clusterResourceThreshold           int
	clusterResourceThresholdScaleValue int
}

// NewInstallationSupervisor creates a new InstallationSupervisor.
func NewInstallationSupervisor(
	store installationStore,
	installationProvisioner installationProvisioner,
	aws aws.AWS,
	instanceID string,
	keepDatabaseData,
	keepFilestoreData bool,
	scheduling InstallationSupervisorSchedulingOptions,
	resourceUtil *utils.ResourceUtil,
	logger log.FieldLogger,
	metrics *metrics.CloudMetrics,
	eventsProducer eventProducer,
	forceCRUpgrade bool) *InstallationSupervisor {
	return &InstallationSupervisor{
		store:             store,
		provisioner:       installationProvisioner,
		aws:               aws,
		instanceID:        instanceID,
		keepDatabaseData:  keepDatabaseData,
		keepFilestoreData: keepFilestoreData,
		scheduling:        scheduling,
		resourceUtil:      resourceUtil,
		logger:            logger,
		metrics:           metrics,
		eventsProducer:    eventsProducer,
		forceCRUpgrade:    forceCRUpgrade,
	}
}

// NewInstallationSupervisorSchedulingOptions creates a new InstallationSupervisorSchedulingOptions.
func NewInstallationSupervisorSchedulingOptions(balanceInstallations bool, clusterResourceThreshold, clusterResourceThresholdScaleValue int) InstallationSupervisorSchedulingOptions {
	return InstallationSupervisorSchedulingOptions{
		balanceInstallations:               balanceInstallations,
		clusterResourceThreshold:           clusterResourceThreshold,
		clusterResourceThresholdScaleValue: clusterResourceThresholdScaleValue,
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
		// Perform a final group configuration check. This time, it is vital to
		// check the installation while the group is locked.
		groupLock := newGroupLock(*installation.GroupID, s.instanceID, s.store, logger)
		if !groupLock.TryLock() {
			logger.Error("Failed to lock group for final configuration check")
			return
		}
		defer groupLock.Unlock()

		group, err := s.store.GetGroup(*installation.GroupID)
		if err != nil {
			logger.WithError(err).Error("Failed to get group for final configuration check")
			return
		}
		if *installation.GroupSequence != group.Sequence {
			logger.Warnf("The installation's group configuration has changed; moving installation back to %s", model.InstallationStateUpdateRequested)
			installation.State = model.InstallationStateUpdateRequested
		}
	}

	err = s.store.UpdateInstallationState(installation)
	if err != nil {
		logger.WithError(err).Errorf("Failed to set installation state to %s", newState)
		return
	}

	err = s.processInstallationMetrics(installation, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to process installation metrics")
	}

	err = s.eventsProducer.ProduceInstallationStateChangeEvent(installation, oldState)
	if err != nil {
		logger.WithError(err).Error("Failed to create installation state change event")
	}

	logger.Debugf("Transitioned installation from %s to %s", oldState, installation.State)
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

	case model.InstallationStateWakeUpRequested:
		return s.wakeUpInstallation(installation, instanceID, logger)

	case model.InstallationStateDeletionRequested,
		model.InstallationStateDeletionInProgress:
		return s.deleteInstallation(installation, instanceID, logger)

	case model.InstallationStateDeletionFinalCleanup:
		return s.finalDeletionCleanup(installation, instanceID, logger)

	case model.InstallationStateDNSMigrationHibernating:
		return s.dnsSwitchForHibernatingInstallation(installation, instanceID, logger)
	default:
		logger.Warnf("Found installation pending work in unexpected state %s", installation.State)
		return installation.State
	}
}

func (s *InstallationSupervisor) createInstallation(installation *model.Installation, instanceID string, logger log.FieldLogger) string {
	// Before starting, we check the installation and group sequence numbers and
	// sync them if they are not already. This is used to check if the group
	// configuration has changed during the creation process or not.
	if !installation.InstallationSequenceMatchesMergedGroupSequence() {
		installation.SyncGroupAndInstallationSequence()

		logger.Debugf("Updating installation to group configuration sequence %d", *installation.GroupSequence)

		err := s.store.UpdateInstallationGroupSequence(installation)
		if err != nil {
			logger.WithError(err).Errorf("Failed to set installation sequence to %d", *installation.GroupSequence)
			return installation.State
		}
	}

	clusterInstallations, err := s.store.GetClusterInstallations(&model.ClusterInstallationFilter{
		InstallationID: installation.ID,
		Paging:         model.AllPagesNotDeleted(),
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
		Paging: model.AllPagesNotDeleted(),
	}

	// Get only clusters that have all annotations present on the installation.
	// Clusters can have additional annotations not present on the installation.
	annotations, err := s.store.GetAnnotationsForInstallation(installation.ID)
	if err != nil {
		logger.WithError(err).Warn("Failed to get annotations for Installation")
		return model.InstallationStateCreationRequested
	}
	if len(annotations) > 0 {
		clusterFilter.Annotations = &model.AnnotationsFilter{MatchAllIDs: getAnnotationsIDs(annotations)}
	}

	// Proceed to requesting cluster installation creation on any available clusters.
	clusters, err := s.store.GetClusters(clusterFilter)
	if err != nil {
		logger.WithError(err).Warn("Failed to query clusters")
		return model.InstallationStateCreationRequested
	}
	if len(clusters) == 0 {
		logger.Warnf("No clusters found matching the filter, installation annotations are: [%s]", strings.Join(getAnnotationsNames(annotations), ", "))
	}

	if s.scheduling.balanceInstallations {
		logger.Info("Attempting to schedule installation on the lowest-utilized cluster")
		clusters = s.prioritizeLowerUtilizedClusters(clusters, installation, instanceID, logger)
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

// prioritizeLowerUtilizedClusters attempts filter the given cluster list and
// order it by lowest resource usage first. This should be considered best
// effort.
// Note the following:
//   - This check is done without locking to avoid creating additional
//     congestion.
//   - Resource usage ordering is done by taking an average of CPU + memory
//     percentages.
//   - The returned list will generally be in order of lowest-to-highest
//     resource usage, but the only guarantee is that the first cluster in the
//     list will be the lowest at the time it was checked.
//   - When scheduling an installation, all of the standard scheduling checks
//     should be performed again under cluster lock.
func (s *InstallationSupervisor) prioritizeLowerUtilizedClusters(clusters []*model.Cluster, installation *model.Installation, instanceID string, logger log.FieldLogger) []*model.Cluster {
	lowestResourcePercent := 10000
	var filteredPrioritizedClusters []*model.Cluster

	for _, cluster := range clusters {
		if !s.installationCanBeScheduledOnCluster(cluster, installation, logger) {
			continue
		}

		clusterResources, err := s.provisioner.GetClusterResources(cluster, true)
		if err != nil {
			logger.WithError(err).Error("Failed to get cluster resources")
			continue
		}
		size, err := mmv1alpha1.GetClusterSize(installation.Size)
		if err != nil {
			logger.WithError(err).Error("Invalid cluster installation size")
			continue
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
		combinedPercent := (cpuPercent + memoryPercent) / 2
		logger.Debugf("Cluster %s analyzed with %d%% expected resource usage", cluster.ID, combinedPercent)
		if combinedPercent < lowestResourcePercent {
			// This is the lowest utilized cluster so far so prepend.
			filteredPrioritizedClusters = append([]*model.Cluster{cluster}, filteredPrioritizedClusters...)
			lowestResourcePercent = combinedPercent
		} else {
			// Otherwise just append it to the end of the list.
			filteredPrioritizedClusters = append(filteredPrioritizedClusters, cluster)
		}
	}

	return filteredPrioritizedClusters
}

// createClusterInstallation attempts to schedule a cluster installation onto the given cluster.
func (s *InstallationSupervisor) createClusterInstallation(cluster *model.Cluster, installation *model.Installation, instanceID string, logger log.FieldLogger) *model.ClusterInstallation {
	clusterLock := newClusterLock(cluster.ID, instanceID, s.store, logger)
	if !clusterLock.TryLock() {
		logger.Debugf("Failed to lock cluster %s", cluster.ID)
		return nil
	}
	defer clusterLock.Unlock()

	if !s.installationCanBeScheduledOnCluster(cluster, installation, logger) {
		return nil
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

	if cpuPercent > s.scheduling.clusterResourceThreshold || memoryPercent > s.scheduling.clusterResourceThreshold {
		if s.scheduling.clusterResourceThresholdScaleValue == 0 ||
			cluster.ProvisionerMetadataKops.NodeMinCount == cluster.ProvisionerMetadataKops.NodeMaxCount ||
			cluster.State != model.ClusterStateStable {
			logger.Debugf("Cluster %s would exceed the cluster load threshold (%d%%): CPU=%d%% (+%dm), Memory=%d%% (+%dMi)",
				cluster.ID,
				s.scheduling.clusterResourceThreshold,
				cpuPercent, installationCPURequirement,
				memoryPercent, installationMemRequirement/1048576000, // Have to convert to Mi
			)
			return nil
		}

		// This cluster is ready to scale to meet increased resource demand.
		// TODO: if this ends up working well, build a safer interface for
		// updating the cluster. We should try to reuse some of the API flow
		// that already does this.

		newWorkerCount := cluster.ProvisionerMetadataKops.NodeMinCount + int64(s.scheduling.clusterResourceThresholdScaleValue)
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

		err = s.eventsProducer.ProduceClusterStateChangeEvent(cluster, model.ClusterStateStable)
		if err != nil {
			logger.WithError(err).Error("Failed to create cluster state change event")
		}
	}

	// The cluster can support the cluster installation.
	clusterInstallation := &model.ClusterInstallation{
		ClusterID:      cluster.ID,
		InstallationID: installation.ID,
		Namespace:      installation.ID,
		State:          model.ClusterInstallationStateCreationRequested,
		IsActive:       true,
	}

	err = s.store.CreateClusterInstallation(clusterInstallation)
	if err != nil {
		logger.WithError(err).Warn("Failed to create cluster installation")
		return nil
	}

	err = s.eventsProducer.ProduceClusterInstallationStateChangeEvent(clusterInstallation, model.NonApplicableState)
	if err != nil {
		logger.WithError(err).Error("Failed to create cluster installation state change event")
	}

	logger.Infof("Requested creation of cluster installation on cluster %s. Expected resource load: CPU=%d%%, Memory=%d%%", cluster.ID, cpuPercent, memoryPercent)

	return clusterInstallation
}

// installationCanBeScheduledOnCluster checks if the given installation can be
// scheduled on the given cluster in regards to configuration and state. This
// does not include resource checks.
func (s *InstallationSupervisor) installationCanBeScheduledOnCluster(cluster *model.Cluster, installation *model.Installation, logger log.FieldLogger) bool {
	if cluster.State != model.ClusterStateStable {
		logger.Debugf("Cluster %s is not stable (currently %s)", cluster.ID, cluster.State)
		return false
	}
	if !cluster.AllowInstallations {
		logger.Debugf("Cluster %s is set to not allow for new installation scheduling", cluster.ID)
		return false
	}

	existingClusterInstallations, err := s.store.GetClusterInstallations(&model.ClusterInstallationFilter{
		Paging:    model.AllPagesNotDeleted(),
		ClusterID: cluster.ID,
	})
	if err != nil {
		logger.WithError(err).Error("Failed to get existing cluster installations")
		return false
	}

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
			return false
		}
	} else {
		if len(existingClusterInstallations) == 1 {
			// This should be the only scenario where we need to check if the
			// cluster installation running requires isolation or not.
			installation, err := s.store.GetInstallation(existingClusterInstallations[0].InstallationID, true, false)
			if err != nil {
				logger.WithError(err).Error("Failed to get existing installation")
				return false
			}
			if installation.Affinity == model.InstallationAffinityIsolated {
				logger.Debugf("Cluster %s already has an isolated installation %s", cluster.ID, installation.ID)
				return false
			}
		}
	}

	return true
}

func (s *InstallationSupervisor) preProvisionInstallation(installation *model.Installation, instanceID string, logger log.FieldLogger) string {
	err := s.resourceUtil.GetDatabaseForInstallation(installation).Provision(s.store, logger)
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
	// TODO: Check group config for changes.

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

	endpoints, err := s.getPublicLBEndpoint(installation, logger)
	if err != nil {
		logger.WithError(err).Warn("Failed to find load balancer endpoint (nginx) for Cluster Installation")
		return model.InstallationStateCreationDNS
	}

	err = s.aws.CreatePublicCNAME(installation.DNS, endpoints, "", logger)
	if err != nil {
		logger.WithError(err).Error("Failed to create DNS CNAME record")
		return model.InstallationStateCreationDNS
	}

	logger.Infof("Successfully configured DNS %s", installation.DNS)

	return s.waitForCreationStable(installation, instanceID, logger)
}

func (s *InstallationSupervisor) getPublicLBEndpoint(installation *model.Installation, logger log.FieldLogger) ([]string, error) {
	isActive := true
	clusterInstallations, err := s.store.GetClusterInstallations(&model.ClusterInstallationFilter{
		InstallationID: installation.ID,
		IsActive:       &isActive,
		Paging:         model.AllPagesNotDeleted(),
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed to find cluster installations")
	}

	var endpoints []string
	for _, clusterInstallation := range clusterInstallations {
		cluster, err := s.store.GetCluster(clusterInstallation.ClusterID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to query cluster %s", clusterInstallation.ClusterID)
		}
		if cluster == nil {
			return nil, errors.Wrapf(err, "failed to find cluster %s", clusterInstallation.ClusterID)
		}

		endpoint, err := s.provisioner.GetPublicLoadBalancerEndpoint(cluster, "nginx")
		if err != nil {
			return nil, errors.Wrap(err, "Couldn't get the load balancer endpoint (nginx) for Cluster Installation")
		}

		endpoints = append(endpoints, endpoint)
	}

	return endpoints, nil
}
func (s *InstallationSupervisor) updateInstallation(installation *model.Installation, instanceID string, logger log.FieldLogger) string {
	// Before starting, we check the installation and group sequence numbers and
	// sync them if they are not already. This is used to check if the group
	// configuration has changed during the upgrade process or not.
	if !installation.InstallationSequenceMatchesMergedGroupSequence() {
		installation.SyncGroupAndInstallationSequence()

		logger.Debugf("Updating installation to group configuration sequence %d", *installation.GroupSequence)

		err := s.store.UpdateInstallationGroupSequence(installation)
		if err != nil {
			logger.WithError(err).Errorf("Failed to set installation sequence to %d", *installation.GroupSequence)
			return installation.State
		}
	}

	if s.forceCRUpgrade && installation.CRVersion != latestCRVersion {
		installation.CRVersion = latestCRVersion
		logger.Infof("Updating Installation CR Version to '%s'", latestCRVersion)

		err := s.store.UpdateInstallationCRVersion(installation.ID, latestCRVersion)
		if err != nil {
			logger.WithError(err).Error("Failed to update installation CRVersion")
			return installation.State
		}
	}

	clusterInstallations, err := s.store.GetClusterInstallations(&model.ClusterInstallationFilter{
		Paging:         model.AllPagesNotDeleted(),
		InstallationID: installation.ID,
	})
	if err != nil {
		logger.WithError(err).Warn("Failed to find cluster installations")
		return installation.State
	}

	clusterInstallationIDs := getClusterInstallationIDs(clusterInstallations)
	if len(clusterInstallations) > 0 {
		clusterInstallationLocks := newClusterInstallationLocks(clusterInstallationIDs, instanceID, s.store, logger)
		if !clusterInstallationLocks.TryLock() {
			logger.Debugf("Failed to lock %d cluster installations", len(clusterInstallations))
			return installation.State
		}
		defer clusterInstallationLocks.Unlock()

		// Fetch the same cluster installations again, now that we have the locks.
		clusterInstallations, err = s.store.GetClusterInstallations(&model.ClusterInstallationFilter{
			Paging: model.AllPagesNotDeleted(),
			IDs:    clusterInstallationIDs,
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

		isReady, err := s.provisioner.ClusterInstallationProvisioner(installation.CRVersion).
			EnsureCRMigrated(cluster, clusterInstallation)
		if err != nil {
			logger.WithError(err).Error("Failed to migrate cluster installation CR")
			return installation.State
		}
		if !isReady {
			logger.Info("Cluster installation CR migration not finished")
			return installation.State
		}

		err = s.provisioner.ClusterInstallationProvisioner(installation.CRVersion).
			UpdateClusterInstallation(cluster, installation, clusterInstallation)
		if err != nil {
			logger.WithError(err).Error("Failed to update cluster installation")
			return installation.State
		}

		if clusterInstallation.State != model.ClusterInstallationStateReconciling {
			oldState := clusterInstallation.State
			clusterInstallation.State = model.ClusterInstallationStateReconciling
			err = s.store.UpdateClusterInstallation(clusterInstallation)
			if err != nil {
				logger.Errorf("Failed to change cluster installation state to %s", model.ClusterInstallationStateReconciling)
				return installation.State
			}

			err = s.eventsProducer.ProduceClusterInstallationStateChangeEvent(clusterInstallation, oldState)
			if err != nil {
				logger.WithError(err).Error("Failed to create cluster installation state change event")
			}
		}
	}

	logger.Info("Finished updating cluster installations")

	return s.waitForUpdateStable(installation, instanceID, logger)
}

func (s *InstallationSupervisor) waitForUpdateStable(installation *model.Installation, instanceID string, logger log.FieldLogger) string {
	// If the installation belongs to a group that has been updated, we requeue
	// the installation update.
	if !installation.InstallationSequenceMatchesMergedGroupSequence() {
		logger.Warnf("The installation's group configuration has changed; moving installation back to %s", model.InstallationStateUpdateRequested)
		return model.InstallationStateUpdateRequested
	}

	stable, err := s.checkIfClusterInstallationsAreStable(installation, logger)
	if err != nil {
		logger.WithError(err).Error("Installation update failed")
		return model.InstallationStateUpdateFailed
	}
	if !stable {
		return model.InstallationStateUpdateInProgress
	}

	err = s.aws.UpdatePublicRecordIDForCNAME(installation.DNS, installation.DNS, logger)
	if err != nil {
		logger.WithError(err).Warn("Failed to update the installation route53 record to the standard ID value")
		return installation.State
	}

	logger.Info("Finished updating installation")

	return model.InstallationStateStable
}

// Unused stub function
// Will verify that all cluster installation belonging to an installation match
// the provisioner's config.
func (s *InstallationSupervisor) verifyClusterInstallationResourcesMatchInstallationConfig(installation *model.Installation, logger log.FieldLogger) (bool, error) {
	clusterInstallations, err := s.store.GetClusterInstallations(&model.ClusterInstallationFilter{
		Paging:         model.AllPagesNotDeleted(),
		InstallationID: installation.ID,
	})
	if err != nil {
		return false, errors.Wrap(err, "failed to lookup cluster installations")
	}

	if len(clusterInstallations) == 0 {
		return false, errors.Wrap(err, "cluster installation list contained no results")
	}

	for _, clusterInstallation := range clusterInstallations {
		cluster, err := s.store.GetCluster(clusterInstallation.ClusterID)
		if err != nil {
			return false, errors.Wrapf(err, "failed to query cluster %s", clusterInstallation.ClusterID)
		}
		if cluster == nil {
			return false, errors.Wrapf(err, "failed to find cluster %s", clusterInstallation.ClusterID)
		}

		match, err := s.provisioner.ClusterInstallationProvisioner(installation.CRVersion).
			VerifyClusterInstallationMatchesConfig(cluster, installation, clusterInstallation)
		if err != nil {
			return false, errors.Wrapf(err, "failed to verify cluster installation matches")
		}
		if !match {
			return false, nil
		}
	}

	return true, nil
}

func (s *InstallationSupervisor) hibernateInstallation(installation *model.Installation, instanceID string, logger log.FieldLogger) string {
	err := s.aws.UpdatePublicRecordIDForCNAME(installation.DNS, aws.HibernatingInstallationResourceRecordIDPrefix+installation.DNS, logger)
	if err != nil {
		logger.WithError(err).Warn("Failed to update the installation route53 record with hibernation prefix")
		return installation.State
	}

	err = s.resourceUtil.GetDatabaseForInstallation(installation).RefreshResourceMetadata(s.store, logger)
	if err != nil {
		logger.WithError(err).Warn("Failed to update database resource metadata")
		return installation.State
	}

	clusterInstallations, err := s.store.GetClusterInstallations(&model.ClusterInstallationFilter{
		Paging:         model.AllPagesNotDeleted(),
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

	clusterInstallationIDs := getClusterInstallationIDs(clusterInstallations)

	clusterInstallationLocks := newClusterInstallationLocks(clusterInstallationIDs, instanceID, s.store, logger)
	if !clusterInstallationLocks.TryLock() {
		logger.Debugf("Failed to lock %d cluster installations", len(clusterInstallations))
		return installation.State
	}
	defer clusterInstallationLocks.Unlock()

	// Fetch the same cluster installations again, now that we have the locks.
	clusterInstallations, err = s.store.GetClusterInstallations(&model.ClusterInstallationFilter{
		Paging: model.AllPagesNotDeleted(),
		IDs:    clusterInstallationIDs,
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

		err = s.provisioner.ClusterInstallationProvisioner(installation.CRVersion).
			HibernateClusterInstallation(cluster, installation, clusterInstallation)
		if err != nil {
			logger.WithError(err).Error("Failed to update cluster installation")
			return installation.State
		}

		oldState := clusterInstallation.State
		clusterInstallation.State = model.ClusterInstallationStateReconciling
		err = s.store.UpdateClusterInstallation(clusterInstallation)
		if err != nil {
			logger.WithError(err).Errorf("Failed to change cluster installation state to %s", model.ClusterInstallationStateReconciling)
			return installation.State
		}

		err = s.eventsProducer.ProduceClusterInstallationStateChangeEvent(clusterInstallation, oldState)
		if err != nil {
			logger.WithError(err).Error("Failed to create cluster installation state change event")
		}
	}

	logger.Info("Finished updating cluster installations")

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

	logger.Info("Finished hibernating installation")

	return model.InstallationStateHibernating
}

func (s *InstallationSupervisor) wakeUpInstallation(installation *model.Installation, instanceID string, logger log.FieldLogger) string {
	err := s.resourceUtil.GetDatabaseForInstallation(installation).RefreshResourceMetadata(s.store, logger)
	if err != nil {
		logger.WithError(err).Warn("Failed to update database resource metadata")
		return installation.State
	}

	return s.updateInstallation(installation, instanceID, logger)
}

func (s *InstallationSupervisor) deleteInstallation(installation *model.Installation, instanceID string, logger log.FieldLogger) string {
	clusterInstallations, err := s.store.GetClusterInstallations(&model.ClusterInstallationFilter{
		Paging:         model.AllPagesWithDeleted(),
		InstallationID: installation.ID,
	})
	if err != nil {
		logger.WithError(err).Warn("Failed to find cluster installations")
		return installation.State
	}

	clusterInstallationIDs := getClusterInstallationIDs(clusterInstallations)
	if len(clusterInstallations) > 0 {
		clusterInstallationLocks := newClusterInstallationLocks(clusterInstallationIDs, instanceID, s.store, logger)
		if !clusterInstallationLocks.TryLock() {
			logger.Debugf("Failed to lock %d cluster installations", len(clusterInstallations))
			return installation.State
		}
		defer clusterInstallationLocks.Unlock()

		// Fetch the same cluster installations again, now that we have the locks.
		clusterInstallations, err = s.store.GetClusterInstallations(&model.ClusterInstallationFilter{
			Paging: model.AllPagesWithDeleted(),
			IDs:    clusterInstallationIDs,
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

		oldState := clusterInstallation.State
		clusterInstallation.State = model.ClusterInstallationStateDeletionRequested
		err = s.store.UpdateClusterInstallation(clusterInstallation)
		if err != nil {
			logger.WithError(err).Warnf("Failed to mark cluster installation %s for deletion", clusterInstallation.ID)
			return installation.State
		}

		err = s.eventsProducer.ProduceClusterInstallationStateChangeEvent(clusterInstallation, oldState)
		if err != nil {
			logger.WithError(err).Error("Failed to create cluster installation state change event")
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

	return s.finalDeletionCleanup(installation, instanceID, logger)
}

func (s *InstallationSupervisor) finalDeletionCleanup(installation *model.Installation, instanceID string, logger log.FieldLogger) string {
	err := s.aws.DeletePublicCNAME(installation.DNS, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to delete installation DNS")
		return model.InstallationStateDeletionFinalCleanup
	}

	// Backups are stored in Installations file store, therefore if file store is deleted
	// the backups will be deleted also.
	if !s.keepFilestoreData {
		finished, err := s.deleteBackups(installation, instanceID, logger)
		if err != nil {
			logger.WithError(err).Error("Failed to delete backups")
			return model.InstallationStateDeletionFinalCleanup
		}
		if !finished {
			logger.Info("Installation backups deletion in progress")
			return model.InstallationStateDeletionFinalCleanup
		}
	}

	migrationDeletionFinished, err := s.deleteMigrationOperations(installation, instanceID, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to delete db migration operations")
		return model.InstallationStateDeletionFinalCleanup
	}
	restorationDeletionFinished, err := s.deleteRestorationOperations(installation, instanceID, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to delete db restoration operations")
		return model.InstallationStateDeletionFinalCleanup
	}
	if !migrationDeletionFinished || !restorationDeletionFinished {
		logger.Info("Installation db restoration and migration deletion in progress")
		return model.InstallationStateDeletionFinalCleanup
	}

	err = s.resourceUtil.GetDatabaseForInstallation(installation).Teardown(s.store, s.keepDatabaseData, logger)
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

func (s *InstallationSupervisor) deleteBackups(installation *model.Installation, instanceID string, logger log.FieldLogger) (bool, error) {
	logger.Info("Deleting installation backups")

	backups, err := s.store.GetInstallationBackups(&model.InstallationBackupFilter{
		InstallationID: installation.ID,
		Paging:         model.AllPagesNotDeleted(),
	})
	if err != nil {
		return false, errors.Wrap(err, "failed to list backup")
	}

	if len(backups) == 0 {
		logger.Info("No existing backups found for installation")
		return true, nil
	}

	backupIDs := getInstallationBackupsIDs(backups)

	installationBackupsLocks := newBackupsLock(backupIDs, instanceID, s.store, logger)
	if !installationBackupsLocks.TryLock() {
		return false, errors.Errorf("Failed to lock %d installation backups", len(backups))
	}
	defer installationBackupsLocks.Unlock()

	// Fetch the same backups again, now that we have the locks.
	backups, err = s.store.GetInstallationBackups(&model.InstallationBackupFilter{
		Paging: model.AllPagesNotDeleted(),
		IDs:    backupIDs,
	})
	if err != nil {
		return false, errors.Wrapf(err, "failed to fetch %d installation backups by ids", len(backups))
	}

	if len(backups) != len(backupIDs) {
		logger.Warnf("Found only %d installation backups after locking, expected %d", len(backups), len(backupIDs))
	}

	deletedBackups := 0
	deletingBackups := 0
	deletionFailedBackups := 0

	for _, backup := range backups {
		switch backup.State {
		case model.InstallationBackupStateDeleted:
			deletedBackups++
			continue
		case model.InstallationBackupStateDeletionRequested:
			deletingBackups++
			continue
		case model.InstallationBackupStateDeletionFailed:
			deletionFailedBackups++
			continue
		}

		logger.Debugf("Deleting installation backup %s in state %s", backup.ID, backup.State)
		backup.State = model.InstallationBackupStateDeletionRequested
		err = s.store.UpdateInstallationBackupState(backup)
		if err != nil {
			return false, errors.Wrapf(err, "failed to mark istallation backup %s for deletion", backup.ID)
		}
		deletingBackups++
	}

	logger.Debugf(
		"Found %d installation backups, %d deleting, %d deleted, %d failed",
		len(backups),
		deletingBackups,
		deletedBackups,
		deletionFailedBackups,
	)

	if deletionFailedBackups > 0 {
		return false, errors.Errorf("Failed to delete %d installation backups", deletionFailedBackups)
	}

	if deletingBackups > 0 {
		logger.Infof("Installation backups deletion in progress, deleting backups %d", deletingBackups)
		return false, nil
	}

	return true, nil
}

func (s *InstallationSupervisor) deleteRestorationOperations(installation *model.Installation, instanceID string, logger log.FieldLogger) (bool, error) {
	logger.Info("Deleting installation db restoration operations")

	restorationOperations, err := s.store.GetInstallationDBRestorationOperations(&model.InstallationDBRestorationFilter{
		InstallationID: installation.ID,
		Paging:         model.AllPagesNotDeleted(),
	})
	if err != nil {
		return false, errors.Wrap(err, "failed to list db restoration operations")
	}

	if len(restorationOperations) == 0 {
		logger.Info("No existing db restoration operations found for installation")
		return true, nil
	}

	operationIDs := getInstallationDBRestorationOperationIDs(restorationOperations)
	installationBackupsLocks := newInstallationDBRestorationLocks(operationIDs, instanceID, s.store, logger)

	if !installationBackupsLocks.TryLock() {
		return false, errors.Errorf("Failed to lock %d installation db restorations", len(restorationOperations))
	}
	defer installationBackupsLocks.Unlock()

	// Fetch the same elements again, now that we have the locks.
	restorationOperations, err = s.store.GetInstallationDBRestorationOperations(&model.InstallationDBRestorationFilter{
		IDs:    operationIDs,
		Paging: model.AllPagesNotDeleted(),
	})
	if err != nil {
		return false, errors.Wrapf(err, "failed to fetch %d installation db restoration operations by ids", len(restorationOperations))
	}

	if len(restorationOperations) != len(operationIDs) {
		logger.Warnf("Found only %d installation db restoration operations after locking, expected %d", len(restorationOperations), len(operationIDs))
	}

	deleted := 0
	deleting := 0

	for _, operation := range restorationOperations {
		switch operation.State {
		case model.InstallationDBRestorationStateDeleted:
			deleted++
			continue
		case model.InstallationDBRestorationStateDeletionRequested:
			deleting++
			continue
		}

		logger.Debugf("Deleting installation db restoration operation %s in state %s", operation.ID, operation.State)
		operation.State = model.InstallationDBRestorationStateDeletionRequested
		err = s.store.UpdateInstallationDBRestorationOperationState(operation)
		if err != nil {
			return false, errors.Wrapf(err, "failed to mark istallation db restoration %s for deletion", operation.ID)
		}
		deleting++
	}

	logger.Debugf("Found %d installation db restorations, %d deleting, %d deleted", len(restorationOperations), deleting, deleted)

	if deleting > 0 {
		logger.Infof("Installation db restorations deletion in progress, deleting operations %d", deleting)
		return false, nil
	}

	return true, nil
}

func (s *InstallationSupervisor) deleteMigrationOperations(installation *model.Installation, instanceID string, logger log.FieldLogger) (bool, error) {
	logger.Info("Deleting installation db migration operations")

	migrationOperations, err := s.store.GetInstallationDBMigrationOperations(&model.InstallationDBMigrationFilter{
		InstallationID: installation.ID,
		Paging:         model.AllPagesNotDeleted(),
	})
	if err != nil {
		return false, errors.Wrap(err, "failed to list db migration operations")
	}

	if len(migrationOperations) == 0 {
		logger.Info("No existing db migration operations found for installation")
		return true, nil
	}

	operationIDs := getInstallationDBMigrationOperationIDs(migrationOperations)
	installationBackupsLocks := newInstallationDBMigrationOperationLocks(operationIDs, instanceID, s.store, logger)

	if !installationBackupsLocks.TryLock() {
		return false, errors.Errorf("Failed to lock %d installation db migrations", len(migrationOperations))
	}
	defer installationBackupsLocks.Unlock()

	// Fetch the same elements again, now that we have the locks.
	migrationOperations, err = s.store.GetInstallationDBMigrationOperations(&model.InstallationDBMigrationFilter{
		IDs:    operationIDs,
		Paging: model.AllPagesNotDeleted(),
	})
	if err != nil {
		return false, errors.Wrapf(err, "failed to fetch %d installation db migration operations by ids", len(migrationOperations))
	}

	if len(migrationOperations) != len(operationIDs) {
		logger.Warnf("Found only %d installation db migration operations after locking, expected %d", len(migrationOperations), len(operationIDs))
	}

	deleted := 0
	deleting := 0

	for _, operation := range migrationOperations {
		switch operation.State {
		case model.InstallationDBMigrationStateDeleted:
			deleted++
			continue
		case model.InstallationDBMigrationStateDeletionRequested:
			deleting++
			continue
		}

		logger.Debugf("Deleting installation db migration operation %s in state %s", operation.ID, operation.State)
		operation.State = model.InstallationDBMigrationStateDeletionRequested
		err = s.store.UpdateInstallationDBMigrationOperationState(operation)
		if err != nil {
			return false, errors.Wrapf(err, "failed to mark istallation db migration %s for deletion", operation.ID)
		}
		deleting++
	}

	logger.Debugf("Found %d installation db migrations, %d deleting, %d deleted", len(migrationOperations), deleting, deleted)

	if deleting > 0 {
		logger.Infof("Installation db migrations deletion in progress, deleting operations %d", deleting)
		return false, nil
	}

	return true, nil
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
		Paging:         model.AllPagesNotDeleted(),
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

func getClusterInstallationIDs(clusterInstallations []*model.ClusterInstallation) []string {
	clusterInstallationIDs := make([]string, 0, len(clusterInstallations))
	for _, clusterInstallation := range clusterInstallations {
		clusterInstallationIDs = append(clusterInstallationIDs, clusterInstallation.ID)
	}
	return clusterInstallationIDs
}

func getInstallationBackupsIDs(backups []*model.InstallationBackup) []string {
	backupIDs := make([]string, 0, len(backups))
	for _, backup := range backups {
		backupIDs = append(backupIDs, backup.ID)
	}
	return backupIDs
}

func getInstallationDBRestorationOperationIDs(operations []*model.InstallationDBRestorationOperation) []string {
	ids := make([]string, 0, len(operations))
	for _, op := range operations {
		ids = append(ids, op.ID)
	}
	return ids
}

func getInstallationDBMigrationOperationIDs(operations []*model.InstallationDBMigrationOperation) []string {
	ids := make([]string, 0, len(operations))
	for _, op := range operations {
		ids = append(ids, op.ID)
	}
	return ids
}

func getAnnotationsIDs(annotations []*model.Annotation) []string {
	ids := make([]string, 0, len(annotations))
	for _, ann := range annotations {
		ids = append(ids, ann.ID)
	}
	return ids
}

func getAnnotationsNames(annotations []*model.Annotation) []string {
	names := make([]string, 0, len(annotations))
	for _, ann := range annotations {
		names = append(names, ann.Name)
	}
	return names
}

// dnsSwitchForHibernatingInstallation deals with dns update for hibernating installations during migration
func (s *InstallationSupervisor) dnsSwitchForHibernatingInstallation(installation *model.Installation, instanceID string, logger log.FieldLogger) string {
	endpoints, err := s.getPublicLBEndpoint(installation, logger)
	if err != nil {
		logger.WithError(err).Warn("Failed to find load balancer endpoint (nginx) for Cluster Installation")
		return model.InstallationStateDNSMigrationHibernating
	}

	err = s.aws.CreatePublicCNAME(installation.DNS, endpoints, aws.HibernatingInstallationResourceRecordIDPrefix+installation.DNS, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to create DNS CNAME record")
		return model.InstallationStateDNSMigrationHibernating
	}

	logger.Infof("Successfully configured DNS %s", installation.DNS)

	return s.waitForHibernationStable(installation, instanceID, logger)
}

func (s *InstallationSupervisor) processInstallationMetrics(installation *model.Installation, logger log.FieldLogger) error {
	if installation.State != model.InstallationStateStable &&
		installation.State != model.InstallationStateHibernating &&
		installation.State != model.InstallationStateDeleted {
		return nil
	}

	// Get the latest event of a 'requested' type to emit the correct metrics.
	events, err := s.store.GetStateChangeEvents(&model.StateChangeEventFilter{
		ResourceID:   installation.ID,
		ResourceType: model.TypeInstallation,
		NewStates:    model.AllInstallationRequestStates,
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
	case model.InstallationStateCreationRequested:
		s.metrics.InstallationCreationDurationHist.Observe(elapsedSeconds)
		logger.Debugf("Installation was created in %d seconds", int(elapsedSeconds))
	case model.InstallationStateUpdateRequested:
		s.metrics.InstallationUpdateDurationHist.Observe(elapsedSeconds)
		logger.Debugf("Installation was updated in %d seconds", int(elapsedSeconds))
	case model.InstallationStateHibernationRequested:
		s.metrics.InstallationHibernationDurationHist.Observe(elapsedSeconds)
		logger.Debugf("Installation was hibernated in %d seconds", int(elapsedSeconds))
	case model.InstallationStateWakeUpRequested:
		s.metrics.InstallationWakeUpDurationHist.Observe(elapsedSeconds)
		logger.Debugf("Installation was woken up in %d seconds", int(elapsedSeconds))
	case model.InstallationStateDeletionRequested:
		s.metrics.InstallationDeletionDurationHist.Observe(elapsedSeconds)
		logger.Debugf("Installation was deleted in %d seconds", int(elapsedSeconds))
	default:
		return errors.Errorf("failed to handle event %s with new state %s", event.Event.ID, event.StateChange.NewState)
	}

	return nil
}
