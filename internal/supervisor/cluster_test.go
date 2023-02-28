// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor_test

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/internal/testutil"
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
	if s.Cluster != nil {
		return s.Cluster, nil
	}
	for _, cluster := range s.Clusters {
		if cluster.ID == clusterID {
			return cluster, nil
		}
	}
	return nil, nil
}

func (s *mockClusterStore) GetUnlockedClustersPendingWork() ([]*model.Cluster, error) {
	clusters := make([]*model.Cluster, len(s.UnlockedClustersPendingWork))
	copy(clusters, s.UnlockedClustersPendingWork)
	return clusters, nil
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

func (s *mockClusterStore) GetStateChangeEvents(filter *model.StateChangeEventFilter) ([]*model.StateChangeEventData, error) {
	return nil, nil
}

type mockClusterProvisionerOption struct {
	mock *mockClusterProvisioner
}

func (p *mockClusterProvisionerOption) GetClusterProvisioner(provisioner string) supervisor.ClusterProvisioner {
	if p.mock == nil {
		p.mock = &mockClusterProvisioner{}
	}
	return p.mock
}

type mockClusterProvisioner struct{}

func (p *mockClusterProvisioner) CreateNodes(cluster *model.Cluster, aws aws.AWS) error {
	return nil
}

func (p *mockClusterProvisioner) RefreshClusterMetadata(cluster *model.Cluster) error {
	return nil
}

func (p *mockClusterProvisioner) PrepareCluster(cluster *model.Cluster) bool {
	return true
}

func (p *mockClusterProvisioner) CreateCluster(cluster *model.Cluster, aws aws.AWS) error {
	return nil
}

func (p *mockClusterProvisioner) CheckClusterCreated(cluster *model.Cluster, awsClient aws.AWS) (bool, error) {
	return true, nil
}

func (p *mockClusterProvisioner) CheckNodesCreated(cluster *model.Cluster, awsClient aws.AWS) (bool, error) {
	return true, nil
}

func (p *mockClusterProvisioner) ProvisionCluster(cluster *model.Cluster, aws aws.AWS) error {
	return nil
}

func (p *mockClusterProvisioner) UpgradeCluster(cluster *model.Cluster, aws aws.AWS) error {
	return nil
}

func (p *mockClusterProvisioner) ResizeCluster(cluster *model.Cluster, aws aws.AWS) error {
	return nil
}

func (p *mockClusterProvisioner) DeleteCluster(cluster *model.Cluster, aws aws.AWS) (bool, error) {
	return true, nil
}

func (p *mockClusterProvisioner) RefreshKopsMetadata(cluster *model.Cluster) error {
	return nil
}

func TestClusterSupervisorDo(t *testing.T) {
	t.Run("no clusters pending work", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockClusterStore{}

		supervisor := supervisor.NewClusterSupervisor(
			mockStore,
			&mockClusterProvisionerOption{},
			&mockAWS{},
			&mockEventProducer{},
			"instanceID",
			logger,
			cloudMetrics,
		)
		err := supervisor.Do()
		require.NoError(t, err)

		require.Equal(t, 0, mockStore.UpdateClusterCalls)
	})

	t.Run("mock cluster creation", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockClusterStore{}

		mockStore.UnlockedClustersPendingWork = []*model.Cluster{{
			ID:    model.NewID(),
			State: model.ClusterStateCreationRequested,
		}}
		mockStore.Cluster = mockStore.UnlockedClustersPendingWork[0]
		mockStore.UnlockChan = make(chan interface{})

		supervisor := supervisor.NewClusterSupervisor(
			mockStore,
			&mockClusterProvisionerOption{},
			&mockAWS{},
			&mockEventProducer{},
			"instanceID",
			logger,
			cloudMetrics,
		)
		err := supervisor.Do()
		require.NoError(t, err)

		<-mockStore.UnlockChan
		require.Equal(t, 2, mockStore.UpdateClusterCalls)
	})

	t.Run("order of pending works", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockClusterStore{}

		clusters := []*model.Cluster{
			{
				ID:    "1",
				State: model.ClusterStateDeletionRequested, // should be 4th
			},
			{
				ID:    "2",
				State: model.ClusterStateCreationRequested, // should be 1st
			},
			{
				ID:    "3",
				State: model.ClusterStateResizeRequested, // should be 5th
			},
			{
				ID:    "4",
				State: model.ClusterStateCreationInProgress, // should be 3rd
			},
			{
				ID:    "5",
				State: model.ClusterStateProvisioningRequested, // should be 6th
			},
			{
				ID:    "6",
				State: model.ClusterStateCreationRequested, // should be 2nd
			},
		}
		preferredClusterOrder := []string{"2", "6", "4", "1", "3", "5"}

		mockStore.UnlockedClustersPendingWork = clusters
		mockStore.Clusters = clusters

		mockEventProducer := &mockEventProducer{}
		supervisor := supervisor.NewClusterSupervisor(
			mockStore,
			&mockClusterProvisionerOption{},
			&mockAWS{},
			mockEventProducer,
			"instanceID",
			logger,
			cloudMetrics,
		)
		err := supervisor.Do()
		require.NoError(t, err)

		clusterListByWorkOrder := mockEventProducer.clusterListByEventOrder
		require.Equal(t, preferredClusterOrder, clusterListByWorkOrder)
	})

}

func TestClusterSupervisorSupervise(t *testing.T) {
	testCases := []struct {
		Description   string
		InitialState  string
		ExpectedState string
	}{
		{"unexpected state", model.ClusterStateStable, model.ClusterStateStable},
		{"creation requested", model.ClusterStateCreationRequested, model.ClusterStateProvisionInProgress},
		{"provision requested", model.ClusterStateProvisioningRequested, model.ClusterStateStable},
		{"upgrade requested", model.ClusterStateUpgradeRequested, model.ClusterStateStable},
		{"resize requested", model.ClusterStateResizeRequested, model.ClusterStateStable},
		{"deletion requested", model.ClusterStateDeletionRequested, model.ClusterStateDeleted},
		{"refresh metadata", model.ClusterStateRefreshMetadata, model.ClusterStateStable},
	}

	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {
			logger := testlib.MakeLogger(t)
			sqlStore := store.MakeTestSQLStore(t, logger)
			defer store.CloseConnection(t, sqlStore)
			supervisor := supervisor.NewClusterSupervisor(
				sqlStore,
				&mockClusterProvisionerOption{},
				&mockAWS{},
				testutil.SetupTestEventsProducer(sqlStore, logger),
				"instanceID",
				logger,
				cloudMetrics,
			)

			cluster := &model.Cluster{
				Provider:                model.ProviderAWS,
				ProvisionerMetadataKops: &model.KopsMetadata{},
				State:                   tc.InitialState,
			}
			err := sqlStore.CreateCluster(cluster, nil)
			require.NoError(t, err)

			supervisor.Supervise(cluster)

			cluster, err = sqlStore.GetCluster(cluster.ID)
			require.NoError(t, err)
			require.Equal(t, tc.ExpectedState, cluster.State)
		})
	}

	t.Run("state has changed since cluster was selected to be worked on", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)
		supervisor := supervisor.NewClusterSupervisor(
			sqlStore,
			&mockClusterProvisionerOption{},
			&mockAWS{},
			testutil.SetupTestEventsProducer(sqlStore, logger),
			"instanceID",
			logger,
			cloudMetrics,
		)

		cluster := &model.Cluster{
			Provider: model.ProviderAWS,
			State:    model.ClusterStateDeletionRequested,
		}
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		// The stored cluster is ClusterStateDeletionRequested, so we will pass
		// in a cluster with state of ClusterStateCreationRequested to simulate
		// stale state.
		cluster.State = model.ClusterStateCreationRequested

		supervisor.Supervise(cluster)

		cluster, err = sqlStore.GetCluster(cluster.ID)
		require.NoError(t, err)
		require.Equal(t, model.ClusterStateDeletionRequested, cluster.State)
	})
}
