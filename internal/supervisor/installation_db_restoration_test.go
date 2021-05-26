// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor_test

import (
	"fmt"
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/provisioner"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRestorationStore struct {
	InstallationRestorationOperation *model.InstallationDBRestorationOperation
	RestorationPending               []*model.InstallationDBRestorationOperation
	Installation                     *model.Installation
	UnlockChan                       chan interface{}

	UpdateRestorationOperationCalls int
}

func (m *mockRestorationStore) GetUnlockedInstallationDBRestorationOperationsPendingWork() ([]*model.InstallationDBRestorationOperation, error) {
	return m.RestorationPending, nil
}

func (m *mockRestorationStore) GetInstallationDBRestorationOperation(id string) (*model.InstallationDBRestorationOperation, error) {
	return m.InstallationRestorationOperation, nil
}

func (m *mockRestorationStore) UpdateInstallationDBRestorationOperationState(dbRestoration *model.InstallationDBRestorationOperation) error {
	m.UpdateRestorationOperationCalls++
	return nil
}

func (m *mockRestorationStore) UpdateInstallationDBRestorationOperation(dbRestoration *model.InstallationDBRestorationOperation) error {
	m.UpdateRestorationOperationCalls++
	return nil
}

func (m *mockRestorationStore) DeleteInstallationDBRestorationOperation(id string) error {
	return nil
}

func (m *mockRestorationStore) LockInstallationDBRestorationOperations(id []string, lockerID string) (bool, error) {
	return true, nil
}

func (m *mockRestorationStore) UnlockInstallationDBRestorationOperations(id []string, lockerID string, force bool) (bool, error) {
	if m.UnlockChan != nil {
		close(m.UnlockChan)
	}
	return true, nil
}

func (m *mockRestorationStore) GetInstallationBackup(id string) (*model.InstallationBackup, error) {
	return &model.InstallationBackup{ID: id}, nil
}

func (m *mockRestorationStore) LockInstallationBackups(backupsID []string, lockerID string) (bool, error) {
	panic("implement me")
}

func (m *mockRestorationStore) UnlockInstallationBackups(backupsID []string, lockerID string, force bool) (bool, error) {
	panic("implement me")
}

func (m *mockRestorationStore) GetInstallation(installationID string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.Installation, error) {
	return m.Installation, nil
}

func (m *mockRestorationStore) UpdateInstallation(installation *model.Installation) error {
	panic("implement me")
}

func (m *mockRestorationStore) LockInstallation(installationID, lockerID string) (bool, error) {
	return true, nil
}

func (m *mockRestorationStore) UnlockInstallation(installationID, lockerID string, force bool) (bool, error) {
	return true, nil
}

func (m *mockRestorationStore) GetClusterInstallations(filter *model.ClusterInstallationFilter) ([]*model.ClusterInstallation, error) {
	return []*model.ClusterInstallation{{ID: "id"}}, nil
}

func (m *mockRestorationStore) GetClusterInstallation(clusterInstallationID string) (*model.ClusterInstallation, error) {
	return &model.ClusterInstallation{ID: "id"}, nil
}

func (m *mockRestorationStore) LockClusterInstallations(clusterInstallationID []string, lockerID string) (bool, error) {
	return true, nil
}

func (m *mockRestorationStore) UnlockClusterInstallations(clusterInstallationID []string, lockerID string, force bool) (bool, error) {
	return true, nil
}

func (m *mockRestorationStore) GetCluster(id string) (*model.Cluster, error) {
	return &model.Cluster{ID: id}, nil
}

func (m *mockRestorationStore) GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error) {
	return nil, nil
}

type mockRestoreProvisioner struct {
	RestoreCompleteTime int64
	err                 error
}

func (p *mockRestoreProvisioner) TriggerRestore(installation *model.Installation, backup *model.InstallationBackup, cluster *model.Cluster) error {
	return p.err
}

func (p *mockRestoreProvisioner) CheckRestoreStatus(backupMeta *model.InstallationBackup, cluster *model.Cluster) (int64, error) {
	return p.RestoreCompleteTime, p.err
}

func (p *mockRestoreProvisioner) CleanupRestoreJob(backup *model.InstallationBackup, cluster *model.Cluster) error {
	return p.err
}

func TestInstallationDBRestorationSupervisor_Do(t *testing.T) {
	t.Run("no installation restoration operations pending work", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockRestorationStore{}

		restorationSupervisor := supervisor.NewInstallationDBRestorationSupervisor(mockStore, &mockAWS{}, &mockRestoreProvisioner{}, "instanceID", logger)
		err := restorationSupervisor.Do()
		require.NoError(t, err)

		require.Equal(t, 0, mockStore.UpdateRestorationOperationCalls)
	})

	t.Run("mock restoration trigger", func(t *testing.T) {
		logger := testlib.MakeLogger(t)

		installation := &model.Installation{
			ID:        model.NewID(),
			State:     model.InstallationStateHibernating,
			Database:  model.InstallationDatabaseMultiTenantRDSPostgres,
			Filestore: model.InstallationFilestoreBifrost,
		}
		mockStore := &mockRestorationStore{
			Installation: installation,
			RestorationPending: []*model.InstallationDBRestorationOperation{
				{ID: model.NewID(), InstallationID: installation.ID, State: model.InstallationDBRestorationStateRequested},
			},
			InstallationRestorationOperation: &model.InstallationDBRestorationOperation{ID: model.NewID(), InstallationID: installation.ID, State: model.InstallationDBRestorationStateRequested},
			UnlockChan:                       make(chan interface{}),
		}

		restorationSupervisor := supervisor.NewInstallationDBRestorationSupervisor(mockStore, &mockAWS{}, &mockRestoreProvisioner{}, "instanceID", logger)
		err := restorationSupervisor.Do()
		require.NoError(t, err)

		<-mockStore.UnlockChan
		require.Equal(t, 2, mockStore.UpdateRestorationOperationCalls)
	})
}

func TestInstallationDBRestorationSupervisor_Supervise(t *testing.T) {

	t.Run("transition to restoration", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		mockRestoreOp := &mockRestoreProvisioner{}

		installation, clusterInstallation, backup := setupRestoreRequiredResources(t, sqlStore)

		restorationOp := &model.InstallationDBRestorationOperation{
			InstallationID:          installation.ID,
			BackupID:                backup.ID,
			State:                   model.InstallationDBRestorationStateRequested,
			TargetInstallationState: model.InstallationStateHibernating,
		}
		err := sqlStore.CreateInstallationDBRestorationOperation(restorationOp)
		require.NoError(t, err)

		restorationSupervisor := supervisor.NewInstallationDBRestorationSupervisor(sqlStore, &mockAWS{}, mockRestoreOp, "instanceID", logger)
		restorationSupervisor.Supervise(restorationOp)

		// Assert
		restorationOp, err = sqlStore.GetInstallationDBRestorationOperation(restorationOp.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationDBRestorationStateInProgress, restorationOp.State)
		assert.Equal(t, clusterInstallation.ID, restorationOp.ClusterInstallationID)

		installation, err = sqlStore.GetInstallation(installation.ID, false, false)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationStateDBRestorationInProgress, installation.State)
	})

	t.Run("check restoration status", func(t *testing.T) {
		for _, testCase := range []struct {
			description   string
			mockRestoreOp *mockRestoreProvisioner
			expectedState model.InstallationDBRestorationState
		}{
			{
				description:   "when restore finished",
				mockRestoreOp: &mockRestoreProvisioner{RestoreCompleteTime: 100},
				expectedState: model.InstallationDBRestorationStateFinalizing,
			},
			{
				description:   "when still in progress",
				mockRestoreOp: &mockRestoreProvisioner{RestoreCompleteTime: -1},
				expectedState: model.InstallationDBRestorationStateInProgress,
			},
			{
				description:   "when non terminal error",
				mockRestoreOp: &mockRestoreProvisioner{RestoreCompleteTime: -1, err: errors.New("some error")},
				expectedState: model.InstallationDBRestorationStateInProgress,
			},
			{
				description:   "when terminal error",
				mockRestoreOp: &mockRestoreProvisioner{RestoreCompleteTime: -1, err: provisioner.ErrJobBackoffLimitReached},
				expectedState: model.InstallationDBRestorationStateFailing,
			},
		} {
			t.Run(testCase.description, func(t *testing.T) {
				logger := testlib.MakeLogger(t)
				sqlStore := store.MakeTestSQLStore(t, logger)
				defer store.CloseConnection(t, sqlStore)

				installation, clusterInstallation, backup := setupRestoreRequiredResources(t, sqlStore)

				restorationOp := &model.InstallationDBRestorationOperation{
					InstallationID:        installation.ID,
					BackupID:              backup.ID,
					State:                 model.InstallationDBRestorationStateInProgress,
					ClusterInstallationID: clusterInstallation.ID,
				}
				err := sqlStore.CreateInstallationDBRestorationOperation(restorationOp)
				require.NoError(t, err)

				restorationSupervisor := supervisor.NewInstallationDBRestorationSupervisor(sqlStore, &mockAWS{}, testCase.mockRestoreOp, "instanceID", logger)
				restorationSupervisor.Supervise(restorationOp)

				// Assert
				restorationOp, err = sqlStore.GetInstallationDBRestorationOperation(restorationOp.ID)
				require.NoError(t, err)
				assert.Equal(t, testCase.expectedState, restorationOp.State)
				assert.Equal(t, clusterInstallation.ID, restorationOp.ClusterInstallationID)
			})
		}
	})

	t.Run("finalizing restoration", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		mockRestoreOp := &mockRestoreProvisioner{}

		installation, clusterInstallation, backup := setupRestoreRequiredResources(t, sqlStore)

		restorationOp := &model.InstallationDBRestorationOperation{
			InstallationID:          installation.ID,
			BackupID:                backup.ID,
			State:                   model.InstallationDBRestorationStateFinalizing,
			ClusterInstallationID:   clusterInstallation.ID,
			TargetInstallationState: model.InstallationStateHibernating,
		}
		err := sqlStore.CreateInstallationDBRestorationOperation(restorationOp)
		require.NoError(t, err)

		restorationSupervisor := supervisor.NewInstallationDBRestorationSupervisor(sqlStore, &mockAWS{}, mockRestoreOp, "instanceID", logger)
		restorationSupervisor.Supervise(restorationOp)

		// Assert
		restorationOp, err = sqlStore.GetInstallationDBRestorationOperation(restorationOp.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationDBRestorationStateSucceeded, restorationOp.State)

		installation, err = sqlStore.GetInstallation(installation.ID, false, false)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationStateHibernating, installation.State)
	})

	t.Run("failing restoration", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		mockRestoreOp := &mockRestoreProvisioner{}

		installation, clusterInstallation, backup := setupRestoreRequiredResources(t, sqlStore)

		restorationOp := &model.InstallationDBRestorationOperation{
			InstallationID:          installation.ID,
			BackupID:                backup.ID,
			State:                   model.InstallationDBRestorationStateFailing,
			ClusterInstallationID:   clusterInstallation.ID,
			TargetInstallationState: model.InstallationStateHibernating,
		}
		err := sqlStore.CreateInstallationDBRestorationOperation(restorationOp)
		require.NoError(t, err)

		restorationSupervisor := supervisor.NewInstallationDBRestorationSupervisor(sqlStore, &mockAWS{}, mockRestoreOp, "instanceID", logger)
		restorationSupervisor.Supervise(restorationOp)

		// Assert
		restorationOp, err = sqlStore.GetInstallationDBRestorationOperation(restorationOp.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationDBRestorationStateFailed, restorationOp.State)

		installation, err = sqlStore.GetInstallation(installation.ID, false, false)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationStateDBRestorationFailed, installation.State)
	})

	t.Run("cleanup restoration", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		mockRestoreOp := &mockRestoreProvisioner{}

		installation, clusterInstallation, backup := setupRestoreRequiredResources(t, sqlStore)

		restorationOp := &model.InstallationDBRestorationOperation{
			InstallationID:        installation.ID,
			BackupID:              backup.ID,
			State:                 model.InstallationDBRestorationStateDeletionRequested,
			ClusterInstallationID: clusterInstallation.ID,
		}
		err := sqlStore.CreateInstallationDBRestorationOperation(restorationOp)
		require.NoError(t, err)

		restorationSupervisor := supervisor.NewInstallationDBRestorationSupervisor(sqlStore, &mockAWS{}, mockRestoreOp, "instanceID", logger)
		restorationSupervisor.Supervise(restorationOp)

		// Assert
		restorationOp, err = sqlStore.GetInstallationDBRestorationOperation(restorationOp.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationDBRestorationStateDeleted, restorationOp.State)
		assert.True(t, restorationOp.DeleteAt > 0)
	})
}

func setupRestoreRequiredResources(t *testing.T, sqlStore *store.SQLStore) (*model.Installation, *model.ClusterInstallation, *model.InstallationBackup) {
	installation := &model.Installation{
		Database:  model.InstallationDatabaseMultiTenantRDSPostgres,
		Filestore: model.InstallationFilestoreBifrost,
		State:     model.InstallationStateDBRestorationInProgress,
		DNS:       fmt.Sprintf("dns-%s", uuid.NewRandom().String()[:6]),
	}
	err := sqlStore.CreateInstallation(installation, nil)
	require.NoError(t, err)

	cluster := &model.Cluster{}
	err = sqlStore.CreateCluster(cluster, nil)
	require.NoError(t, err)

	clusterInstallation := &model.ClusterInstallation{InstallationID: installation.ID, ClusterID: cluster.ID}
	err = sqlStore.CreateClusterInstallation(clusterInstallation)
	require.NoError(t, err)

	backup := &model.InstallationBackup{
		InstallationID: installation.ID,
		State:          model.InstallationBackupStateBackupSucceeded,
	}
	err = sqlStore.CreateInstallationBackup(backup)
	require.NoError(t, err)

	return installation, clusterInstallation, backup
}
