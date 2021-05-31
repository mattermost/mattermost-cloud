// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor_test

import (
	"fmt"
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/provisioner"

	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/mattermost/mattermost-cloud/internal/metrics"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/utils"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	mmv1alpha1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1alpha1"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

type mockInstallationStore struct {
	Installation                     *model.Installation
	UnlockedInstallationsPendingWork []*model.Installation

	Group *model.Group

	UnlockChan              chan interface{}
	UpdateInstallationCalls int

	mockMultitenantDBStore
}

var cloudMetrics = metrics.New()

func (s *mockInstallationStore) GetClusters(clusterFilter *model.ClusterFilter) ([]*model.Cluster, error) {
	return nil, nil
}

func (s *mockInstallationStore) GetCluster(id string) (*model.Cluster, error) {
	return nil, nil
}

func (s *mockInstallationStore) UpdateCluster(cluster *model.Cluster) error {
	return nil
}

func (s *mockInstallationStore) LockCluster(clusterID, lockerID string) (bool, error) {
	return true, nil
}

func (s *mockInstallationStore) UnlockCluster(clusterID string, lockerID string, force bool) (bool, error) {
	return true, nil
}

func (s *mockInstallationStore) GetInstallation(installationID string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.Installation, error) {
	return s.Installation, nil
}

func (s *mockInstallationStore) GetUnlockedInstallationsPendingWork() ([]*model.Installation, error) {
	return s.UnlockedInstallationsPendingWork, nil
}

func (s *mockInstallationStore) UpdateInstallation(installation *model.Installation) error {
	s.UpdateInstallationCalls++
	return nil
}

func (s *mockInstallationStore) UpdateInstallationGroupSequence(installation *model.Installation) error {
	return nil
}

func (s *mockInstallationStore) UpdateInstallationState(installation *model.Installation) error {
	s.UpdateInstallationCalls++
	return nil
}

func (s *mockInstallationStore) UpdateInstallationCRVersion(installationID, crVersion string) error {
	return nil
}

func (s *mockInstallationStore) LockInstallation(installationID, lockerID string) (bool, error) {
	return true, nil
}

func (s *mockInstallationStore) UnlockInstallation(installationID, lockerID string, force bool) (bool, error) {
	if s.UnlockChan != nil {
		close(s.UnlockChan)
	}
	return true, nil
}

func (s *mockInstallationStore) DeleteInstallation(installationID string) error {
	return nil
}

func (s *mockInstallationStore) CreateClusterInstallation(clusterInstallation *model.ClusterInstallation) error {
	return nil
}

func (s *mockInstallationStore) GetClusterInstallation(clusterInstallationID string) (*model.ClusterInstallation, error) {
	return nil, nil
}

func (s *mockInstallationStore) GetClusterInstallations(*model.ClusterInstallationFilter) ([]*model.ClusterInstallation, error) {
	return []*model.ClusterInstallation{{
		ID:              model.NewID(),
		ClusterID:       model.NewID(),
		InstallationID:  s.Installation.ID,
		Namespace:       s.Installation.ID,
		State:           "stable",
		CreateAt:        s.Installation.CreateAt,
		DeleteAt:        s.Installation.DeleteAt,
		APISecurityLock: false,
	},
	}, nil
}

func (s *mockInstallationStore) LockClusterInstallations(clusterInstallationID []string, lockerID string) (bool, error) {
	return true, nil
}

func (s *mockInstallationStore) UnlockClusterInstallations(clusterInstallationID []string, lockerID string, force bool) (bool, error) {
	return true, nil
}

func (s *mockInstallationStore) UpdateClusterInstallation(clusterInstallation *model.ClusterInstallation) error {
	return nil
}

func (s *mockInstallationStore) GetGroup(groupId string) (*model.Group, error) {
	return nil, nil
}

func (s *mockInstallationStore) LockGroup(groupID, lockerID string) (bool, error) {
	return true, nil
}

func (s *mockInstallationStore) UnlockGroup(groupID, lockerID string, force bool) (bool, error) {
	if s.UnlockChan != nil {
		close(s.UnlockChan)
	}
	return true, nil
}

func (s *mockInstallationStore) GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error) {
	return nil, nil
}

func (s *mockInstallationStore) GetAnnotationsForInstallation(installationID string) ([]*model.Annotation, error) {
	return nil, nil
}

func (s *mockInstallationStore) GetInstallationBackups(filter *model.InstallationBackupFilter) ([]*model.InstallationBackup, error) {
	return nil, nil
}

func (s *mockInstallationStore) UpdateInstallationBackupState(backup *model.InstallationBackup) error {
	return nil
}

func (s *mockInstallationStore) LockInstallationBackups(backupIDs []string, lockerID string) (bool, error) {
	return true, nil
}

func (s *mockInstallationStore) UnlockInstallationBackups(backupIDs []string, lockerID string, force bool) (bool, error) {
	return true, nil
}

func (s *mockInstallationStore) GetInstallationDBMigrationOperations(filter *model.InstallationDBMigrationFilter) ([]*model.InstallationDBMigrationOperation, error) {
	return nil, nil
}

func (s *mockInstallationStore) UpdateInstallationDBMigrationOperationState(operation *model.InstallationDBMigrationOperation) error {
	return nil
}

func (s *mockInstallationStore) LockInstallationDBMigrationOperations(backupIDs []string, lockerID string) (bool, error) {
	return true, nil
}

func (s *mockInstallationStore) UnlockInstallationDBMigrationOperations(backupIDs []string, lockerID string, force bool) (bool, error) {
	return true, nil
}

func (s *mockInstallationStore) GetInstallationDBRestorationOperations(filter *model.InstallationDBRestorationFilter) ([]*model.InstallationDBRestorationOperation, error) {
	return nil, nil
}
func (s *mockInstallationStore) UpdateInstallationDBRestorationOperationState(operation *model.InstallationDBRestorationOperation) error {
	return nil
}

func (s *mockInstallationStore) LockInstallationDBRestorationOperations(backupIDs []string, lockerID string) (bool, error) {
	return true, nil
}

func (s *mockInstallationStore) UnlockInstallationDBRestorationOperations(backupIDs []string, lockerID string, force bool) (bool, error) {
	return true, nil
}

type mockMultitenantDBStore struct{}

func (m *mockMultitenantDBStore) GetMultitenantDatabase(multitenantdatabaseID string) (*model.MultitenantDatabase, error) {
	return nil, nil
}

func (m *mockMultitenantDBStore) GetMultitenantDatabases(filter *model.MultitenantDatabaseFilter) ([]*model.MultitenantDatabase, error) {
	return nil, nil
}

func (m *mockMultitenantDBStore) GetInstallationsTotalDatabaseWeight(installationIDs []string) (float64, error) {
	return 0, nil
}

func (m *mockMultitenantDBStore) CreateMultitenantDatabase(multitenantDatabase *model.MultitenantDatabase) error {
	return nil
}

func (m *mockMultitenantDBStore) LockMultitenantDatabase(multitenantdatabaseID, lockerID string) (bool, error) {
	return true, nil
}

func (m *mockMultitenantDBStore) UnlockMultitenantDatabase(multitenantdatabaseID, lockerID string, force bool) (bool, error) {
	return true, nil
}

func (m *mockMultitenantDBStore) UpdateMultitenantDatabase(multitenantDatabase *model.MultitenantDatabase) error {
	return nil
}

func (m *mockMultitenantDBStore) GetMultitenantDatabaseForInstallationID(installationID string) (*model.MultitenantDatabase, error) {
	return nil, nil
}

func (m mockMultitenantDBStore) LockMultitenantDatabases(ids []string, lockerID string) (bool, error) {
	return true, nil
}

func (m mockMultitenantDBStore) UnlockMultitenantDatabases(ids []string, lockerID string, force bool) (bool, error) {
	return true, nil
}

func (m mockMultitenantDBStore) GetSingleTenantDatabaseConfigForInstallation(installationID string) (*model.SingleTenantDatabaseConfig, error) {
	return nil, nil
}

type mockInstallationProvisioner struct {
	UseCustomClusterResources bool
	CustomClusterResources    *k8s.ClusterResources
}

func (p *mockInstallationProvisioner) ClusterInstallationProvisioner(version string) provisioner.ClusterInstallationProvisioner {
	return p
}

func (p *mockInstallationProvisioner) IsResourceReady(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation) (bool, error) {
	return true, nil
}

func (p *mockInstallationProvisioner) CreateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	return nil
}

func (p *mockInstallationProvisioner) UpdateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	return nil
}

func (p *mockInstallationProvisioner) EnsureCRMigrated(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation) (bool, error) {
	return true, nil
}

func (p *mockInstallationProvisioner) HibernateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	return nil
}

func (p *mockInstallationProvisioner) DeleteOldClusterInstallationLicenseSecrets(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	return nil
}

func (p *mockInstallationProvisioner) DeleteClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	return nil
}

func (p *mockInstallationProvisioner) VerifyClusterInstallationMatchesConfig(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) (bool, error) {
	return true, nil
}

func (p *mockInstallationProvisioner) GetClusterResources(cluster *model.Cluster, onlySchedulable bool) (*k8s.ClusterResources, error) {
	if p.UseCustomClusterResources {
		return p.CustomClusterResources, nil
	}

	return &k8s.ClusterResources{
			MilliTotalCPU:    100000,
			MilliUsedCPU:     100,
			MilliTotalMemory: 100000000000000,
			MilliUsedMemory:  100,
		},
		nil
}

func (p *mockInstallationProvisioner) GetPublicLoadBalancerEndpoint(cluster *model.Cluster, namespace string) (string, error) {
	return "example.elb.us-east-1.amazonaws.com", nil
}

func (p *mockInstallationProvisioner) RefreshSecrets(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	return nil
}

func (p *mockInstallationProvisioner) PrepareClusterUtilities(cluster *model.Cluster, installation *model.Installation, store model.ClusterUtilityDatabaseStoreInterface, awsClient aws.AWS) error {
	return nil
}

// TODO(gsagula): this can be replaced with /internal/mocks/aws-tools/AWS.go so that inputs and other variants
// can be tested.
type mockAWS struct{}

func (a *mockAWS) GetCertificateSummaryByTag(key, value string, logger log.FieldLogger) (*acm.CertificateSummary, error) {
	return nil, nil
}

func (a *mockAWS) GetCloudEnvironmentName() string {
	return "test"
}

func (a *mockAWS) DynamoDBEnsureTableDeleted(tableName string, logger log.FieldLogger) error {
	return nil
}

func (a *mockAWS) S3EnsureBucketDeleted(bucketName string, logger log.FieldLogger) error {
	return nil
}

func (a *mockAWS) S3EnsureObjectDeleted(bucketName, path string) error {
	return nil
}

func (a *mockAWS) GetAndClaimVpcResources(clusterID, owner string, logger log.FieldLogger) (aws.ClusterResources, error) {
	return aws.ClusterResources{}, nil
}

func (a *mockAWS) GetVpcResources(clusterID string, logger log.FieldLogger) (aws.ClusterResources, error) {
	return aws.ClusterResources{}, nil
}

func (a *mockAWS) ReleaseVpc(clusterID string, logger log.FieldLogger) error {
	return nil
}

func (a *mockAWS) AttachPolicyToRole(roleName, policyName string, logger log.FieldLogger) error {
	return nil
}

func (a *mockAWS) DetachPolicyFromRole(roleName, policyName string, logger log.FieldLogger) error {
	return nil
}

func (a *mockAWS) GetPrivateZoneDomainName(logger log.FieldLogger) (string, error) {
	return "test.domain", nil
}

func (a *mockAWS) GetTagByKeyAndZoneID(key string, id string, logger log.FieldLogger) (*aws.Tag, error) {
	return &aws.Tag{
		Key:   "examplekey",
		Value: "examplevalue",
	}, nil
}
func (a *mockAWS) GetPrivateHostedZoneID() string {
	return "EXAMPLER53ID"
}

func (a *mockAWS) CreatePrivateCNAME(dnsName string, dnsEndpoints []string, logger log.FieldLogger) error {
	return nil
}

func (a *mockAWS) CreatePublicCNAME(dnsName string, dnsEndpoints []string, logger log.FieldLogger) error {
	return nil
}

func (a *mockAWS) UpdatePublicRecordIDForCNAME(dnsName, newID string, logger log.FieldLogger) error {
	return nil
}

func (a *mockAWS) IsProvisionedPrivateCNAME(dnsName string, logger log.FieldLogger) bool {
	return false
}

func (a *mockAWS) DeletePrivateCNAME(dnsName string, logger log.FieldLogger) error {
	return nil
}

func (a *mockAWS) DeletePublicCNAME(dnsName string, logger log.FieldLogger) error {
	return nil
}

func (a *mockAWS) TagResource(resourceID, key, value string, logger log.FieldLogger) error {
	return nil
}

func (a *mockAWS) UntagResource(resourceID, key, value string, logger log.FieldLogger) error {
	return nil
}

func (a *mockAWS) IsValidAMI(AMIID string, logger log.FieldLogger) (bool, error) {
	return true, nil
}

func (a *mockAWS) S3FilestoreProvision(installationID string, logger log.FieldLogger) error {
	return nil
}

func (a *mockAWS) S3FilestoreTeardown(installationID string, keepBucket bool, logger log.FieldLogger) error {
	return nil
}

func (a *mockAWS) SecretsManagerGetIAMAccessKey(installationID string, logger log.FieldLogger) (*aws.IAMAccessKey, error) {
	return nil, nil
}

func (a *mockAWS) GenerateBifrostUtilitySecret(clusterID string, logger log.FieldLogger) (*corev1.Secret, error) {
	return nil, nil
}

func (a *mockAWS) GetCIDRByVPCTag(vpcTagName string, logger log.FieldLogger) (string, error) {
	return "", nil
}
func (a *mockAWS) S3LargeCopy(srcBucketName, srcKey, destBucketName, destKey *string) error {
	return nil
}

func (a *mockAWS) GetMultitenantBucketNameForInstallation(installationID string, store model.InstallationDatabaseStoreInterface) (string, error) {
	return "", nil
}
func (a *mockAWS) GetVpcResourcesByVpcID(vpcID string, logger log.FieldLogger) (aws.ClusterResources, error) {
	return aws.ClusterResources{}, nil
}
func (a *mockAWS) TagResourcesByCluster(clusterResources aws.ClusterResources, clusterID string, owner string, logger log.FieldLogger) error {
	return nil
}

func (a *mockAWS) SecretsManagerGetPGBouncerAuthUserPassword(vpcID string) (string, error) {
	return "password", nil
}

func TestInstallationSupervisorDo(t *testing.T) {
	standardSchedulingOptions := supervisor.NewInstallationSupervisorSchedulingOptions(false, 80, 0)

	t.Run("no installations pending work", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockInstallationStore{}

		supervisor := supervisor.NewInstallationSupervisor(mockStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)
		err := supervisor.Do()
		require.NoError(t, err)

		require.Equal(t, 0, mockStore.UpdateInstallationCalls)
	})

	t.Run("mock installation creation", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockInstallationStore{}

		mockStore.UnlockedInstallationsPendingWork = []*model.Installation{{
			ID:    model.NewID(),
			State: model.InstallationStateDeletionRequested,
		}}
		mockStore.Installation = mockStore.UnlockedInstallationsPendingWork[0]
		mockStore.UnlockChan = make(chan interface{})

		supervisor := supervisor.NewInstallationSupervisor(mockStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)
		err := supervisor.Do()
		require.NoError(t, err)

		<-mockStore.UnlockChan
		require.Equal(t, 1, mockStore.UpdateInstallationCalls)
	})
}

func TestInstallationSupervisor(t *testing.T) {
	standardSchedulingOptions := supervisor.NewInstallationSupervisorSchedulingOptions(false, 80, 0)

	expectInstallationState := func(t *testing.T, sqlStore *store.SQLStore, installation *model.Installation, expectedState string) {
		t.Helper()

		installation, err := sqlStore.GetInstallation(installation.ID, false, false)
		require.NoError(t, err)
		require.Equal(t, expectedState, installation.State)
	}

	expectClusterInstallations := func(t *testing.T, sqlStore *store.SQLStore, installation *model.Installation, expectedCount int, state string) {
		t.Helper()
		clusterInstallations, err := sqlStore.GetClusterInstallations(&model.ClusterInstallationFilter{
			Paging:         model.AllPagesNotDeleted(),
			InstallationID: installation.ID,
		})
		require.NoError(t, err)
		require.Len(t, clusterInstallations, expectedCount)
		for _, clusterInstallation := range clusterInstallations {
			require.Equal(t, state, clusterInstallation.State)
		}
	}

	expectClusterInstallationsOnCluster := func(t *testing.T, sqlStore *store.SQLStore, cluster *model.Cluster, expectedCount int) {
		t.Helper()
		clusterInstallations, err := sqlStore.GetClusterInstallations(&model.ClusterInstallationFilter{
			Paging:    model.AllPagesNotDeleted(),
			ClusterID: cluster.ID,
		})
		require.NoError(t, err)
		require.Len(t, clusterInstallations, expectedCount)
	}

	standardStableTestCluster := func() *model.Cluster {
		return &model.Cluster{
			State:              model.ClusterStateStable,
			AllowInstallations: true,
			ProvisionerMetadataKops: &model.KopsMetadata{
				MasterCount:  1,
				NodeMinCount: 1,
				NodeMaxCount: 5,
			},
		}
	}

	t.Run("unexpected state", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateStable,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateStable,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateStable)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateStable)
	})

	t.Run("state has changed since installation was selected to be worked on", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateCreationInProgress,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		// The stored installation is InstallationStateCreationInProgress, so we
		// will pass in an installation with state of
		// InstallationStateCreationRequested to simulate stale state.
		installation.State = model.InstallationStateCreationRequested

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationInProgress)
	})

	t.Run("creation requested, cluster installations not yet created, no clusters", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateCreationRequested,
		}

		err := sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationNoCompatibleClusters)
		expectClusterInstallations(t, sqlStore, installation, 0, "")
	})

	t.Run("creation requested, cluster installations not yet created, cluster doesn't allow scheduling", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		cluster.AllowInstallations = false
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateCreationRequested,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationNoCompatibleClusters)
		expectClusterInstallations(t, sqlStore, installation, 0, "")
	})

	t.Run("creation requested, cluster installations not yet created, no empty clusters", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: model.NewID(),
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateStable,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateCreationRequested,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationNoCompatibleClusters)
		expectClusterInstallations(t, sqlStore, installation, 0, "")
	})

	t.Run("creation requested, cluster installations reconciling", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateCreationRequested,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateReconciling,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationInProgress)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateReconciling)
	})

	t.Run("creation requested, cluster installations reconciling", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateCreationRequested,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateReconciling,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationInProgress)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateReconciling)
	})

	t.Run("creation DNS, cluster installations reconciling", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateCreationDNS,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateCreationRequested,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationInProgress)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateCreationRequested)
	})

	t.Run("creation requested, cluster installations stable", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateCreationRequested,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateStable,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateStable)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateStable)
	})

	t.Run("creation requested, cluster installations stable, in group with different sequence", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		group := &model.Group{
			ID:       model.NewID(),
			Sequence: 2,
			Version:  "gversion",
			Image:    "gImage",
		}

		err = sqlStore.CreateGroup(group)
		require.NoError(t, err)

		owner := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &group.ID,
			State:    model.InstallationStateCreationRequested,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateStable,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateStable)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateStable)

		installation, err = sqlStore.GetInstallation(installation.ID, true, false)
		require.NoError(t, err)
		assert.True(t, installation.InstallationSequenceMatchesMergedGroupSequence())
	})

	t.Run("pre provisioning requested, cluster installations reconciling", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateCreationPreProvisioning,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateCreationRequested,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationInProgress)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateCreationRequested)
	})

	t.Run("creation requested, cluster installations failed", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateCreationPreProvisioning,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateCreationFailed,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationFailed)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateCreationFailed)
	})

	t.Run("creation in progress, cluster installations reconciling", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateCreationInProgress,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateCreationRequested,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationInProgress)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateCreationRequested)
	})

	t.Run("creation in progress, cluster installations stable", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateCreationInProgress,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateStable,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateStable)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateStable)
	})

	t.Run("creation in progress, cluster installations stable, in group with same sequence", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		group := &model.Group{
			ID:      model.NewID(),
			Version: "gversion",
			Image:   "gImage",
		}

		err = sqlStore.CreateGroup(group)
		require.NoError(t, err)
		// Group Sequence always set to 0 when created so we need to update it.
		err = sqlStore.UpdateGroup(group, true)
		require.NoError(t, err)

		owner := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &group.ID,
			State:    model.InstallationStateCreationInProgress,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		installation.MergeWithGroup(group, false)
		installation.SyncGroupAndInstallationSequence()
		err = sqlStore.UpdateInstallationGroupSequence(installation)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateStable,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateStable)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateStable)

		installation, err = sqlStore.GetInstallation(installation.ID, true, false)
		require.NoError(t, err)
		assert.True(t, installation.InstallationSequenceMatchesMergedGroupSequence())
	})

	t.Run("creation in progress, cluster installations failed", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateCreationInProgress,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateCreationFailed,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationFailed)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateCreationFailed)
	})

	t.Run("creation final tasks, cluster installations stable", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateCreationFinalTasks,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateStable,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateStable)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateStable)
	})

	t.Run("no compatible clusters, cluster installations not yet created, no clusters", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateCreationNoCompatibleClusters,
		}

		err := sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationNoCompatibleClusters)
		expectClusterInstallations(t, sqlStore, installation, 0, "")
	})

	t.Run("no compatible clusters, cluster installations not yet created, no available clusters", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: model.NewID(),
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateStable,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateCreationNoCompatibleClusters,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationNoCompatibleClusters)
		expectClusterInstallations(t, sqlStore, installation, 0, "")
	})

	t.Run("no compatible clusters, cluster installations not yet created, available cluster", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateCreationNoCompatibleClusters,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationInProgress)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateCreationRequested)
	})

	t.Run("update requested, cluster installations stable", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateUpdateRequested,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateStable,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateUpdateInProgress)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateReconciling)
	})

	t.Run("update requested, cluster installations stable, in group with different sequence", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		group := &model.Group{
			ID:       model.NewID(),
			Sequence: 2,
			Version:  "gversion",
			Image:    "gImage",
		}

		err = sqlStore.CreateGroup(group)
		require.NoError(t, err)

		owner := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &group.ID,
			State:    model.InstallationStateUpdateRequested,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateStable,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateUpdateInProgress)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateReconciling)

		installation, err = sqlStore.GetInstallation(installation.ID, true, false)
		require.NoError(t, err)
		assert.True(t, installation.InstallationSequenceMatchesMergedGroupSequence())
	})

	t.Run("update in progress, cluster installations reconciling", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateUpdateInProgress,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateReconciling,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateUpdateInProgress)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateReconciling)
	})

	t.Run("update requested, cluster installations reconciling, in group with different sequence", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		group := &model.Group{
			ID:       model.NewID(),
			Sequence: 2,
			Version:  "gversion",
			Image:    "gImage",
		}

		err = sqlStore.CreateGroup(group)
		require.NoError(t, err)

		owner := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &group.ID,
			State:    model.InstallationStateUpdateInProgress,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateReconciling,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateUpdateRequested)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateReconciling)

		installation, err = sqlStore.GetInstallation(installation.ID, true, false)
		require.NoError(t, err)
		assert.False(t, installation.InstallationSequenceMatchesMergedGroupSequence())
	})

	t.Run("update in progress, cluster installations stable", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateUpdateInProgress,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateStable,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateStable)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateStable)
	})

	t.Run("update requested, cluster installations stable, in group with same sequence", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		group := &model.Group{
			ID:      model.NewID(),
			Version: "gversion",
			Image:   "gImage",
		}

		err = sqlStore.CreateGroup(group)
		require.NoError(t, err)
		// Group Sequence always set to 0 when created so we need to update it
		// by calling group update once.
		oldSequence := group.Sequence
		err = sqlStore.UpdateGroup(group, true)
		require.NoError(t, err)
		require.NotEqual(t, oldSequence, group.Sequence)

		owner := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &group.ID,
			State:    model.InstallationStateUpdateInProgress,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		installation.MergeWithGroup(group, false)
		installation.SyncGroupAndInstallationSequence()
		err = sqlStore.UpdateInstallationGroupSequence(installation)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateStable,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateStable)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateStable)

		installation, err = sqlStore.GetInstallation(installation.ID, true, false)
		require.NoError(t, err)
		assert.True(t, installation.InstallationSequenceMatchesMergedGroupSequence())
	})

	t.Run("hibernation requested, cluster installations stable", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateHibernationRequested,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateStable,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateHibernationInProgress)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateReconciling)
	})

	t.Run("hibernation in progress, cluster installations reconciling", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateHibernationInProgress,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateReconciling,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateHibernationInProgress)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateReconciling)
	})

	t.Run("hibernation in progress, cluster installations stable", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateHibernationInProgress,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateStable,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateHibernating)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateStable)
	})

	t.Run("wake up requested, cluster installations stable", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateWakeUpRequested,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateStable,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateUpdateInProgress)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateReconciling)
	})

	t.Run("deletion requested, cluster installations stable", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateDeletionRequested,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateStable,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateDeletionInProgress)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateDeletionRequested)
	})

	t.Run("deletion requested, cluster installations deleting", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateDeletionRequested,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateDeletionRequested,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateDeletionInProgress)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateDeletionRequested)
	})

	t.Run("deletion in progress, cluster installations failed", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateDeletionInProgress,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateDeletionFailed,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateDeletionFailed)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateDeletionFailed)
	})

	t.Run("deletion requested, cluster installations failed, so retry", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateDeletionRequested,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateDeletionFailed,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateDeletionInProgress)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateDeletionRequested)
	})

	t.Run("deletion requested, delete backups", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateDeletionRequested,
		}
		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateDeleted,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		backup := &model.InstallationBackup{
			InstallationID:        installation.ID,
			ClusterInstallationID: clusterInstallation.ID,
			State:                 model.InstallationBackupStateBackupSucceeded,
		}
		err = sqlStore.CreateInstallationBackup(backup)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateDeletionFinalCleanup)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateDeleted)
		fetchedBackup, err := sqlStore.GetInstallationBackup(backup.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationBackupStateDeletionRequested, fetchedBackup.State)
	})

	t.Run("deletion requested, delete migrations and restorations", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateDeletionRequested,
		}
		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateDeleted,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		restorationOP := &model.InstallationDBRestorationOperation{
			InstallationID:        installation.ID,
			ClusterInstallationID: clusterInstallation.ID,
			State:                 model.InstallationDBRestorationStateSucceeded,
		}
		err = sqlStore.CreateInstallationDBRestorationOperation(restorationOP)
		require.NoError(t, err)

		migrationOP := &model.InstallationDBMigrationOperation{
			InstallationID: installation.ID,
			State:          model.InstallationDBMigrationStateSucceeded,
		}
		err = sqlStore.CreateInstallationDBMigrationOperation(migrationOP)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateDeletionFinalCleanup)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateDeleted)
		fetchedRestoration, err := sqlStore.GetInstallationDBRestorationOperation(restorationOP.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationDBRestorationStateDeletionRequested, fetchedRestoration.State)
		fetchedMigration, err := sqlStore.GetInstallationDBMigrationOperation(migrationOP.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationDBMigrationStateDeletionRequested, fetchedMigration.State)
	})

	t.Run("creation requested, cluster installations deleted", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateDeletionRequested,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateDeleted,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateDeleted)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateDeleted)
	})

	t.Run("multitenant", func(t *testing.T) {
		t.Run("creation requested, cluster installations not yet created, available cluster", func(t *testing.T) {
			logger := testlib.MakeLogger(t)
			sqlStore := store.MakeTestSQLStore(t, logger)
			supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

			cluster := standardStableTestCluster()
			err := sqlStore.CreateCluster(cluster, nil)
			require.NoError(t, err)

			owner := model.NewID()
			groupID := model.NewID()
			installation := &model.Installation{
				OwnerID:  owner,
				Version:  "version",
				DNS:      "dns.example.com",
				Size:     mmv1alpha1.Size100String,
				Affinity: model.InstallationAffinityMultiTenant,
				GroupID:  &groupID,
				State:    model.InstallationStateCreationRequested,
			}

			err = sqlStore.CreateInstallation(installation, nil)
			require.NoError(t, err)

			supervisor.Supervise(installation)
			expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationInProgress)
			expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateCreationRequested)
			expectClusterInstallationsOnCluster(t, sqlStore, cluster, 1)
		})

		t.Run("creation requested, cluster installations not yet created, 3 installations, available cluster", func(t *testing.T) {
			logger := testlib.MakeLogger(t)
			sqlStore := store.MakeTestSQLStore(t, logger)
			supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

			cluster := standardStableTestCluster()
			err := sqlStore.CreateCluster(cluster, nil)
			require.NoError(t, err)

			for i := 1; i < 3; i++ {
				t.Run(fmt.Sprintf("cluster-%d", i), func(t *testing.T) {
					owner := model.NewID()
					groupID := model.NewID()
					installation := &model.Installation{
						OwnerID:  owner,
						Version:  "version",
						DNS:      fmt.Sprintf("dns%d.example.com", i),
						Size:     mmv1alpha1.Size100String,
						Affinity: model.InstallationAffinityMultiTenant,
						GroupID:  &groupID,
						State:    model.InstallationStateCreationRequested,
					}

					err = sqlStore.CreateInstallation(installation, nil)
					require.NoError(t, err)

					supervisor.Supervise(installation)
					expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationInProgress)
					expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateCreationRequested)
					expectClusterInstallationsOnCluster(t, sqlStore, cluster, i)
				})
			}
		})

		t.Run("creation requested, cluster installations not yet created, 1 isolated and 1 multitenant, available cluster", func(t *testing.T) {
			logger := testlib.MakeLogger(t)
			sqlStore := store.MakeTestSQLStore(t, logger)
			supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

			cluster := standardStableTestCluster()
			err := sqlStore.CreateCluster(cluster, nil)
			require.NoError(t, err)

			owner := model.NewID()
			groupID := model.NewID()
			isolatedInstallation := &model.Installation{
				OwnerID:  owner,
				Version:  "version",
				DNS:      "iso-dns.example.com",
				Size:     mmv1alpha1.Size100String,
				Affinity: model.InstallationAffinityIsolated,
				GroupID:  &groupID,
				State:    model.InstallationStateCreationRequested,
			}

			err = sqlStore.CreateInstallation(isolatedInstallation, nil)
			require.NoError(t, err)

			supervisor.Supervise(isolatedInstallation)
			expectInstallationState(t, sqlStore, isolatedInstallation, model.InstallationStateCreationInProgress)
			expectClusterInstallations(t, sqlStore, isolatedInstallation, 1, model.ClusterInstallationStateCreationRequested)
			expectClusterInstallationsOnCluster(t, sqlStore, cluster, 1)

			owner = model.NewID()
			groupID = model.NewID()
			multitenantInstallation := &model.Installation{
				OwnerID:  owner,
				Version:  "version",
				DNS:      "mt-dns.example.com",
				Size:     mmv1alpha1.Size100String,
				Affinity: model.InstallationAffinityMultiTenant,
				GroupID:  &groupID,
				State:    model.InstallationStateCreationRequested,
			}

			err = sqlStore.CreateInstallation(multitenantInstallation, nil)
			require.NoError(t, err)

			supervisor.Supervise(multitenantInstallation)
			expectInstallationState(t, sqlStore, multitenantInstallation, model.InstallationStateCreationNoCompatibleClusters)
			expectClusterInstallations(t, sqlStore, multitenantInstallation, 0, "")
			expectClusterInstallationsOnCluster(t, sqlStore, cluster, 1)
		})

		t.Run("creation requested, cluster installations not yet created, insufficient cluster resources", func(t *testing.T) {
			logger := testlib.MakeLogger(t)
			sqlStore := store.MakeTestSQLStore(t, logger)
			mockInstallationProvisioner := &mockInstallationProvisioner{
				UseCustomClusterResources: true,
				CustomClusterResources: &k8s.ClusterResources{
					MilliTotalCPU:    200,
					MilliUsedCPU:     100,
					MilliTotalMemory: 200,
					MilliUsedMemory:  100,
				},
			}
			supervisor := supervisor.NewInstallationSupervisor(sqlStore, mockInstallationProvisioner, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

			cluster := standardStableTestCluster()
			err := sqlStore.CreateCluster(cluster, nil)
			require.NoError(t, err)

			owner := model.NewID()
			groupID := model.NewID()
			installation := &model.Installation{
				OwnerID:  owner,
				Version:  "version",
				DNS:      "dns.example.com",
				Size:     mmv1alpha1.Size100String,
				Affinity: model.InstallationAffinityMultiTenant,
				GroupID:  &groupID,
				State:    model.InstallationStateCreationRequested,
			}

			err = sqlStore.CreateInstallation(installation, nil)
			require.NoError(t, err)

			supervisor.Supervise(installation)
			expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationNoCompatibleClusters)
			expectClusterInstallations(t, sqlStore, installation, 0, "")
			expectClusterInstallationsOnCluster(t, sqlStore, cluster, 0)
		})
	})

	t.Run("creation requested, cluster installations not yet created, insufficient cluster resources, but scale", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		mockInstallationProvisioner := &mockInstallationProvisioner{
			UseCustomClusterResources: true,
			CustomClusterResources: &k8s.ClusterResources{
				MilliTotalCPU:    200,
				MilliUsedCPU:     100,
				MilliTotalMemory: 200,
				MilliUsedMemory:  100,
			},
		}
		schedulingOptions := supervisor.NewInstallationSupervisorSchedulingOptions(false, 80, 2)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, mockInstallationProvisioner, &mockAWS{}, "instanceID", false, false, schedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityMultiTenant,
			GroupID:  &groupID,
			State:    model.InstallationStateCreationRequested,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationInProgress)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateCreationRequested)
		expectClusterInstallationsOnCluster(t, sqlStore, cluster, 1)
	})

	t.Run("creation requested, cluster installations not yet created, use balanced installation scheduling", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		schedulingOptions := supervisor.NewInstallationSupervisorSchedulingOptions(true, 80, 0)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, schedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

		cluster1 := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster1, nil)
		require.NoError(t, err)
		cluster2 := standardStableTestCluster()
		err = sqlStore.CreateCluster(cluster2, nil)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityMultiTenant,
			GroupID:  &groupID,
			State:    model.InstallationStateCreationRequested,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationInProgress)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateCreationRequested)
		expectClusterInstallationsOnCluster(t, sqlStore, cluster1, 1)
		expectClusterInstallationsOnCluster(t, sqlStore, cluster2, 0)
	})

	t.Run("cluster with proper annotations selected", func(t *testing.T) {
		annotations := []*model.Annotation{
			{Name: "multi-tenant"}, {Name: "customer-abc"},
		}

		installationInCreationRequestedState := func() *model.Installation {
			groupID := model.NewID()

			return &model.Installation{
				OwnerID:  model.NewID(),
				Version:  "version",
				DNS:      "dns.example.com",
				Size:     mmv1alpha1.Size100String,
				Affinity: model.InstallationAffinityMultiTenant,
				GroupID:  &groupID,
				State:    model.InstallationStateCreationRequested,
			}
		}

		t.Run("cluster with matching annotations exists", func(t *testing.T) {
			logger := testlib.MakeLogger(t)
			sqlStore := store.MakeTestSQLStore(t, logger)
			defer store.CloseConnection(t, sqlStore)
			supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

			cluster := standardStableTestCluster()
			err := sqlStore.CreateCluster(cluster, annotations)
			require.NoError(t, err)

			installation := installationInCreationRequestedState()

			err = sqlStore.CreateInstallation(installation, annotations)
			require.NoError(t, err)

			supervisor.Supervise(installation)
			expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationInProgress)
			expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateCreationRequested)
			expectClusterInstallationsOnCluster(t, sqlStore, cluster, 1)
		})

		t.Run("cluster with matching annotations does not exists", func(t *testing.T) {
			logger := testlib.MakeLogger(t)
			sqlStore := store.MakeTestSQLStore(t, logger)
			defer store.CloseConnection(t, sqlStore)
			supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

			cluster := standardStableTestCluster()
			err := sqlStore.CreateCluster(cluster, nil)
			require.NoError(t, err)

			installation := installationInCreationRequestedState()

			err = sqlStore.CreateInstallation(installation, annotations)
			require.NoError(t, err)

			supervisor.Supervise(installation)
			expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationNoCompatibleClusters)
			expectClusterInstallations(t, sqlStore, installation, 0, "")
			expectClusterInstallationsOnCluster(t, sqlStore, cluster, 0)
		})

		t.Run("annotations filter ignored when installation without annotations", func(t *testing.T) {
			logger := testlib.MakeLogger(t)
			sqlStore := store.MakeTestSQLStore(t, logger)
			defer store.CloseConnection(t, sqlStore)
			supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, false)

			cluster := standardStableTestCluster()
			err := sqlStore.CreateCluster(cluster, annotations)
			require.NoError(t, err)

			installation := installationInCreationRequestedState()

			err = sqlStore.CreateInstallation(installation, nil)
			require.NoError(t, err)

			supervisor.Supervise(installation)
			expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationInProgress)
			expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateCreationRequested)
			expectClusterInstallationsOnCluster(t, sqlStore, cluster, 1)
		})
	})

	t.Run("force CR upgrade to v1Beta", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", false, false, standardSchedulingOptions, &utils.ResourceUtil{}, logger, cloudMetrics, true)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		installation := &model.Installation{
			Version:   "version",
			DNS:       "dns.example.com",
			Size:      mmv1alpha1.Size100String,
			Affinity:  model.InstallationAffinityMultiTenant,
			State:     model.InstallationStateUpdateRequested,
			CRVersion: model.V1alphaCRVersion,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		supervisor.Supervise(installation)

		updatedInstallation, err := sqlStore.GetInstallation(installation.ID, false, false)
		require.NoError(t, err)
		require.Equal(t, model.V1betaCRVersion, updatedInstallation.CRVersion)
	})
}
