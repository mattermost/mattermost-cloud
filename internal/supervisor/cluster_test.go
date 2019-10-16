package supervisor_test

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/require"
)

type mockClusterStore struct {
	Cluster                     *model.Cluster
	UnlockedClustersPendingWork []*model.Cluster
	Clusters                    []*model.Cluster

	UnlockChan         chan interface{}
	UpdateClusterCalls int
}

func (s *mockClusterStore) GetCluster(clusterID string) (*model.Cluster, error) {
	return s.Cluster, nil
}

func (s *mockClusterStore) GetUnlockedClustersPendingWork() ([]*model.Cluster, error) {
	return s.UnlockedClustersPendingWork, nil
}

func (s *mockClusterStore) GetClusters(clusterFilter *model.ClusterFilter) ([]*model.Cluster, error) {
	return s.Clusters, nil
}

func (s *mockClusterStore) UpdateCluster(cluster *model.Cluster) error {
	s.UpdateClusterCalls++
	return nil
}

func (s *mockClusterStore) LockCluster(clusterID, lockerID string) (bool, error) {
	return true, nil
}

func (s *mockClusterStore) UnlockCluster(clusterID string, lockerID string, force bool) (bool, error) {
	if s.UnlockChan != nil {
		close(s.UnlockChan)
	}
	return true, nil
}

func (s *mockClusterStore) DeleteCluster(clusterID string) error {
	return nil
}

func (s *mockClusterStore) GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error) {
	return nil, nil
}

type mockClusterProvisioner struct{}

func (p *mockClusterProvisioner) PrepareCluster(cluster *model.Cluster) (bool, error) {
	return true, nil
}

func (p *mockClusterProvisioner) CreateCluster(cluster *model.Cluster, aws aws.AWS) error {
	return nil
}

func (p *mockClusterProvisioner) ProvisionCluster(cluster *model.Cluster) error {
	return nil
}

func (p *mockClusterProvisioner) UpgradeCluster(cluster *model.Cluster) error {
	return nil
}

func (p *mockClusterProvisioner) DeleteCluster(cluster *model.Cluster, aws aws.AWS) error {
	return nil
}

func TestClusterSupervisorDo(t *testing.T) {
	t.Run("no clusters pending work", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockClusterStore{}

		supervisor := supervisor.NewClusterSupervisor(mockStore, &mockClusterProvisioner{}, &mockAWS{}, "instanceID", logger)
		err := supervisor.Do()
		require.NoError(t, err)

		require.Equal(t, 0, mockStore.UpdateClusterCalls)
	})

	t.Run("mock cluster creation", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockClusterStore{}

		mockStore.UnlockedClustersPendingWork = []*model.Cluster{&model.Cluster{
			ID:    model.NewID(),
			State: model.ClusterStateCreationRequested,
		}}
		mockStore.Cluster = mockStore.UnlockedClustersPendingWork[0]
		mockStore.UnlockChan = make(chan interface{})

		supervisor := supervisor.NewClusterSupervisor(mockStore, &mockClusterProvisioner{}, &mockAWS{}, "instanceID", logger)
		err := supervisor.Do()
		require.NoError(t, err)

		<-mockStore.UnlockChan
		require.Equal(t, 2, mockStore.UpdateClusterCalls)
	})
}

func TestClusterSupervisorSupervise(t *testing.T) {
	testCases := []struct {
		Description   string
		InitialState  string
		ExpectedState string
	}{
		{"unexpected state", model.ClusterStateStable, model.ClusterStateStable},
		{"creation requested", model.ClusterStateCreationRequested, model.ClusterStateStable},
		{"provision requested", model.ClusterStateProvisioningRequested, model.ClusterStateStable},
		{"upgrade requested", model.ClusterStateUpgradeRequested, model.ClusterStateStable},
		{"deletion requested", model.ClusterStateDeletionRequested, model.ClusterStateDeleted},
	}

	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {
			logger := testlib.MakeLogger(t)
			sqlStore := store.MakeTestSQLStore(t, logger)
			supervisor := supervisor.NewClusterSupervisor(sqlStore, &mockClusterProvisioner{}, &mockAWS{}, "instanceID", logger)

			cluster := &model.Cluster{
				Provider: model.ProviderAWS,
				Size:     model.SizeAlef500,
				State:    tc.InitialState,
			}
			err := sqlStore.CreateCluster(cluster)
			require.NoError(t, err)

			supervisor.Supervise(cluster)

			cluster, err = sqlStore.GetCluster(cluster.ID)
			require.NoError(t, err)
			require.Equal(t, tc.ExpectedState, cluster.State)
		})
	}
}
