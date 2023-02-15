// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"time"

	"github.com/mattermost/mattermost-cloud/internal/events"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

// Supervisor describes the interface to notify the background jobs of an actionable change.
type Supervisor interface {
	Do() error
}

// Store describes the interface required to persist changes made via API requests.
type Store interface {
	model.InstallationDatabaseStoreInterface
	DeleteMultitenantDatabase(multitenantDatabaseID string) error

	CreateCluster(cluster *model.Cluster, annotations []*model.Annotation) error
	GetCluster(clusterID string) (*model.Cluster, error)
	GetClusterDTO(clusterID string) (*model.ClusterDTO, error)
	GetClusters(filter *model.ClusterFilter) ([]*model.Cluster, error)
	GetClusterDTOs(filter *model.ClusterFilter) ([]*model.ClusterDTO, error)
	UpdateCluster(cluster *model.Cluster) error
	LockCluster(clusterID, lockerID string) (bool, error)
	UnlockCluster(clusterID, lockerID string, force bool) (bool, error)
	LockClusterAPI(clusterID string) error
	UnlockClusterAPI(clusterID string) error
	DeleteCluster(clusterID string) error

	CreateInstallation(installation *model.Installation, annotations []*model.Annotation, dnsRecords []*model.InstallationDNS) error
	GetInstallation(installationID string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.Installation, error)
	GetInstallationDTO(installationID string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.InstallationDTO, error)
	GetInstallations(filter *model.InstallationFilter, includeGroupConfig, includeGroupConfigOverrides bool) ([]*model.Installation, error)
	GetInstallationDTOs(filter *model.InstallationFilter, includeGroupConfig, includeGroupConfigOverrides bool) ([]*model.InstallationDTO, error)
	GetInstallationsCount(filter *model.InstallationFilter) (int64, error)
	GetInstallationsStatus() (*model.InstallationsStatus, error)
	UpdateInstallation(installation *model.Installation) error
	UpdateInstallationState(installation *model.Installation) error
	LockInstallation(installationID, lockerID string) (bool, error)
	UnlockInstallation(installationID, lockerID string, force bool) (bool, error)
	LockInstallationAPI(installationID string) error
	UnlockInstallationAPI(installationID string) error
	DeleteInstallation(installationID string) error

	GetClusterInstallation(clusterInstallationID string) (*model.ClusterInstallation, error)
	GetClusterInstallations(filter *model.ClusterInstallationFilter) ([]*model.ClusterInstallation, error)
	LockClusterInstallationAPI(clusterInstallationID string) error
	UnlockClusterInstallationAPI(clusterInstallationID string) error

	CreateGroup(group *model.Group, annotations []*model.Annotation) error
	GetGroup(groupID string) (*model.Group, error)
	GetGroupDTO(groupID string) (*model.GroupDTO, error)
	GetGroups(filter *model.GroupFilter) ([]*model.Group, error)
	GetGroupDTOs(filter *model.GroupFilter) ([]*model.GroupDTO, error)
	UpdateGroup(group *model.Group, forceSequenceUpdate bool) error
	LockGroup(groupID, lockerID string) (bool, error)
	UnlockGroup(groupID, lockerID string, force bool) (bool, error)
	LockGroupAPI(groupID string) error
	UnlockGroupAPI(groupID string) error
	DeleteGroup(groupID string) error
	GetGroupStatus(groupID string) (*model.GroupStatus, error)
	CreateGroupAnnotations(groupID string, annotations []*model.Annotation) ([]*model.Annotation, error)
	DeleteGroupAnnotation(groupID string, annotationName string) error

	CreateWebhook(webhook *model.Webhook) error
	GetWebhook(webhookID string) (*model.Webhook, error)
	GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error)
	DeleteWebhook(webhookID string) error

	GetOrCreateAnnotations(annotations []*model.Annotation) ([]*model.Annotation, error)
	GetAnnotationsByName(names []string) ([]*model.Annotation, error)

	CreateClusterAnnotations(clusterID string, annotations []*model.Annotation) ([]*model.Annotation, error)
	DeleteClusterAnnotation(clusterID string, annotationName string) error

	CreateInstallationAnnotations(installationID string, annotations []*model.Annotation) ([]*model.Annotation, error)
	DeleteInstallationAnnotation(installationID string, annotationName string) error

	IsInstallationBackupRunning(installationID string) (bool, error)
	IsInstallationBackupBeingUsed(backupID string) (bool, error)
	CreateInstallationBackup(backupMeta *model.InstallationBackup) error
	UpdateInstallationBackupState(backupMeta *model.InstallationBackup) error
	GetInstallationBackup(id string) (*model.InstallationBackup, error)
	GetInstallationBackups(filter *model.InstallationBackupFilter) ([]*model.InstallationBackup, error)
	LockInstallationBackup(backupMetadataID, lockerID string) (bool, error)
	UnlockInstallationBackup(backupMetadataID, lockerID string, force bool) (bool, error)
	LockInstallationBackupAPI(backupID string) error
	UnlockInstallationBackupAPI(backupID string) error

	TriggerInstallationRestoration(installation *model.Installation, backup *model.InstallationBackup) (*model.InstallationDBRestorationOperation, error)
	GetInstallationDBRestorationOperation(id string) (*model.InstallationDBRestorationOperation, error)
	GetInstallationDBRestorationOperations(filter *model.InstallationDBRestorationFilter) ([]*model.InstallationDBRestorationOperation, error)

	MigrateClusterInstallations(clusterInstallations []*model.ClusterInstallation, targetCluster string) error
	SwitchDNS(oldCIsIDs, newCIsIDs, installationIDs []string, hibernatingInstallationIDs []string) error
	DeleteClusterInstallation(id string) error
	DeleteInActiveClusterInstallationByClusterID(clusterID string) (int64, error)
	LockInstallations(installationIDs []string, lockerID string) (bool, error)
	UnlockInstallations(installationIDs []string, lockerID string, force bool) (bool, error)
	UpdateClusterInstallation(clusterInstallation *model.ClusterInstallation) error
	TriggerInstallationDBMigration(dbMigrationOp *model.InstallationDBMigrationOperation, installation *model.Installation) (*model.InstallationDBMigrationOperation, error)
	TriggerInstallationDBMigrationRollback(dbMigrationOp *model.InstallationDBMigrationOperation, installation *model.Installation) error
	GetInstallationDBMigrationOperations(filter *model.InstallationDBMigrationFilter) ([]*model.InstallationDBMigrationOperation, error)
	GetInstallationDBMigrationOperation(id string) (*model.InstallationDBMigrationOperation, error)
	UpdateInstallationDBMigrationOperationState(dbMigration *model.InstallationDBMigrationOperation) error
	LockInstallationDBMigrationOperation(id, lockerID string) (bool, error)
	UnlockInstallationDBMigrationOperation(id, lockerID string, force bool) (bool, error)

	CreateSubscription(sub *model.Subscription) error
	GetSubscriptions(filter *model.SubscriptionsFilter) ([]*model.Subscription, error)
	GetSubscription(subID string) (*model.Subscription, error)
	DeleteSubscription(subID string) error

	GetStateChangeEvents(filter *model.StateChangeEventFilter) ([]*model.StateChangeEventData, error)

	AddInstallationDomain(installation *model.Installation, dnsRecord *model.InstallationDNS) error
	GetInstallationDNS(id string) (*model.InstallationDNS, error)
	SwitchPrimaryInstallationDomain(installationID string, installationDNSID string) error
	GetDNSRecordsForInstallation(installationID string) ([]*model.InstallationDNS, error)
}

// Provisioner describes the interface required to communicate with the Kubernetes cluster.
type Provisioner interface {
	ProvisionerType() string
	ExecClusterInstallationCLI(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) ([]byte, error, error)
	ExecMMCTL(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) ([]byte, error)
	ExecMattermostCLI(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) ([]byte, error)
	GetClusterResources(*model.Cluster, bool, log.FieldLogger) (*k8s.ClusterResources, error)
}

// AwsClient describes the interface required to communicate with the AWS
type AwsClient interface {
	SwitchClusterTags(clusterID string, targetClusterID string, logger log.FieldLogger) error
	SecretsManagerValidateExternalDatabaseSecret(name string) error
	RDSDBCLusterExists(awsID string) (bool, error)
}

// DBProvider describes the interface required to get database for specific installation and specified type.
type DBProvider interface {
	GetDatabase(installationID, dbType string) model.Database
}

// EventProducer produces Provisioners' state change events.
type EventProducer interface {
	ProduceInstallationStateChangeEvent(installation *model.Installation, oldState string, extraDataFields ...events.DataField) error
	ProduceClusterStateChangeEvent(cluster *model.Cluster, oldState string, extraDataFields ...events.DataField) error
}

// Metrics exposes metrics from API usage.
type Metrics interface {
	IncrementAPIRequest()
	ObserveAPIEndpointDuration(handler, method string, statusCode int, elapsed float64)
}

// Context provides the API with all necessary data and interfaces for responding to requests.
//
// It is cloned before each request, allowing per-request changes such as logger annotations.
type Context struct {
	Store                             Store
	Supervisor                        Supervisor
	Provisioner                       ProvisionerOption
	DBProvider                        DBProvider
	EventProducer                     EventProducer
	AwsClient                         AwsClient
	Metrics                           Metrics
	Logger                            log.FieldLogger
	InstallationDeletionExpiryDefault time.Duration
	RequestID                         string
	Environment                       string
}

// Clone creates a shallow copy of context, allowing clones to apply per-request changes.
func (c *Context) Clone() *Context {
	return &Context{
		Store:                             c.Store,
		Supervisor:                        c.Supervisor,
		Provisioner:                       c.Provisioner,
		DBProvider:                        c.DBProvider,
		EventProducer:                     c.EventProducer,
		AwsClient:                         c.AwsClient,
		Metrics:                           c.Metrics,
		Logger:                            c.Logger,
		InstallationDeletionExpiryDefault: c.InstallationDeletionExpiryDefault,
	}
}
