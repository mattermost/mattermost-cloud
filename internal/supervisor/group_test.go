// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor_test

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	mmv1alpha1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1alpha1"
	"github.com/stretchr/testify/require"
)

type mockGroupStore struct {
	Group                     *model.Group
	UnlockedGroupsPendingWork []*model.Group
	GroupRollingMetadata      *store.GroupRollingMetadata

	Installation *model.Installation

	UnlockChan              chan interface{}
	UpdateInstallationCalls int
}

func (s *mockGroupStore) GetUnlockedGroupsPendingWork() ([]*model.Group, error) {
	return s.UnlockedGroupsPendingWork, nil
}

func (s *mockGroupStore) GetGroupRollingMetadata(groupID string) (*store.GroupRollingMetadata, error) {
	return s.GroupRollingMetadata, nil
}

func (s *mockGroupStore) LockGroup(groupID, lockerID string) (bool, error) {
	return true, nil
}

func (s *mockGroupStore) UnlockGroup(groupID, lockerID string, force bool) (bool, error) {
	if s.UnlockChan != nil {
		close(s.UnlockChan)
	}
	return true, nil
}

func (s *mockGroupStore) GetInstallation(installationID string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.Installation, error) {
	return s.Installation, nil
}

func (s *mockGroupStore) UpdateInstallation(installation *model.Installation) error {
	s.UpdateInstallationCalls++
	return nil
}

func (s *mockGroupStore) UpdateInstallationState(installation *model.Installation) error {
	s.UpdateInstallationCalls++
	return nil
}

func (s *mockGroupStore) LockInstallation(installationID, lockerID string) (bool, error) {
	return true, nil
}

func (s *mockGroupStore) UnlockInstallation(installationID, lockerID string, force bool) (bool, error) {
	if s.UnlockChan != nil {
		close(s.UnlockChan)
	}
	return true, nil
}

func (s *mockGroupStore) GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error) {
	return nil, nil
}

func TestGroupSupervisorDo(t *testing.T) {
	t.Run("no groups pending work", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockGroupStore{}

		supervisor := supervisor.NewGroupSupervisor(mockStore, "instanceID", logger)
		err := supervisor.Do()
		require.NoError(t, err)

		require.Equal(t, 0, mockStore.UpdateInstallationCalls)
	})

	t.Run("mock group and installation creation", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockGroupStore{}

		mockStore.UnlockedGroupsPendingWork = []*model.Group{{
			ID: model.NewID(),
		}}
		mockStore.Group = mockStore.UnlockedGroupsPendingWork[0]

		mockStore.Installation = &model.Installation{
			ID:      model.NewID(),
			GroupID: &mockStore.Group.ID,
		}

		mockStore.GroupRollingMetadata = &store.GroupRollingMetadata{
			InstallationIDsToBeRolled: []string{mockStore.Installation.ID},
		}
		mockStore.UnlockChan = make(chan interface{})

		supervisor := supervisor.NewGroupSupervisor(mockStore, "instanceID", logger)
		err := supervisor.Do()
		require.NoError(t, err)

		<-mockStore.UnlockChan
		require.Equal(t, 0, mockStore.UpdateInstallationCalls)
	})
}

func TestGroupSupervisor(t *testing.T) {
	standardGroup := func() *model.Group {
		return &model.Group{
			Version:    model.NewID(),
			Image:      model.NewID(),
			MaxRolling: 1,
		}
	}

	expectInstallations := func(t *testing.T, sqlStore *store.SQLStore, expectedCount int, state string) {
		t.Helper()

		installations, err := sqlStore.GetInstallations(&model.InstallationFilter{
			PerPage: model.AllPerPage,
		}, true, true)
		require.NoError(t, err)
		require.Len(t, installations, expectedCount)
		for _, installation := range installations {
			require.Equal(t, state, installation.State)
		}
	}

	expectInstallationStateCounts := func(t *testing.T, sqlStore *store.SQLStore, expectedStateCounts map[string]int) {
		t.Helper()

		installations, err := sqlStore.GetInstallations(&model.InstallationFilter{
			PerPage: model.AllPerPage,
		}, true, true)
		require.NoError(t, err)

		actualStateCounts := make(map[string]int)
		for _, installation := range installations {
			if _, ok := actualStateCounts[installation.State]; ok {
				actualStateCounts[installation.State]++
			} else {
				actualStateCounts[installation.State] = 1
			}
		}

		require.Equal(t, expectedStateCounts, actualStateCounts)
	}

	t.Run("no installations", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewGroupSupervisor(sqlStore, "instanceID", logger)

		group := standardGroup()
		err := sqlStore.CreateGroup(group)
		require.NoError(t, err)

		supervisor.Supervise(group)
		expectInstallations(t, sqlStore, 0, model.InstallationStateUpdateRequested)
	})

	t.Run("one installation, stable", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewGroupSupervisor(sqlStore, "instanceID", logger)

		group := standardGroup()
		err := sqlStore.CreateGroup(group)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		installation := &model.Installation{
			OwnerID:  model.NewID(),
			Version:  "version",
			DNS:      "dns1.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &group.ID,
			State:    model.InstallationStateStable,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		supervisor.Supervise(group)
		expectInstallations(t, sqlStore, 1, model.InstallationStateUpdateRequested)
	})

	t.Run("three installations, stable", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewGroupSupervisor(sqlStore, "instanceID", logger)

		group := standardGroup()
		group.MaxRolling = 10
		err := sqlStore.CreateGroup(group)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		err = sqlStore.CreateInstallation(&model.Installation{
			OwnerID:  model.NewID(),
			Version:  "version",
			DNS:      "dns1.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &group.ID,
			State:    model.InstallationStateStable,
		}, nil)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		err = sqlStore.CreateInstallation(&model.Installation{
			OwnerID:  model.NewID(),
			Version:  "version",
			DNS:      "dns2.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &group.ID,
			State:    model.InstallationStateStable,
		}, nil)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		err = sqlStore.CreateInstallation(&model.Installation{
			OwnerID:  model.NewID(),
			Version:  "version",
			DNS:      "dns3.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &group.ID,
			State:    model.InstallationStateStable,
		}, nil)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		supervisor.Supervise(group)
		expectInstallations(t, sqlStore, 3, model.InstallationStateUpdateRequested)
	})

	t.Run("one installation, not stable", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		supervisor := supervisor.NewGroupSupervisor(sqlStore, "instanceID", logger)

		group := standardGroup()
		err := sqlStore.CreateGroup(group)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		installation := &model.Installation{
			OwnerID:  model.NewID(),
			Version:  "version",
			DNS:      "dns1.example.com",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			GroupID:  &group.ID,
			State:    model.InstallationStateDeletionRequested,
		}

		err = sqlStore.CreateInstallation(installation, nil)
		require.NoError(t, err)

		supervisor.Supervise(group)
		expectInstallations(t, sqlStore, 1, model.InstallationStateDeletionRequested)
	})

	t.Run("more than max rolling", func(t *testing.T) {
		t.Run("two installations, stable", func(t *testing.T) {
			logger := testlib.MakeLogger(t)
			sqlStore := store.MakeTestSQLStore(t, logger)
			supervisor := supervisor.NewGroupSupervisor(sqlStore, "instanceID", logger)

			group := standardGroup()
			err := sqlStore.CreateGroup(group)
			require.NoError(t, err)

			time.Sleep(1 * time.Millisecond)

			err = sqlStore.CreateInstallation(&model.Installation{
				OwnerID:  model.NewID(),
				Version:  "version",
				DNS:      "dns1.example.com",
				Size:     mmv1alpha1.Size100String,
				Affinity: model.InstallationAffinityIsolated,
				GroupID:  &group.ID,
				State:    model.InstallationStateStable,
			}, nil)
			require.NoError(t, err)

			time.Sleep(1 * time.Millisecond)

			err = sqlStore.CreateInstallation(&model.Installation{
				OwnerID:  model.NewID(),
				Version:  "version",
				DNS:      "dns2.example.com",
				Size:     mmv1alpha1.Size100String,
				Affinity: model.InstallationAffinityIsolated,
				GroupID:  &group.ID,
				State:    model.InstallationStateStable,
			}, nil)
			require.NoError(t, err)

			supervisor.Supervise(group)
			expected := map[string]int{
				model.InstallationStateUpdateRequested: 1,
				model.ClusterInstallationStateStable:   1,
			}
			expectInstallationStateCounts(t, sqlStore, expected)
		})

		t.Run("two installations, one stable", func(t *testing.T) {
			logger := testlib.MakeLogger(t)
			sqlStore := store.MakeTestSQLStore(t, logger)
			supervisor := supervisor.NewGroupSupervisor(sqlStore, "instanceID", logger)

			group := standardGroup()
			err := sqlStore.CreateGroup(group)
			require.NoError(t, err)

			time.Sleep(1 * time.Millisecond)

			err = sqlStore.CreateInstallation(&model.Installation{
				OwnerID:  model.NewID(),
				Version:  "version",
				DNS:      "dns1.example.com",
				Size:     mmv1alpha1.Size100String,
				Affinity: model.InstallationAffinityIsolated,
				GroupID:  &group.ID,
				State:    model.InstallationStateStable,
			}, nil)
			require.NoError(t, err)

			time.Sleep(1 * time.Millisecond)

			err = sqlStore.CreateInstallation(&model.Installation{
				OwnerID:  model.NewID(),
				Version:  "version",
				DNS:      "dns2.example.com",
				Size:     mmv1alpha1.Size100String,
				Affinity: model.InstallationAffinityIsolated,
				GroupID:  &group.ID,
				State:    model.InstallationStateDeletionInProgress,
			}, nil)
			require.NoError(t, err)

			supervisor.Supervise(group)
			expected := map[string]int{
				model.InstallationStateDeletionInProgress: 1,
				model.ClusterInstallationStateStable:      1,
			}
			expectInstallationStateCounts(t, sqlStore, expected)
		})
	})
}
