package supervisor_test

import (
	"context"
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/model"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
)

type mockClusterProvisioner struct {
}

func (p *mockClusterProvisioner) CreateCluster(cluster *model.Cluster) error {
	return nil
}

func (p *mockClusterProvisioner) UpgradeCluster(cluster *model.Cluster) error {
	return nil
}

func (p *mockClusterProvisioner) DeleteCluster(cluster *model.Cluster) error {
	return nil
}

func TestClusterSupervisor(t *testing.T) {
	t.Run("no available workers", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		workers := semaphore.NewWeighted(0)
		supervisor := supervisor.NewClusterSupervisor(sqlStore, &mockClusterProvisioner{}, workers, logger)
		err := supervisor.Do()
		require.NoError(t, err)
	})

	t.Run("no clusters pending work", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		workers := semaphore.NewWeighted(1)
		supervisor := supervisor.NewClusterSupervisor(sqlStore, &mockClusterProvisioner{}, workers, logger)
		err := supervisor.Do()
		require.NoError(t, err)
	})

	expectClusterState := func(t *testing.T, sqlStore *store.SQLStore, cluster *model.Cluster, expectedState string) {
		t.Helper()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		done := make(chan struct{})
		for {
			cluster, err := sqlStore.GetCluster(cluster.ID)
			require.NoError(t, err)
			if cluster.State == expectedState {
				close(done)
				return
			}

			select {
			case <-ctx.Done():
				t.Fatalf("cluster did not transition to %s within 5 seconds", expectedState)
				return
			default:
			}
		}
	}

	t.Run("transition", func(t *testing.T) {
		testCases := []struct {
			Description   string
			InitialState  string
			ExpectedState string
		}{
			{"creation requested", model.ClusterStateCreationRequested, model.ClusterStateStable},
			{"upgrade requested", model.ClusterStateUpgradeRequested, model.ClusterStateStable},
			{"deletion requested", model.ClusterStateDeletionRequested, model.ClusterStateDeleted},
		}

		for _, tc := range testCases {
			t.Run(tc.Description, func(t *testing.T) {
				logger := testlib.MakeLogger(t)
				sqlStore := store.MakeTestSQLStore(t, logger)
				workers := semaphore.NewWeighted(1)
				supervisor := supervisor.NewClusterSupervisor(sqlStore, &mockClusterProvisioner{}, workers, logger)

				cluster := &model.Cluster{
					Provider: model.ProviderAWS,
					Size:     model.SizeAlef500,
					State:    tc.InitialState,
				}
				err := sqlStore.CreateCluster(cluster)
				require.NoError(t, err)

				err = supervisor.Do()
				require.NoError(t, err)

				expectClusterState(t, sqlStore, cluster, tc.ExpectedState)
			})
		}
	})
}
