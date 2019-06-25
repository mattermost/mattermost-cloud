package supervisor_test

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/model"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
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

func (s *mockInstallationStore) GetInstallation(installationID string) (*model.Installation, error) {
	return s.Installation, nil
}

func (s *mockInstallationStore) GetUnlockedInstallationsPendingWork() ([]*model.Installation, error) {
	return s.UnlockedInstallationsPendingWork, nil
}

func (s *mockInstallationStore) UpdateInstallation(installation *model.Installation) error {
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

type mockInstallationProvisioner struct{}

func (p *mockInstallationProvisioner) CreateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
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

type mockAWS struct{}

func (a *mockAWS) CreateCNAME(dnsName string, dnsEndpoints []string, logger log.FieldLogger) error {
	return nil
}

func (a *mockAWS) DeleteCNAME(dnsName string, logger log.FieldLogger) error {
	return nil
}

func (a *mockAWS) TagResource(resourceID, key, value string, logger log.FieldLogger) error {
	return nil
}

func (a *mockAWS) UntagResource(resourceID, key, value string, logger log.FieldLogger) error {
	return nil
}

func TestInstallationSupervisorDo(t *testing.T) {
	t.Run("no clusters pending work", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockInstallationStore{}

		supervisor := supervisor.NewInstallationSupervisor(mockStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", logger)
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

		supervisor := supervisor.NewInstallationSupervisor(mockStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", logger)
		err := supervisor.Do()
		require.NoError(t, err)

		<-mockStore.UnlockChan
		require.Equal(t, 1, mockStore.UpdateInstallationCalls)
	})
}

func TestInstallationSupervisor(t *testing.T) {
	expectInstallationState := func(t *testing.T, sqlStore *store.SQLStore, installation *model.Installation, expectedState string) {
		t.Helper()

		installation, err := sqlStore.GetInstallation(installation.ID)
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

	t.Run("unexpected state", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", logger)

		cluster := &model.Cluster{
			State: model.ClusterStateStable,
		}
		err := sqlStore.CreateCluster(cluster)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
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
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", logger)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
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

	t.Run("creation requested, cluster installations not yet created, no available clusters", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", logger)

		cluster := &model.Cluster{
			State: model.ClusterStateStable,
		}
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

	t.Run("creation requested, cluster installations not yet created, available cluster", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", logger)

		cluster := &model.Cluster{
			State: model.ClusterStateStable,
		}
		err := sqlStore.CreateCluster(cluster)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateCreationRequested,
		}

		err = sqlStore.CreateInstallation(installation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationRequested)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateCreationRequested)
	})

	t.Run("creation requested, cluster installations failed", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", logger)

		cluster := &model.Cluster{
			State: model.ClusterStateStable,
		}
		err := sqlStore.CreateCluster(cluster)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
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
			State:          model.ClusterInstallationStateCreationFailed,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationFailed)
		expectClusterInstallations(t, sqlStore, installation, 1, model.InstallationStateCreationFailed)
	})

	t.Run("creation requested, cluster installations stable", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", logger)

		cluster := &model.Cluster{
			State: model.ClusterStateStable,
		}
		err := sqlStore.CreateCluster(cluster)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
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
		expectClusterInstallations(t, sqlStore, installation, 1, model.InstallationStateStable)
	})

	t.Run("creation dns requested, cluster installations stable", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", logger)

		cluster := &model.Cluster{
			State: model.ClusterStateStable,
		}
		err := sqlStore.CreateCluster(cluster)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
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
		expectClusterInstallations(t, sqlStore, installation, 1, model.InstallationStateStable)
	})

	t.Run("no compatible clusters, cluster installations not yet created, no clusters", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", logger)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
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
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", logger)

		cluster := &model.Cluster{
			State: model.ClusterStateStable,
		}
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
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", logger)

		cluster := &model.Cluster{
			State: model.ClusterStateStable,
		}
		err := sqlStore.CreateCluster(cluster)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateCreationNoCompatibleClusters,
		}

		err = sqlStore.CreateInstallation(installation)
		require.NoError(t, err)

		supervisor.Supervise(installation)
		expectInstallationState(t, sqlStore, installation, model.InstallationStateCreationRequested)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateCreationRequested)
	})

	t.Run("upgrade requested, cluster installations stable", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", logger)

		cluster := &model.Cluster{
			State: model.ClusterStateStable,
		}
		err := sqlStore.CreateCluster(cluster)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateUpgradeRequested,
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
		expectInstallationState(t, sqlStore, installation, model.InstallationStateUpgradeInProgress)
		expectClusterInstallations(t, sqlStore, installation, 1, model.ClusterInstallationStateReconciling)
	})

	t.Run("upgrade in progress, cluster installations stable", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", logger)

		cluster := &model.Cluster{
			State: model.ClusterStateStable,
		}
		err := sqlStore.CreateCluster(cluster)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &groupID,
			State:    model.InstallationStateUpgradeInProgress,
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
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", logger)

		cluster := &model.Cluster{
			State: model.ClusterStateStable,
		}
		err := sqlStore.CreateCluster(cluster)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
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
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", logger)

		cluster := &model.Cluster{
			State: model.ClusterStateStable,
		}
		err := sqlStore.CreateCluster(cluster)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
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
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", logger)

		cluster := &model.Cluster{
			State: model.ClusterStateStable,
		}
		err := sqlStore.CreateCluster(cluster)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
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
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", logger)

		cluster := &model.Cluster{
			State: model.ClusterStateStable,
		}
		err := sqlStore.CreateCluster(cluster)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
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
		supervisor := supervisor.NewInstallationSupervisor(sqlStore, &mockInstallationProvisioner{}, &mockAWS{}, "instanceID", logger)

		cluster := &model.Cluster{
			State: model.ClusterStateStable,
		}
		err := sqlStore.CreateCluster(cluster)
		require.NoError(t, err)

		owner := model.NewID()
		groupID := model.NewID()
		installation := &model.Installation{
			OwnerID:  owner,
			Version:  "version",
			DNS:      "dns.example.com",
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
}
