// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor_test

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/provisioner"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/internal/testutil"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBackupStore struct {
	BackupMetadata        *model.InstallationBackup
	BackupMetadataPending []*model.InstallationBackup
	Cluster               *model.Cluster
	Installation          *model.Installation
	ClusterInstallations  []*model.ClusterInstallation
	UnlockChan            chan interface{}

	UpdateBackupMetadataCalls int
}

func (s mockBackupStore) GetUnlockedInstallationBackupPendingWork() ([]*model.InstallationBackup, error) {
	return s.BackupMetadataPending, nil
}

func (s mockBackupStore) GetInstallationBackup(id string) (*model.InstallationBackup, error) {
	return s.BackupMetadataPending[0], nil
}

func (s *mockBackupStore) UpdateInstallationBackupState(backupMeta *model.InstallationBackup) error {
	s.UpdateBackupMetadataCalls++
	return nil
}

func (s *mockBackupStore) UpdateInstallationBackupSchedulingData(backupMeta *model.InstallationBackup) error {
	s.UpdateBackupMetadataCalls++
	return nil
}

func (s mockBackupStore) UpdateInstallationBackupStartTime(backupMeta *model.InstallationBackup) error {
	panic("implement me")
}

func (s mockBackupStore) LockInstallationBackup(installationID, lockerID string) (bool, error) {
	return true, nil
}

func (s *mockBackupStore) UnlockInstallationBackup(installationID, lockerID string, force bool) (bool, error) {
	if s.UnlockChan != nil {
		close(s.UnlockChan)
	}
	return true, nil
}

func (s mockBackupStore) GetInstallation(installationID string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.Installation, error) {
	return s.Installation, nil
}

func (s mockBackupStore) LockInstallation(installationID, lockerID string) (bool, error) {
	return true, nil
}

func (s mockBackupStore) UnlockInstallation(installationID, lockerID string, force bool) (bool, error) {
	return true, nil
}

func (s mockBackupStore) GetClusterInstallations(filter *model.ClusterInstallationFilter) ([]*model.ClusterInstallation, error) {
	return s.ClusterInstallations, nil
}

func (s mockBackupStore) GetClusterInstallation(clusterInstallationID string) (*model.ClusterInstallation, error) {
	return s.ClusterInstallations[0], nil
}

func (s mockBackupStore) LockClusterInstallations(clusterInstallationID []string, lockerID string) (bool, error) {
	return true, nil
}

func (s mockBackupStore) UnlockClusterInstallations(clusterInstallationID []string, lockerID string, force bool) (bool, error) {
	return true, nil
}

func (s mockBackupStore) GetCluster(id string) (*model.Cluster, error) {
	return s.Cluster, nil
}

func (s mockBackupStore) GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error) {
	return nil, nil
}

type mockBackupProvisioner struct {
	BackupStartTime int64
	err             error
}

func (b *mockBackupProvisioner) TriggerBackup(backup *model.InstallationBackup, cluster *model.Cluster, installation *model.Installation) (*model.S3DataResidence, error) {
	return &model.S3DataResidence{URL: "file-store.com"}, b.err
}

func (b *mockBackupProvisioner) CheckBackupStatus(backup *model.InstallationBackup, cluster *model.Cluster) (int64, error) {
	return b.BackupStartTime, b.err
}

func (b *mockBackupProvisioner) CleanupBackup(backup *model.InstallationBackup, cluster *model.Cluster) error {
	return nil
}

func TestBackupSupervisorDo(t *testing.T) {
	t.Run("no backup pending work", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockBackupStore{}
		mockBackupOp := &mockBackupProvisioner{}

		backupSupervisor := supervisor.NewBackupSupervisor(mockStore, mockBackupOp, &mockAWS{}, "instanceID", logger)
		err := backupSupervisor.Do()
		require.NoError(t, err)

		require.Equal(t, 0, mockStore.UpdateBackupMetadataCalls)
	})

	t.Run("mock backup trigger", func(t *testing.T) {
		logger := testlib.MakeLogger(t)

		cluster := &model.Cluster{ID: model.NewID()}
		installation := &model.Installation{
			ID:        model.NewID(),
			State:     model.InstallationStateHibernating,
			Database:  model.InstallationDatabaseMultiTenantRDSPostgres,
			Filestore: model.InstallationFilestoreBifrost,
		}
		mockStore := &mockBackupStore{
			Cluster:      cluster,
			Installation: installation,
			BackupMetadataPending: []*model.InstallationBackup{
				{ID: model.NewID(), InstallationID: installation.ID, State: model.InstallationBackupStateBackupRequested},
			},
			ClusterInstallations: []*model.ClusterInstallation{{
				ID:             model.NewID(),
				ClusterID:      cluster.ID,
				InstallationID: installation.ID,
				State:          model.ClusterInstallationStateStable,
			}},
			UnlockChan: make(chan interface{}),
		}

		backupSupervisor := supervisor.NewBackupSupervisor(mockStore, &mockBackupProvisioner{}, &mockAWS{}, "instanceID", logger)
		err := backupSupervisor.Do()
		require.NoError(t, err)

		<-mockStore.UnlockChan
		require.Equal(t, 2, mockStore.UpdateBackupMetadataCalls)
	})
}

func TestBackupMetadataSupervisorSupervise(t *testing.T) {

	t.Run("trigger backup", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		mockBackupOp := &mockBackupProvisioner{}

		installation, clusterInstallation := setupBackupRequiredResources(t, sqlStore)

		backupMeta := &model.InstallationBackup{
			InstallationID: installation.ID,
			State:          model.InstallationBackupStateBackupRequested,
		}
		err := sqlStore.CreateInstallationBackup(backupMeta)
		require.NoError(t, err)

		backupSupervisor := supervisor.NewBackupSupervisor(sqlStore, mockBackupOp, &mockAWS{}, "instanceID", logger)
		backupSupervisor.Supervise(backupMeta)

		// Assert
		backupMeta, err = sqlStore.GetInstallationBackup(backupMeta.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationBackupStateBackupInProgress, backupMeta.State)
		assert.Equal(t, clusterInstallation.ID, backupMeta.ClusterInstallationID)
		assert.Equal(t, "file-store.com", backupMeta.DataResidence.URL)
	})

	t.Run("do not trigger backup if installation not hibernated", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		mockBackupOp := &mockBackupProvisioner{}

		installation, _ := setupBackupRequiredResources(t, sqlStore)
		installation.State = model.InstallationStateStable
		err := sqlStore.UpdateInstallationState(installation)
		require.NoError(t, err)

		backupMeta := &model.InstallationBackup{
			InstallationID: installation.ID,
			State:          model.InstallationBackupStateBackupRequested,
		}
		err = sqlStore.CreateInstallationBackup(backupMeta)
		require.NoError(t, err)

		backupSupervisor := supervisor.NewBackupSupervisor(sqlStore, mockBackupOp, &mockAWS{}, "instanceID", logger)
		backupSupervisor.Supervise(backupMeta)

		// Assert
		backupMeta, err = sqlStore.GetInstallationBackup(backupMeta.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationBackupStateBackupRequested, backupMeta.State)
	})

	t.Run("set backup as failed if installation deleted", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		mockBackupOp := &mockBackupProvisioner{}

		backupMeta := &model.InstallationBackup{
			InstallationID: "deleted-installation-id",
			State:          model.InstallationBackupStateBackupRequested,
		}
		err := sqlStore.CreateInstallationBackup(backupMeta)
		require.NoError(t, err)

		backupSupervisor := supervisor.NewBackupSupervisor(sqlStore, mockBackupOp, &mockAWS{}, "instanceID", logger)
		backupSupervisor.Supervise(backupMeta)

		// Assert
		backupMeta, err = sqlStore.GetInstallationBackup(backupMeta.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationBackupStateBackupFailed, backupMeta.State)
	})

	t.Run("check backup status", func(t *testing.T) {
		for _, testCase := range []struct {
			description   string
			mockBackupOp  *mockBackupProvisioner
			expectedState model.InstallationBackupState
		}{
			{
				description:   "when backup finished",
				mockBackupOp:  &mockBackupProvisioner{BackupStartTime: 100},
				expectedState: model.InstallationBackupStateBackupSucceeded,
			},
			{
				description:   "when still in progress",
				mockBackupOp:  &mockBackupProvisioner{BackupStartTime: -1},
				expectedState: model.InstallationBackupStateBackupInProgress,
			},
			{
				description:   "when non terminal error",
				mockBackupOp:  &mockBackupProvisioner{BackupStartTime: -1, err: errors.New("some error")},
				expectedState: model.InstallationBackupStateBackupInProgress,
			},
			{
				description:   "when terminal error",
				mockBackupOp:  &mockBackupProvisioner{BackupStartTime: -1, err: provisioner.ErrJobBackoffLimitReached},
				expectedState: model.InstallationBackupStateBackupFailed,
			},
		} {
			t.Run(testCase.description, func(t *testing.T) {
				logger := testlib.MakeLogger(t)
				sqlStore := store.MakeTestSQLStore(t, logger)

				installation, clusterInstallation := setupBackupRequiredResources(t, sqlStore)

				backupMeta := &model.InstallationBackup{
					InstallationID:        installation.ID,
					ClusterInstallationID: clusterInstallation.ID,
					State:                 model.InstallationBackupStateBackupInProgress,
				}
				err := sqlStore.CreateInstallationBackup(backupMeta)
				require.NoError(t, err)

				backupSupervisor := supervisor.NewBackupSupervisor(sqlStore, testCase.mockBackupOp, &mockAWS{}, "instanceID", logger)
				backupSupervisor.Supervise(backupMeta)

				// Assert
				backupMeta, err = sqlStore.GetInstallationBackup(backupMeta.ID)
				require.NoError(t, err)
				assert.Equal(t, testCase.expectedState, backupMeta.State)

				if testCase.mockBackupOp.BackupStartTime > 0 {
					assert.Equal(t, testCase.mockBackupOp.BackupStartTime, backupMeta.StartAt)
				}
			})
		}
	})
}

func setupBackupRequiredResources(t *testing.T, sqlStore *store.SQLStore) (*model.Installation, *model.ClusterInstallation) {
	installation := testutil.CreateBackupCompatibleInstallation(t, sqlStore)

	cluster := &model.Cluster{}
	err := sqlStore.CreateCluster(cluster, nil)
	require.NoError(t, err)

	clusterInstallation := &model.ClusterInstallation{InstallationID: installation.ID, ClusterID: cluster.ID}
	err = sqlStore.CreateClusterInstallation(clusterInstallation)
	require.NoError(t, err)

	return installation, clusterInstallation
}
