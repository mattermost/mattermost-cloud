package supervisor_test

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/k8s"
	"github.com/mattermost/mattermost-cloud/internal/tools/utils"
	"github.com/mattermost/mattermost-cloud/model"
	mmv1alpha1 "github.com/mattermost/mattermost-operator/pkg/apis/mattermost/v1alpha1"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type mockInstallationStore struct {
	Installation                     *model.Installation
	UnlockedInstallationsPendingWork []*model.Installation

	UnlockChan              chan interface{}
	UpdateInstallationCalls int
}

func (s *mockInstallationStore) GetClusters(clusterFilter *model.ClusterFilter) ([]*model.Cluster, error) {
	return nil, nil
}

func (s *mockInstallationStore) GetCluster(id string) (*model.Cluster, error) {
	return nil, nil
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
	return nil, nil
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

func (s *mockInstallationStore) GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error) {
	return nil, nil
}

type mockInstallationProvisioner struct {
	UseCustomClusterResources bool
	CustomClusterResources    *k8s.ClusterResources
}

func (p *mockInstallationProvisioner) CreateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation, awsClient aws.AWS) error {
	return nil
}

func (p *mockInstallationProvisioner) UpdateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	return nil
}

func (p *mockInstallationProvisioner) DeleteClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	return nil
}

func (p *mockInstallationProvisioner) GetClusterInstallationResource(cluster *model.Cluster, installation *model.Installation, clusterIntallation *model.ClusterInstallation) (*mmv1alpha1.ClusterInstallation, error) {
	return &mmv1alpha1.ClusterInstallation{
			Spec: mmv1alpha1.ClusterInstallationSpec{},
			Status: mmv1alpha1.ClusterInstallationStatus{
				State:    mmv1alpha1.Stable,
				Endpoint: "example-dns.mattermost.cloud",
			},
		},
		nil
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

// TODO(gsagula): this can be replaced with /internal/mocks/aws-tools/AWS.go so that inputs and other variants
// can be tested.
type mockAWS struct{}

func (a *mockAWS) GetCertificateSummaryByTag(key, value string, logger log.FieldLogger) (*acm.CertificateSummary, error) {
	return nil, nil
}

func (a *mockAWS) GetAndClaimVpcResources(clusterID, owner string, logger log.FieldLogger) (aws.ClusterResources, error) {
	return aws.ClusterResources{}, nil
}

func (a *mockAWS) ReleaseVpc(clusterID string, logger log.FieldLogger) error {
	return nil
}

func (a *mockAWS) GetPrivateZoneDomainName(logger log.FieldLogger) (string, error) {
	return "test.domain", nil
}

func (a *mockAWS) CreatePrivateCNAME(dnsName string, dnsEndpoints []string, logger log.FieldLogger) error {
	return nil
}

func (a *mockAWS) CreatePublicCNAME(dnsName string, dnsEndpoints []string, logger log.FieldLogger) error {
	return nil
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

func TestInstallationSupervisorDo(t *testing.T) {
	t.Run("no clusters pending work", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockInstallationStore{}

		supervisor := supervisor.NewInstallationSupervisor(mockStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)
		err := supervisor.Do()
		require.NoError(t, err)

		require.Equal(t, 0, mockStore.UpdateInstallationCalls)
	})

	t.Run("mock cluster creation", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockInstallationStore{}

		mockStore.UnlockedInstallationsPendingWork = []*model.Installation{&model.Installation{
			ID:    model.NewID(),
			State: model.InstallationStateDeletionRequested,
		}}
		mockStore.Installation = mockStore.UnlockedInstallationsPendingWork[0]
		mockStore.UnlockChan = make(chan interface{})

		supervisor := supervisor.NewInstallationSupervisor(mockStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)
		err := supervisor.Do()
		require.NoError(t, err)

		<-mockStore.UnlockChan
		require.Equal(t, 1, mockStore.UpdateInstallationCalls)
	})
}

func TestInstallationSupervisor(t *testing.T) {
	expectInstallationState := func(t *testing.T, sqlStore *store.SQLStore, installation *model.Installation, expectedState string) {
		t.Helper()

		installation, err := sqlStore.GetInstallation(installation.ID, false, false)
		require.NoError(t, err)
		require.Equal(t, expectedState, installation.State)
	}

	expectClusterInstallations := func(t *testing.T, sqlStore *store.SQLStore, installation *model.Installation, expectedCount int, state string) {
		t.Helper()
		clusterInstallations, err := sqlStore.GetClusterInstallations(&model.ClusterInstallationFilter{
			PerPage:        model.AllPerPage,
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
			PerPage:   model.AllPerPage,
			ClusterID: cluster.ID,
		})
		require.NoError(t, err)
		require.Len(t, clusterInstallations, expectedCount)
	}

	standardStableTestCluster := func() *model.Cluster {
		return &model.Cluster{
			State:              model.ClusterStateStable,
			AllowInstallations: true,
		}
	}

	t.Run("unexpected state", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster)
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

		err = sqlStore.CreateInstallation(installation)
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

	t.Run("creation requested, cluster installations not yet created, no clusters", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

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

		err := sqlStore.CreateInstallation(installation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationNoCompatibleClusters)
		expectClusterInstallations(t, sqlStore, installation, 0, "")
	})

	t.Run("creation requested, cluster installations not yet created, cluster doesn't allow scheduling", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

		cluster := standardStableTestCluster()
		cluster.AllowInstallations = false
		err := sqlStore.CreateCluster(cluster)
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

		err = sqlStore.CreateInstallation(installation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationNoCompatibleClusters)
		expectClusterInstallations(t, sqlStore, installation, 0, "")
	})

	t.Run("creation requested, cluster installations not yet created, no empty clusters", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster)
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

		err = sqlStore.CreateInstallation(installation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationNoCompatibleClusters)
		expectClusterInstallations(t, sqlStore, installation, 0, "")
	})

	t.Run("creation requested, cluster installations reconciling", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster)
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

		err = sqlStore.CreateInstallation(installation)
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
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster)
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

		err = sqlStore.CreateInstallation(installation)
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
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster)
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

		err = sqlStore.CreateInstallation(installation)
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

	t.Run("pre provisioning requested, cluster installations reconciling", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster)
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

		err = sqlStore.CreateInstallation(installation)
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
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster)
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

		err = sqlStore.CreateInstallation(installation)
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
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster)
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

		err = sqlStore.CreateInstallation(installation)
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
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster)
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

		err = sqlStore.CreateInstallation(installation)
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

	t.Run("creation in progress, cluster installations failed", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster)
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

		err = sqlStore.CreateInstallation(installation)
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

	t.Run("creation dns requested, cluster installations stable", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster)
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

		err = sqlStore.CreateInstallation(installation)
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
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

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

		err := sqlStore.CreateInstallation(installation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationNoCompatibleClusters)
		expectClusterInstallations(t, sqlStore, installation, 0, "")
	})

	t.Run("no compatible clusters, cluster installations not yet created, no available clusters", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster)
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

		err = sqlStore.CreateInstallation(installation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationNoCompatibleClusters)
		expectClusterInstallations(t, sqlStore, installation, 0, "")
	})

	t.Run("no compatible clusters, cluster installations not yet created, available cluster", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster)
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

		err = sqlStore.CreateInstallation(installation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationInProgress)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateCreationRequested)
	})

	t.Run("upgrade requested, cluster installations stable", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster)
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

		err = sqlStore.CreateInstallation(installation)
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

	t.Run("upgrade in progress, cluster installations stable", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster)
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

		err = sqlStore.CreateInstallation(installation)
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

	t.Run("deletion requested, cluster installations stable", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster)
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

		err = sqlStore.CreateInstallation(installation)
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
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster)
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

		err = sqlStore.CreateInstallation(installation)
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
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster)
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

		err = sqlStore.CreateInstallation(installation)
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
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster)
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

		err = sqlStore.CreateInstallation(installation)
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

	t.Run("creation requested, cluster installations deleted", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

		cluster := standardStableTestCluster()
		err := sqlStore.CreateCluster(cluster)
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

		err = sqlStore.CreateInstallation(installation)
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
			supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

			cluster := standardStableTestCluster()
			err := sqlStore.CreateCluster(cluster)
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

			err = sqlStore.CreateInstallation(installation)
			require.NoError(t, err)

			supervisor.Supervise(installation)
			expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationInProgress)
			expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateCreationRequested)
			expectClusterInstallationsOnCluster(t, sqlStore, cluster, 1)
		})

		t.Run("creation requested, cluster installations not yet created, 3 installations, available cluster", func(t *testing.T) {
			logger := testlib.MakeLogger(t)
			sqlStore := store.MakeTestSQLStore(t, logger)
			supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

			cluster := standardStableTestCluster()
			err := sqlStore.CreateCluster(cluster)
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

					err = sqlStore.CreateInstallation(installation)
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
			supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

			cluster := standardStableTestCluster()
			err := sqlStore.CreateCluster(cluster)
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

			err = sqlStore.CreateInstallation(isolatedInstallation)
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

			err = sqlStore.CreateInstallation(multitenantInstallation)
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
			supervisor := supervisor.NewInstallationSupervisor(sqlStore, mockInstallationProvisioner, &mockAWS{}, "instanceID", 80, false, false, &utils.ResourceUtil{}, logger)

			cluster := standardStableTestCluster()
			err := sqlStore.CreateCluster(cluster)
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

			err = sqlStore.CreateInstallation(installation)
			require.NoError(t, err)

			supervisor.Supervise(installation)
			expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationNoCompatibleClusters)
			expectClusterInstallations(t, sqlStore, installation, 0, "")
			expectClusterInstallationsOnCluster(t, sqlStore, cluster, 0)
		})
	})
}
