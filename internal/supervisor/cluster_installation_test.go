// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor_test

import (
	"fmt"
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/internal/testutil"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/require"
)

type mockClusterInstallationStore struct {
	Cluster                                 *model.Cluster
	Installation                            *model.Installation
	ClusterInstallation                     *model.ClusterInstallation
	UnlockedClusterInstallationsPendingWork []*model.ClusterInstallation
	Clusters                                []*model.Cluster
	Installations                           []*model.Installation
	ClusterInstallations                    []*model.ClusterInstallation

	UnlockChan                     chan interface{}
	UpdateClusterInstallationCalls int
}

func (s *mockClusterInstallationStore) GetCluster(clusterID string) (*model.Cluster, error) {
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
func (s *mockClusterInstallationStore) GetInstallation(installationID string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.Installation, error) {
	if s.Installation != nil {
		return s.Installation, nil
	}
	for _, installation := range s.Installations {
		if installation.ID == installationID {
			return installation, nil
		}
	}
	return nil, nil
}
func (s *mockClusterInstallationStore) GetClusterInstallation(clusterInstallationID string) (*model.ClusterInstallation, error) {
	if s.ClusterInstallation != nil {
		return s.ClusterInstallation, nil
	}
	for _, clusterInstallation := range s.ClusterInstallations {
		if clusterInstallation.ID == clusterInstallationID {
			return clusterInstallation, nil
		}
	}
	return nil, nil
}
func (s *mockClusterInstallationStore) GetUnlockedClusterInstallationsPendingWork() ([]*model.ClusterInstallation, error) {
	clusterInstallations := make([]*model.ClusterInstallation, len(s.UnlockedClusterInstallationsPendingWork))
	copy(clusterInstallations, s.UnlockedClusterInstallationsPendingWork)
	return clusterInstallations, nil
}
func (s *mockClusterInstallationStore) LockClusterInstallations(clusterInstallationID []string, lockerID string) (bool, error) {
	return true, nil
}
func (s *mockClusterInstallationStore) UnlockClusterInstallations(clusterInstallationID []string, lockerID string, force bool) (bool, error) {
	if s.UnlockChan != nil {
		close(s.UnlockChan)
	}
	return true, nil
}
func (s *mockClusterInstallationStore) UpdateClusterInstallation(clusterInstallation *model.ClusterInstallation) error {
	s.UpdateClusterInstallationCalls++
	return nil
}
func (s *mockClusterInstallationStore) DeleteClusterInstallation(clusterInstallationID string) error {
	return nil
}

func (s *mockClusterInstallationStore) GetInstallationBackups(filter *model.InstallationBackupFilter) ([]*model.InstallationBackup, error) {
	return nil, nil
}

func (s *mockClusterInstallationStore) GetMultitenantDatabases(filter *model.MultitenantDatabaseFilter) ([]*model.MultitenantDatabase, error) {
	return nil, nil
}

func (s *mockClusterInstallationStore) GetLogicalDatabases(filter *model.LogicalDatabaseFilter) ([]*model.LogicalDatabase, error) {
	return nil, nil
}

func (s *mockClusterInstallationStore) GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error) {
	return nil, nil
}

func (s *mockClusterInstallationStore) GetStateChangeEvents(filter *model.StateChangeEventFilter) ([]*model.StateChangeEventData, error) {
	return nil, nil
}

func (s *mockClusterInstallationStore) GetDNSRecordsForInstallation(installationID string) ([]*model.InstallationDNS, error) {
	installation, _ := s.GetInstallation(installationID, false, false)
	if installation == nil {
		return nil, nil
	}
	return []*model.InstallationDNS{
		{ID: "abcd", DomainName: "dns.example.com", InstallationID: installation.ID},
	}, nil
}

func TestClusterInstallationSupervisorDo(t *testing.T) {
	t.Run("no clusters pending work", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockClusterInstallationStore{}

		supervisor := supervisor.NewClusterInstallationSupervisor(
			mockStore,
			&mockInstallationProvisioner{},
			&mockAWS{},
			&mockEventProducer{},
			"instanceID",
			logger,
			cloudMetrics,
		)
		err := supervisor.Do()
		require.NoError(t, err)

		require.Equal(t, 0, mockStore.UpdateClusterInstallationCalls)
	})

	t.Run("mock cluster creation", func(t *testing.T) {
		logger := testlib.MakeLogger(t)

		cluster := &model.Cluster{ID: model.NewID()}
		installation := &model.Installation{ID: model.NewID()}
		mockStore := &mockClusterInstallationStore{
			Cluster:      cluster,
			Installation: installation,
			UnlockedClusterInstallationsPendingWork: []*model.ClusterInstallation{{
				ID:             model.NewID(),
				ClusterID:      cluster.ID,
				InstallationID: installation.ID,
				State:          model.ClusterInstallationStateCreationRequested,
			}},
			UnlockChan: make(chan interface{}),
		}
		mockStore.ClusterInstallation = mockStore.UnlockedClusterInstallationsPendingWork[0]

		supervisor := supervisor.NewClusterInstallationSupervisor(
			mockStore,
			&mockInstallationProvisioner{},
			&mockAWS{},
			&mockEventProducer{},
			"instanceID",
			logger,
			cloudMetrics,
		)
		err := supervisor.Do()
		require.NoError(t, err)

		<-mockStore.UnlockChan
		require.Equal(t, 2, mockStore.UpdateClusterInstallationCalls)
	})

	t.Run("order of pending works", func(t *testing.T) {
		logger := testlib.MakeLogger(t)

		var clusters []*model.Cluster
		var installations []*model.Installation
		var clusterInstallations []*model.ClusterInstallation

		clusterInstallationStates := []string{
			model.ClusterInstallationStateReady,             // should be 3rd
			model.ClusterInstallationStateReconciling,       // Should be 2nd
			model.ClusterInstallationStateDeletionRequested, // Should be 4th
			model.ClusterInstallationStateCreationRequested, // Should be 1st
		}

		for i := 1; i <= 4; i++ {
			clusters = append(clusters, &model.Cluster{ID: fmt.Sprintf("%d", i)})
			installations = append(installations, &model.Installation{ID: fmt.Sprintf("%d", i)})
			clusterInstallations = append(clusterInstallations, &model.ClusterInstallation{
				ID:             fmt.Sprintf("%d", i),
				ClusterID:      fmt.Sprintf("%d", i),
				InstallationID: fmt.Sprintf("%d", i),
				State:          clusterInstallationStates[i-1],
			})
		}

		mockStore := &mockClusterInstallationStore{
			UnlockedClusterInstallationsPendingWork: clusterInstallations,
			Clusters:                                clusters,
			Installations:                           installations,
			ClusterInstallations:                    clusterInstallations,
		}

		preferredClusterInstallationOrder := []string{"4", "2", "1", "3"}

		mockEventProducer := &mockEventProducer{}
		supervisor := supervisor.NewClusterInstallationSupervisor(
			mockStore,
			&mockInstallationProvisioner{},
			&mockAWS{},
			mockEventProducer,
			"instanceID",
			logger,
			cloudMetrics,
		)
		err := supervisor.Do()
		require.NoError(t, err)

		clusterInstallationListByWorkOrder := mockEventProducer.clusterInstallationListByEventOrder
		require.Equal(t, preferredClusterInstallationOrder, clusterInstallationListByWorkOrder)
	})

}

func TestClusterInstallationSupervisorSupervise(t *testing.T) {
	expectClusterInstallationState := func(t *testing.T, sqlStore *store.SQLStore, clusterInstallation *model.ClusterInstallation, expectedState string) {
		t.Helper()

		clusterInstallation, err := sqlStore.GetClusterInstallation(clusterInstallation.ID)
		require.NoError(t, err)
		require.Equal(t, expectedState, clusterInstallation.State)
	}

	t.Run("missing cluster", func(t *testing.T) {
		testCases := []struct {
			Description   string
			InitialState  string
			ExpectedState string
		}{
			{"on create", model.ClusterInstallationStateCreationRequested, model.ClusterInstallationStateCreationFailed},
			{"on delete", model.ClusterInstallationStateDeletionRequested, model.ClusterInstallationStateDeletionFailed},
		}

		for _, tc := range testCases {
			t.Run(tc.Description, func(t *testing.T) {
				logger := testlib.MakeLogger(t)
				sqlStore := store.MakeTestSQLStore(t, logger)
				defer store.CloseConnection(t, sqlStore)
				supervisor := supervisor.NewClusterInstallationSupervisor(
					sqlStore,
					&mockInstallationProvisioner{},
					&mockAWS{},
					testutil.SetupTestEventsProducer(sqlStore, logger),
					"instanceID",
					logger,
					cloudMetrics,
				)

				installation := &model.Installation{}
				err := sqlStore.CreateInstallation(installation, nil, testutil.DNSForInstallation("dns.examole.com"))
				require.NoError(t, err)

				clusterInstallation := &model.ClusterInstallation{
					ClusterID:      model.NewID(),
					InstallationID: installation.ID,
					Namespace:      "namespace",
					State:          tc.InitialState,
				}
				err = sqlStore.CreateClusterInstallation(clusterInstallation)
				require.NoError(t, err)

				supervisor.Supervise(clusterInstallation)
				expectClusterInstallationState(t, sqlStore, clusterInstallation, tc.ExpectedState)
			})
		}
	})

	t.Run("missing installation", func(t *testing.T) {
		testCases := []struct {
			Description   string
			InitialState  string
			ExpectedState string
		}{
			{"on create", model.ClusterInstallationStateCreationRequested, model.ClusterInstallationStateCreationFailed},
			{"on delete", model.ClusterInstallationStateDeletionRequested, model.ClusterInstallationStateDeletionFailed},
		}

		for _, tc := range testCases {
			t.Run(tc.Description, func(t *testing.T) {
				logger := testlib.MakeLogger(t)
				sqlStore := store.MakeTestSQLStore(t, logger)
				defer store.CloseConnection(t, sqlStore)
				supervisor := supervisor.NewClusterInstallationSupervisor(
					sqlStore,
					&mockInstallationProvisioner{},
					&mockAWS{},
					testutil.SetupTestEventsProducer(sqlStore, logger),
					"instanceID",
					logger,
					cloudMetrics,
				)

				cluster := &model.Cluster{}
				err := sqlStore.CreateCluster(cluster, nil)
				require.NoError(t, err)

				clusterInstallation := &model.ClusterInstallation{
					ClusterID:      cluster.ID,
					InstallationID: model.NewID(),
					Namespace:      "namespace",
					State:          tc.InitialState,
				}
				err = sqlStore.CreateClusterInstallation(clusterInstallation)
				require.NoError(t, err)

				supervisor.Supervise(clusterInstallation)
				expectClusterInstallationState(t, sqlStore, clusterInstallation, tc.ExpectedState)
			})
		}
	})

	t.Run("cannot delete when backup is running", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)
		supervisor := supervisor.NewClusterInstallationSupervisor(
			sqlStore,
			&mockInstallationProvisioner{},
			&mockAWS{},
			testutil.SetupTestEventsProducer(sqlStore, logger),
			"instanceID",
			logger,
			cloudMetrics,
		)

		cluster := &model.Cluster{}
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		installation := &model.Installation{}
		err = sqlStore.CreateInstallation(installation, nil, testutil.DNSForInstallation("dns.example.com"))
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateDeletionRequested,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		backup := &model.InstallationBackup{
			ClusterInstallationID: clusterInstallation.ID,
			State:                 model.InstallationBackupStateBackupRequested,
		}
		err = sqlStore.CreateInstallationBackup(backup)
		require.NoError(t, err)

		supervisor.Supervise(clusterInstallation)
		expectClusterInstallationState(t, sqlStore, clusterInstallation, model.ClusterInstallationStateDeletionRequested)
	})

	t.Run("transition", func(t *testing.T) {
		testCases := []struct {
			Description   string
			InitialState  string
			ExpectedState string
		}{
			{"unexpected state", "invalid-state", "invalid-state"},
			{"creation-requested", model.ClusterInstallationStateCreationRequested, model.ClusterInstallationStateReconciling},
			{"reconciling", model.ClusterInstallationStateReconciling, model.ClusterInstallationStateStable},
			{"ready", model.ClusterInstallationStateReady, model.ClusterInstallationStateStable},
			{"deletion-requested", model.ClusterInstallationStateDeletionRequested, model.ClusterInstallationStateDeleted},
		}

		for _, tc := range testCases {
			t.Run(tc.Description, func(t *testing.T) {
				logger := testlib.MakeLogger(t)
				sqlStore := store.MakeTestSQLStore(t, logger)
				defer store.CloseConnection(t, sqlStore)
				supervisor := supervisor.NewClusterInstallationSupervisor(
					sqlStore,
					&mockInstallationProvisioner{},
					&mockAWS{},
					testutil.SetupTestEventsProducer(sqlStore, logger),
					"instanceID",
					logger,
					cloudMetrics,
				)

				cluster := &model.Cluster{}
				err := sqlStore.CreateCluster(cluster, nil)
				require.NoError(t, err)

				installation := &model.Installation{}
				err = sqlStore.CreateInstallation(installation, nil, testutil.DNSForInstallation("dns.example.com"))
				require.NoError(t, err)

				clusterInstallation := &model.ClusterInstallation{
					ClusterID:      cluster.ID,
					InstallationID: installation.ID,
					Namespace:      "namespace",
					State:          tc.InitialState,
				}
				err = sqlStore.CreateClusterInstallation(clusterInstallation)
				require.NoError(t, err)

				supervisor.Supervise(clusterInstallation)
				expectClusterInstallationState(t, sqlStore, clusterInstallation, tc.ExpectedState)
			})
		}
	})

	t.Run("state has changed since cluster installation was selected to be worked on", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)
		supervisor := supervisor.NewClusterInstallationSupervisor(
			sqlStore,
			&mockInstallationProvisioner{},
			&mockAWS{},
			testutil.SetupTestEventsProducer(sqlStore, logger),
			"instanceID",
			logger,
			cloudMetrics,
		)

		cluster := &model.Cluster{}
		err := sqlStore.CreateCluster(cluster, nil)
		require.NoError(t, err)

		installation := &model.Installation{}
		err = sqlStore.CreateInstallation(installation, nil, testutil.DNSForInstallation("dns.example.com"))
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
			Namespace:      "namespace",
			State:          model.ClusterInstallationStateReconciling,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		// The stored cluster installation is ClusterInstallationStateReconciling,
		// so we will pass in a cluster installation with state of
		// ClusterInstallationStateCreationRequested to simulate stale state.
		clusterInstallation.State = model.ClusterInstallationStateCreationRequested

		supervisor.Supervise(clusterInstallation)
		expectClusterInstallationState(t, sqlStore, clusterInstallation, model.ClusterInstallationStateReconciling)
	})
}
