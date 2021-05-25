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
	"github.com/mattermost/mattermost-cloud/internal/tools/utils"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

type mockDBMigrationStore struct {
	DBMigrationOperation *model.InstallationDBMigrationOperation
	MigrationPending     []*model.InstallationDBMigrationOperation
	Installation         *model.Installation
	UnlockChan           chan interface{}

	UpdateMigrationOperationCalls int

	mockMultitenantDBStore
}

func (m *mockDBMigrationStore) GetUnlockedInstallationDBMigrationOperationsPendingWork() ([]*model.InstallationDBMigrationOperation, error) {
	return m.MigrationPending, nil
}

func (m *mockDBMigrationStore) GetInstallationDBMigrationOperation(id string) (*model.InstallationDBMigrationOperation, error) {
	return m.DBMigrationOperation, nil
}

func (m *mockDBMigrationStore) UpdateInstallationDBMigrationOperationState(dbMigration *model.InstallationDBMigrationOperation) error {
	m.UpdateMigrationOperationCalls++
	return nil
}

func (m *mockDBMigrationStore) UpdateInstallationDBMigrationOperation(dbMigration *model.InstallationDBMigrationOperation) error {
	m.UpdateMigrationOperationCalls++
	return nil
}

func (m *mockDBMigrationStore) DeleteInstallationDBMigrationOperation(id string) error {
	return nil
}

func (m *mockDBMigrationStore) LockInstallationDBMigrationOperations(id []string, lockerID string) (bool, error) {
	return true, nil
}

func (m *mockDBMigrationStore) UnlockInstallationDBMigrationOperations(id []string, lockerID string, force bool) (bool, error) {
	if m.UnlockChan != nil {
		close(m.UnlockChan)
	}
	return true, nil
}

func (m *mockDBMigrationStore) TriggerInstallationRestoration(installation *model.Installation, backup *model.InstallationBackup) (*model.InstallationDBRestorationOperation, error) {
	panic("implement me")
}

func (m *mockDBMigrationStore) GetInstallationDBRestorationOperation(id string) (*model.InstallationDBRestorationOperation, error) {
	panic("implement me")
}

func (m *mockDBMigrationStore) UpdateInstallationDBRestorationOperationState(dbRestoration *model.InstallationDBRestorationOperation) error {
	panic("implement me")
}

func (m *mockDBMigrationStore) UpdateInstallationDBRestorationOperation(dbRestoration *model.InstallationDBRestorationOperation) error {
	panic("implement me")
}

func (m *mockDBMigrationStore) IsInstallationBackupRunning(installationID string) (bool, error) {
	return false, nil
}

func (m *mockDBMigrationStore) CreateInstallationBackup(backup *model.InstallationBackup) error {
	return nil
}

func (m *mockDBMigrationStore) GetInstallationBackup(id string) (*model.InstallationBackup, error) {
	panic("implement me")
}

func (m *mockDBMigrationStore) UpdateInstallationBackupState(backupMeta *model.InstallationBackup) error {
	panic("implement me")
}

func (m *mockDBMigrationStore) LockInstallationBackups(backupsID []string, lockerID string) (bool, error) {
	panic("implement me")
}

func (m *mockDBMigrationStore) UnlockInstallationBackups(backupsID []string, lockerID string, force bool) (bool, error) {
	panic("implement me")
}

func (m *mockDBMigrationStore) GetInstallation(installationID string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.Installation, error) {
	return m.Installation, nil
}

func (m *mockDBMigrationStore) UpdateInstallation(installation *model.Installation) error {
	panic("implement me")
}

func (m *mockDBMigrationStore) LockInstallation(installationID, lockerID string) (bool, error) {
	return true, nil
}

func (m *mockDBMigrationStore) UnlockInstallation(installationID, lockerID string, force bool) (bool, error) {
	return true, nil
}

func (m *mockDBMigrationStore) GetClusterInstallations(filter *model.ClusterInstallationFilter) ([]*model.ClusterInstallation, error) {
	panic("implement me")
}

func (m *mockDBMigrationStore) GetClusterInstallation(clusterInstallationID string) (*model.ClusterInstallation, error) {
	panic("implement me")
}

func (m *mockDBMigrationStore) LockClusterInstallations(clusterInstallationID []string, lockerID string) (bool, error) {
	panic("implement me")
}

func (m *mockDBMigrationStore) UnlockClusterInstallations(clusterInstallationID []string, lockerID string, force bool) (bool, error) {
	panic("implement me")
}

func (m *mockDBMigrationStore) GetCluster(id string) (*model.Cluster, error) {
	panic("implement me")
}

func (m *mockDBMigrationStore) GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error) {
	return nil, nil
}

type mockDatabase struct{}

func (m *mockDatabase) TeardownMigrated(store model.InstallationDatabaseStoreInterface, migrationOp *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return nil
}

func (m *mockDatabase) RollbackMigration(store model.InstallationDatabaseStoreInterface, dbMigration *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return nil
}

func (m *mockDatabase) Provision(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	panic("implement me")
}

func (m *mockDatabase) Teardown(store model.InstallationDatabaseStoreInterface, keepData bool, logger log.FieldLogger) error {
	panic("implement me")
}

func (m *mockDatabase) Snapshot(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	panic("implement me")
}

func (m *mockDatabase) GenerateDatabaseSecret(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*corev1.Secret, error) {
	panic("implement me")
}

func (m *mockDatabase) RefreshResourceMetadata(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	panic("implement me")
}

func (m *mockDatabase) MigrateOut(store model.InstallationDatabaseStoreInterface, dbMigration *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return nil
}

func (m *mockDatabase) MigrateTo(store model.InstallationDatabaseStoreInterface, dbMigration *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return nil
}

type mockResourceUtil struct{}

func (m *mockResourceUtil) GetDatabase(installationID, dbType string) model.Database {
	return &mockDatabase{}
}

type mockMigrationProvisioner struct {
	expectedCommand []string
}

func (m *mockMigrationProvisioner) ClusterInstallationProvisioner(version string) provisioner.ClusterInstallationProvisioner {
	return &mockInstallationProvisioner{}
}

func (m *mockMigrationProvisioner) ExecClusterInstallationJob(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) error {
	return nil
}

func TestDBMigrationSupervisor_Do(t *testing.T) {
	t.Run("no installation migration operations pending work", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockDBMigrationStore{}

		dbMigrationSupervisor := supervisor.NewInstallationDBMigrationSupervisor(mockStore, &mockAWS{}, &utils.ResourceUtil{}, "instanceID", nil, logger)
		err := dbMigrationSupervisor.Do()
		require.NoError(t, err)

		require.Equal(t, 0, mockStore.UpdateMigrationOperationCalls)
	})

	t.Run("mock restoration trigger", func(t *testing.T) {
		logger := testlib.MakeLogger(t)

		installation := &model.Installation{
			ID:        model.NewID(),
			State:     model.InstallationStateHibernating,
			Database:  model.InstallationDatabaseMultiTenantRDSPostgres,
			Filestore: model.InstallationFilestoreBifrost,
		}
		mockStore := &mockDBMigrationStore{
			Installation: installation,
			MigrationPending: []*model.InstallationDBMigrationOperation{
				{ID: model.NewID(), InstallationID: installation.ID, State: model.InstallationDBMigrationStateRequested},
			},
			DBMigrationOperation: &model.InstallationDBMigrationOperation{ID: model.NewID(), InstallationID: installation.ID, State: model.InstallationDBMigrationStateRequested},
			UnlockChan:           make(chan interface{}),
		}

		dbMigrationSupervisor := supervisor.NewInstallationDBMigrationSupervisor(mockStore, &mockAWS{}, &utils.ResourceUtil{}, "instanceID", nil, logger)
		err := dbMigrationSupervisor.Do()
		require.NoError(t, err)

		<-mockStore.UnlockChan
		require.Equal(t, 2, mockStore.UpdateMigrationOperationCalls)
	})
}

func TestDBMigrationSupervisor_Supervise(t *testing.T) {

	t.Run("trigger backup", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		installation, _ := setupMigrationRequiredResources(t, sqlStore)

		migrationOp := &model.InstallationDBMigrationOperation{
			InstallationID: installation.ID,
			State:          model.InstallationDBMigrationStateRequested,
		}

		err := sqlStore.CreateInstallationDBMigrationOperation(migrationOp)
		require.NoError(t, err)

		dbMigrationSupervisor := supervisor.NewInstallationDBMigrationSupervisor(sqlStore, &mockAWS{}, &utils.ResourceUtil{}, "instanceID", nil, logger)
		dbMigrationSupervisor.Supervise(migrationOp)

		// Assert
		migrationOp, err = sqlStore.GetInstallationDBMigrationOperation(migrationOp.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationDBMigrationStateBackupInProgress, migrationOp.State)
		assert.NotEmpty(t, migrationOp.BackupID)

		backup, err := sqlStore.GetInstallationBackup(migrationOp.BackupID)
		require.NoError(t, err)
		require.NotNil(t, backup)
		assert.Equal(t, model.InstallationBackupStateBackupRequested, backup.State)
		assert.Equal(t, installation.ID, backup.InstallationID)
	})

	t.Run("wait for installation backup", func(t *testing.T) {
		for _, testCase := range []struct {
			description   string
			backupState   model.InstallationBackupState
			expectedState model.InstallationDBMigrationOperationState
		}{
			{
				description:   "when backup requested",
				backupState:   model.InstallationBackupStateBackupRequested,
				expectedState: model.InstallationDBMigrationStateBackupInProgress,
			},
			{
				description:   "when backup in progress",
				backupState:   model.InstallationBackupStateBackupInProgress,
				expectedState: model.InstallationDBMigrationStateBackupInProgress,
			},
			{
				description:   "when backup succeeded",
				backupState:   model.InstallationBackupStateBackupSucceeded,
				expectedState: model.InstallationDBMigrationStateDatabaseSwitch,
			},
			{
				description:   "when backup failed",
				backupState:   model.InstallationBackupStateBackupFailed,
				expectedState: model.InstallationDBMigrationStateFailing,
			},
		} {
			t.Run(testCase.description, func(t *testing.T) {
				logger := testlib.MakeLogger(t)
				sqlStore := store.MakeTestSQLStore(t, logger)
				defer store.CloseConnection(t, sqlStore)

				installation, _ := setupMigrationRequiredResources(t, sqlStore)

				backup := &model.InstallationBackup{
					InstallationID: installation.ID,
					State:          testCase.backupState,
				}
				err := sqlStore.CreateInstallationBackup(backup)
				require.NoError(t, err)

				migrationOp := &model.InstallationDBMigrationOperation{
					InstallationID: installation.ID,
					State:          model.InstallationDBMigrationStateBackupInProgress,
					BackupID:       backup.ID,
				}

				err = sqlStore.CreateInstallationDBMigrationOperation(migrationOp)
				require.NoError(t, err)

				dbMigrationSupervisor := supervisor.NewInstallationDBMigrationSupervisor(sqlStore, &mockAWS{}, &utils.ResourceUtil{}, "instanceID", nil, logger)
				dbMigrationSupervisor.Supervise(migrationOp)

				// Assert
				migrationOp, err = sqlStore.GetInstallationDBMigrationOperation(migrationOp.ID)
				require.NoError(t, err)
				assert.Equal(t, testCase.expectedState, migrationOp.State)
			})
		}
	})

	t.Run("switch database", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		installation, _ := setupMigrationRequiredResources(t, sqlStore)

		migrationOp := &model.InstallationDBMigrationOperation{
			InstallationID:         installation.ID,
			State:                  model.InstallationDBMigrationStateDatabaseSwitch,
			SourceDatabase:         model.InstallationDatabaseMultiTenantRDSPostgres,
			DestinationDatabase:    model.InstallationDatabaseSingleTenantRDSPostgres,
			SourceMultiTenant:      &model.MultiTenantDBMigrationData{DatabaseID: "source-id"},
			DestinationMultiTenant: &model.MultiTenantDBMigrationData{DatabaseID: "destination-id"},
		}

		err := sqlStore.CreateInstallationDBMigrationOperation(migrationOp)
		require.NoError(t, err)

		dbMigrationSupervisor := supervisor.NewInstallationDBMigrationSupervisor(sqlStore, &mockAWS{}, &mockResourceUtil{}, "instanceID", nil, logger)
		dbMigrationSupervisor.Supervise(migrationOp)

		// Assert
		migrationOp, err = sqlStore.GetInstallationDBMigrationOperation(migrationOp.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationDBMigrationStateRefreshSecrets, migrationOp.State)

		installation, err = sqlStore.GetInstallation(installation.ID, false, false)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationDatabaseSingleTenantRDSPostgres, installation.Database)
	})

	t.Run("refresh secrets", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		installation, _ := setupMigrationRequiredResources(t, sqlStore)

		migrationOp := &model.InstallationDBMigrationOperation{
			InstallationID: installation.ID,
			State:          model.InstallationDBMigrationStateRefreshSecrets,
		}

		err := sqlStore.CreateInstallationDBMigrationOperation(migrationOp)
		require.NoError(t, err)

		dbMigrationSupervisor := supervisor.NewInstallationDBMigrationSupervisor(sqlStore, &mockAWS{}, &mockResourceUtil{}, "instanceID", &mockMigrationProvisioner{}, logger)
		dbMigrationSupervisor.Supervise(migrationOp)

		// Assert
		migrationOp, err = sqlStore.GetInstallationDBMigrationOperation(migrationOp.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationDBMigrationStateTriggerRestoration, migrationOp.State)
	})

	t.Run("trigger restoration", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		installation, _ := setupMigrationRequiredResources(t, sqlStore)

		backup := &model.InstallationBackup{
			InstallationID: installation.ID,
			State:          model.InstallationBackupStateBackupSucceeded,
		}
		err := sqlStore.CreateInstallationBackup(backup)
		require.NoError(t, err)

		migrationOp := &model.InstallationDBMigrationOperation{
			InstallationID: installation.ID,
			State:          model.InstallationDBMigrationStateTriggerRestoration,
			BackupID:       backup.ID,
		}

		err = sqlStore.CreateInstallationDBMigrationOperation(migrationOp)
		require.NoError(t, err)

		dbMigrationSupervisor := supervisor.NewInstallationDBMigrationSupervisor(sqlStore, &mockAWS{}, &mockResourceUtil{}, "instanceID", nil, logger)
		dbMigrationSupervisor.Supervise(migrationOp)

		// Assert
		migrationOp, err = sqlStore.GetInstallationDBMigrationOperation(migrationOp.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationDBMigrationStateRestorationInProgress, migrationOp.State)
		assert.NotEmpty(t, migrationOp.InstallationDBRestorationOperationID)

		restorationOp, err := sqlStore.GetInstallationDBRestorationOperation(migrationOp.InstallationDBRestorationOperationID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationDBRestorationStateRequested, restorationOp.State)
		assert.Equal(t, installation.ID, restorationOp.InstallationID)
		assert.Equal(t, backup.ID, restorationOp.BackupID)
	})

	t.Run("trigger restoration - fail if no backup", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		installation, _ := setupMigrationRequiredResources(t, sqlStore)

		migrationOp := &model.InstallationDBMigrationOperation{
			InstallationID: installation.ID,
			State:          model.InstallationDBMigrationStateTriggerRestoration,
		}

		err := sqlStore.CreateInstallationDBMigrationOperation(migrationOp)
		require.NoError(t, err)

		dbMigrationSupervisor := supervisor.NewInstallationDBMigrationSupervisor(sqlStore, &mockAWS{}, &mockResourceUtil{}, "instanceID", nil, logger)
		dbMigrationSupervisor.Supervise(migrationOp)

		// Assert
		migrationOp, err = sqlStore.GetInstallationDBMigrationOperation(migrationOp.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationDBMigrationStateFailing, migrationOp.State)
	})

	t.Run("wait for installation restoration", func(t *testing.T) {
		for _, testCase := range []struct {
			description        string
			restorationOpState model.InstallationDBRestorationState
			expectedState      model.InstallationDBMigrationOperationState
		}{
			{
				description:        "when restoration requested",
				restorationOpState: model.InstallationDBRestorationStateRequested,
				expectedState:      model.InstallationDBMigrationStateRestorationInProgress,
			},
			{
				description:        "when restoration in progress",
				restorationOpState: model.InstallationDBRestorationStateInProgress,
				expectedState:      model.InstallationDBMigrationStateRestorationInProgress,
			},
			{
				description:        "when restoration finalizing",
				restorationOpState: model.InstallationDBRestorationStateFinalizing,
				expectedState:      model.InstallationDBMigrationStateRestorationInProgress,
			},
			{
				description:        "when restoration succeeded",
				restorationOpState: model.InstallationDBRestorationStateSucceeded,
				expectedState:      model.InstallationDBMigrationStateUpdatingInstallationConfig,
			},
			{
				description:        "when restoration failed",
				restorationOpState: model.InstallationDBRestorationStateFailed,
				expectedState:      model.InstallationDBMigrationStateFailing,
			},
			{
				description:        "when restoration invalid",
				restorationOpState: model.InstallationDBRestorationStateInvalid,
				expectedState:      model.InstallationDBMigrationStateFailing,
			},
		} {
			t.Run(testCase.description, func(t *testing.T) {
				logger := testlib.MakeLogger(t)
				sqlStore := store.MakeTestSQLStore(t, logger)
				defer store.CloseConnection(t, sqlStore)

				installation, _ := setupMigrationRequiredResources(t, sqlStore)

				restorationOp := &model.InstallationDBRestorationOperation{
					InstallationID: installation.ID,
					State:          testCase.restorationOpState,
				}
				err := sqlStore.CreateInstallationDBRestorationOperation(restorationOp)
				require.NoError(t, err)

				migrationOp := &model.InstallationDBMigrationOperation{
					InstallationID:                       installation.ID,
					State:                                model.InstallationDBMigrationStateRestorationInProgress,
					InstallationDBRestorationOperationID: restorationOp.ID,
				}

				err = sqlStore.CreateInstallationDBMigrationOperation(migrationOp)
				require.NoError(t, err)

				dbMigrationSupervisor := supervisor.NewInstallationDBMigrationSupervisor(sqlStore, &mockAWS{}, &utils.ResourceUtil{}, "instanceID", nil, logger)
				dbMigrationSupervisor.Supervise(migrationOp)

				// Assert
				migrationOp, err = sqlStore.GetInstallationDBMigrationOperation(migrationOp.ID)
				require.NoError(t, err)
				assert.Equal(t, testCase.expectedState, migrationOp.State)
			})
		}
	})

	t.Run("update installation config", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		installation, _ := setupMigrationRequiredResources(t, sqlStore)

		migrationOp := &model.InstallationDBMigrationOperation{
			InstallationID: installation.ID,
			State:          model.InstallationDBMigrationStateUpdatingInstallationConfig,
		}

		err := sqlStore.CreateInstallationDBMigrationOperation(migrationOp)
		require.NoError(t, err)

		dbMigrationSupervisor := supervisor.NewInstallationDBMigrationSupervisor(sqlStore, &mockAWS{}, &mockResourceUtil{}, "instanceID", &mockMigrationProvisioner{}, logger)
		dbMigrationSupervisor.Supervise(migrationOp)

		// Assert
		migrationOp, err = sqlStore.GetInstallationDBMigrationOperation(migrationOp.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationDBMigrationStateFinalizing, migrationOp.State)
	})

	t.Run("finalizing migration", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		installation, _ := setupMigrationRequiredResources(t, sqlStore)

		migrationOp := &model.InstallationDBMigrationOperation{
			InstallationID: installation.ID,
			State:          model.InstallationDBMigrationStateFinalizing,
		}

		err := sqlStore.CreateInstallationDBMigrationOperation(migrationOp)
		require.NoError(t, err)

		dbMigrationSupervisor := supervisor.NewInstallationDBMigrationSupervisor(sqlStore, &mockAWS{}, &mockResourceUtil{}, "instanceID", nil, logger)
		dbMigrationSupervisor.Supervise(migrationOp)

		// Assert
		migrationOp, err = sqlStore.GetInstallationDBMigrationOperation(migrationOp.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationDBMigrationStateSucceeded, migrationOp.State)
		assert.True(t, migrationOp.CompleteAt > 0)

		installation, err = sqlStore.GetInstallation(installation.ID, false, false)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationStateHibernating, installation.State)
	})

	t.Run("failing migration", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		installation, _ := setupMigrationRequiredResources(t, sqlStore)

		migrationOp := &model.InstallationDBMigrationOperation{
			InstallationID: installation.ID,
			State:          model.InstallationDBMigrationStateFailing,
		}

		err := sqlStore.CreateInstallationDBMigrationOperation(migrationOp)
		require.NoError(t, err)

		dbMigrationSupervisor := supervisor.NewInstallationDBMigrationSupervisor(sqlStore, &mockAWS{}, &mockResourceUtil{}, "instanceID", nil, logger)
		dbMigrationSupervisor.Supervise(migrationOp)

		// Assert
		migrationOp, err = sqlStore.GetInstallationDBMigrationOperation(migrationOp.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationDBMigrationStateFailed, migrationOp.State)

		installation, err = sqlStore.GetInstallation(installation.ID, false, false)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationStateDBMigrationFailed, installation.State)
	})
}

func setupMigrationRequiredResources(t *testing.T, sqlStore *store.SQLStore) (*model.Installation, *model.ClusterInstallation) {
	installation := &model.Installation{
		Database:  model.InstallationDatabaseMultiTenantRDSPostgres,
		Filestore: model.InstallationFilestoreBifrost,
		State:     model.InstallationStateDBMigrationInProgress,
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

	return installation, clusterInstallation
}
