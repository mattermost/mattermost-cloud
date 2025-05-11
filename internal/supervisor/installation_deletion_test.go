// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor_test

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/events"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/internal/testutil"
	"github.com/mattermost/mattermost-cloud/model"
	mmv1alpha1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1alpha1"
	"github.com/stretchr/testify/require"
)

type mockInstallationDeletionStore struct {
	Installation                               *model.Installation
	UnlockedInstallationsWithScheduledDeletion []*model.Installation
	UnlockedInstallationsPendingDeletion       []*model.Installation
	Events                                     []*model.StateChangeEventData
	CurrentlyUpdatingCounter                   int64

	UnlockChan              chan interface{}
	UpdateInstallationCalls int

	mockMultitenantDBStore
}

func (s *mockInstallationDeletionStore) GetInstallation(installationID string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.Installation, error) {
	return s.Installation, nil
}

func (s *mockInstallationDeletionStore) GetUnlockedInstallationsWithScheduledDeletion() ([]*model.Installation, error) {
	return s.UnlockedInstallationsWithScheduledDeletion, nil
}

func (s *mockInstallationDeletionStore) GetUnlockedInstallationsPendingDeletion() ([]*model.Installation, error) {
	return s.UnlockedInstallationsPendingDeletion, nil
}

func (s *mockInstallationDeletionStore) GetInstallationsStatus() (*model.InstallationsStatus, error) {
	return &model.InstallationsStatus{InstallationsUpdating: s.CurrentlyUpdatingCounter}, nil
}

func (s *mockInstallationDeletionStore) UpdateInstallationState(installation *model.Installation) error {
	s.UpdateInstallationCalls++
	return nil
}

func (s *mockInstallationDeletionStore) LockInstallation(installationID, lockerID string) (bool, error) {
	return true, nil
}

func (s *mockInstallationDeletionStore) UnlockInstallation(installationID, lockerID string, force bool) (bool, error) {
	if s.UnlockChan != nil {
		close(s.UnlockChan)
	}
	return true, nil
}

func (s *mockInstallationDeletionStore) GetStateChangeEvents(filter *model.StateChangeEventFilter) ([]*model.StateChangeEventData, error) {
	return s.Events, nil
}

func (s *mockInstallationDeletionStore) GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error) {
	return nil, nil
}

type mockEventsProducer struct{}

func (s *mockEventsProducer) ProduceInstallationStateChangeEvent(installation *model.Installation, oldState string, extraDataFields ...events.DataField) error {
	return nil
}

func (s *mockEventsProducer) ProduceClusterStateChangeEvent(cluster *model.Cluster, oldState string, extraDataFields ...events.DataField) error {
	return nil
}

func (s *mockEventsProducer) ProduceClusterInstallationStateChangeEvent(clusterInstallation *model.ClusterInstallation, oldState string, extraDataFields ...events.DataField) error {
	return nil
}

func TestInstallationDeletionSupervisor_Do(t *testing.T) {
	t.Run("no installation deletion operations pending work", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockInstallationDeletionStore{}

		supervisor := supervisor.NewInstallationDeletionSupervisor("instanceID", time.Hour, 10, mockStore, &mockEventsProducer{}, logger)
		err := supervisor.Do()
		require.NoError(t, err)

		require.Equal(t, 0, mockStore.UpdateInstallationCalls)
	})

	t.Run("mock check installation scheduled deletion, scheduled deletion time has not passed", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockInstallationDeletionStore{}

		mockStore.UnlockedInstallationsWithScheduledDeletion = []*model.Installation{{
			ID:                    model.NewID(),
			State:                 model.InstallationStateStable,
			ScheduledDeletionTime: model.GetMillisAtTime(time.Now().Add(time.Minute)),
		}}
		mockStore.Installation = mockStore.UnlockedInstallationsWithScheduledDeletion[0]
		mockStore.UnlockChan = make(chan interface{})

		supervisor := supervisor.NewInstallationDeletionSupervisor("instanceID", time.Hour, 10, mockStore, &mockEventsProducer{}, logger)
		err := supervisor.Do()
		require.NoError(t, err)

		<-mockStore.UnlockChan
		require.Equal(t, 0, mockStore.UpdateInstallationCalls)
		require.Equal(t, model.InstallationStateStable, mockStore.Installation.State)
	})

	t.Run("mock check installation scheduled deletion, scheduled deletion time has passed", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockInstallationDeletionStore{}

		mockStore.UnlockedInstallationsWithScheduledDeletion = []*model.Installation{{
			ID:                    model.NewID(),
			State:                 model.InstallationStateStable,
			ScheduledDeletionTime: model.GetMillisAtTime(time.Now().Add(-time.Minute)),
		}}
		mockStore.Installation = mockStore.UnlockedInstallationsWithScheduledDeletion[0]
		mockStore.UnlockChan = make(chan interface{})

		supervisor := supervisor.NewInstallationDeletionSupervisor("instanceID", time.Hour, 10, mockStore, &mockEventsProducer{}, logger)
		err := supervisor.Do()
		require.NoError(t, err)

		<-mockStore.UnlockChan
		require.Equal(t, 1, mockStore.UpdateInstallationCalls)
		require.Equal(t, model.InstallationStateDeletionPendingRequested, mockStore.Installation.State)
	})

	t.Run("mock check installation scheduled deletion, scheduled deletion time has passed, but deletion locked", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockInstallationDeletionStore{}

		mockStore.UnlockedInstallationsWithScheduledDeletion = []*model.Installation{{
			ID:                    model.NewID(),
			State:                 model.InstallationStateStable,
			ScheduledDeletionTime: model.GetMillisAtTime(time.Now().Add(-time.Minute)),
			DeletionLocked:        true,
		}}
		mockStore.Installation = mockStore.UnlockedInstallationsWithScheduledDeletion[0]
		mockStore.UnlockChan = make(chan interface{})

		supervisor := supervisor.NewInstallationDeletionSupervisor("instanceID", time.Hour, 10, mockStore, &mockEventsProducer{}, logger)
		err := supervisor.Do()
		require.NoError(t, err)

		<-mockStore.UnlockChan
		require.Equal(t, 0, mockStore.UpdateInstallationCalls)
		require.Equal(t, model.InstallationStateStable, mockStore.Installation.State)
	})

	t.Run("mock check installation deletion pending, deletion pending time has passed", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockInstallationDeletionStore{}

		mockStore.UnlockedInstallationsPendingDeletion = []*model.Installation{{
			ID:    model.NewID(),
			State: model.InstallationStateDeletionPending,
		}}
		mockStore.Installation = mockStore.UnlockedInstallationsPendingDeletion[0]
		mockStore.Events = []*model.StateChangeEventData{{Event: model.Event{Timestamp: 1646160276464}}}
		mockStore.UnlockChan = make(chan interface{})

		supervisor := supervisor.NewInstallationDeletionSupervisor("instanceID", time.Hour, 10, mockStore, &mockEventsProducer{}, logger)
		err := supervisor.Do()
		require.NoError(t, err)

		<-mockStore.UnlockChan
		require.Equal(t, 1, mockStore.UpdateInstallationCalls)
		require.Equal(t, model.InstallationStateDeletionRequested, mockStore.Installation.State)
	})
}

func TestInstallationDeletionSupervisor_Supervise(t *testing.T) {
	t.Run("unknown state", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		supervisor := supervisor.NewInstallationDeletionSupervisor("instanceID", time.Hour, 10, sqlStore, &mockEventsProducer{}, logger)

		installation := &model.Installation{
			OwnerID:  "blah",
			Version:  "version",
			Name:     "dns",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			State:    "badstate",
		}

		err := sqlStore.CreateInstallation(installation, nil, testutil.DNSForInstallation("dns.example.com"))
		require.NoError(t, err)

		supervisor.Supervise(installation)
		installation, err = sqlStore.GetInstallation(installation.ID, false, false)
		require.NoError(t, err)
		require.Equal(t, "badstate", installation.State)
	})

	t.Run("deletion pending, not ready for deletion", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		supervisor := supervisor.NewInstallationDeletionSupervisor("instanceID", time.Hour, 10, sqlStore, &mockEventsProducer{}, logger)

		installation := &model.Installation{
			OwnerID:  "blah",
			Version:  "version",
			Name:     "dns",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			State:    model.InstallationStateDeletionPending,
		}

		err := sqlStore.CreateInstallation(installation, nil, testutil.DNSForInstallation("dns.example.com"))
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		event := &model.StateChangeEventData{
			Event: model.Event{
				EventType: model.ResourceStateChangeEventType,
				Timestamp: model.GetMillis(),
			},
			StateChange: model.StateChangeEvent{
				OldState:     "old",
				NewState:     model.InstallationStateDeletionPending,
				ResourceID:   installation.ID,
				ResourceType: "installation",
			},
		}

		err = sqlStore.CreateStateChangeEvent(event)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		supervisor.Supervise(installation)
		installation, err = sqlStore.GetInstallation(installation.ID, false, false)
		require.NoError(t, err)
		require.Equal(t, model.InstallationStateDeletionPending, installation.State)
	})

	t.Run("deletion pending, ready for deletion", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		supervisor := supervisor.NewInstallationDeletionSupervisor("instanceID", time.Nanosecond, 10, sqlStore, &mockEventsProducer{}, logger)

		installation := &model.Installation{
			OwnerID:  "blah",
			Version:  "version",
			Name:     "dns",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			State:    model.InstallationStateDeletionPending,
		}

		err := sqlStore.CreateInstallation(installation, nil, testutil.DNSForInstallation("dns.example.com"))
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		event := &model.StateChangeEventData{
			Event: model.Event{
				EventType: model.ResourceStateChangeEventType,
				Timestamp: model.GetMillis(),
			},
			StateChange: model.StateChangeEvent{
				OldState:     "old",
				NewState:     model.InstallationStateDeletionPending,
				ResourceID:   installation.ID,
				ResourceType: "installation",
			},
		}

		err = sqlStore.CreateStateChangeEvent(event)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		supervisor.Supervise(installation)
		installation, err = sqlStore.GetInstallation(installation.ID, false, false)
		require.NoError(t, err)
		require.Equal(t, model.InstallationStateDeletionRequested, installation.State)
	})

	t.Run("deletion pending, ready for deletion, but max reached", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		supervisor := supervisor.NewInstallationDeletionSupervisor("instanceID", time.Nanosecond, 1, sqlStore, &mockEventsProducer{}, logger)

		installation1 := &model.Installation{
			OwnerID:  "blah",
			Version:  "version",
			Name:     "i1",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			State:    model.InstallationStateUpdateInProgress,
		}

		err := sqlStore.CreateInstallation(installation1, nil, testutil.DNSForInstallation("i1.example.com"))
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		installation2 := &model.Installation{
			OwnerID:  "blah",
			Version:  "version",
			Name:     "i2",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			State:    model.InstallationStateDeletionPending,
		}

		err = sqlStore.CreateInstallation(installation2, nil, testutil.DNSForInstallation("i2.example.com"))
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		status, err := sqlStore.GetInstallationsStatus()
		require.NoError(t, err)
		require.Equal(t, int64(2), status.InstallationsTotal)
		require.Equal(t, int64(1), status.InstallationsUpdating)

		event := &model.StateChangeEventData{
			Event: model.Event{
				EventType: model.ResourceStateChangeEventType,
				Timestamp: model.GetMillis(),
			},
			StateChange: model.StateChangeEvent{
				OldState:     "old",
				NewState:     model.InstallationStateDeletionPending,
				ResourceID:   installation2.ID,
				ResourceType: "installation",
			},
		}

		err = sqlStore.CreateStateChangeEvent(event)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		supervisor.Do()
		installation2, err = sqlStore.GetInstallation(installation2.ID, false, false)
		require.NoError(t, err)
		require.Equal(t, model.InstallationStateDeletionPending, installation2.State)
	})

	t.Run("deletion pending with expiry, not ready for deletion", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		supervisor := supervisor.NewInstallationDeletionSupervisor("instanceID", time.Nanosecond, 10, sqlStore, &mockEventsProducer{}, logger)

		installation := &model.Installation{
			OwnerID:  "blah",
			Version:  "version",
			Name:     "dns",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			State:    model.InstallationStateDeletionPending,
		}

		err := sqlStore.CreateInstallation(installation, nil, testutil.DNSForInstallation("dns.example.com"))
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		installation.DeletionPendingExpiry = model.GetMillisAtTime(time.Now().Add(time.Hour))
		err = sqlStore.UpdateInstallation(installation)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		supervisor.Supervise(installation)
		installation, err = sqlStore.GetInstallation(installation.ID, false, false)
		require.NoError(t, err)
		require.Equal(t, model.InstallationStateDeletionPending, installation.State)
	})

	t.Run("deletion pending with expiry, ready for deletion", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		supervisor := supervisor.NewInstallationDeletionSupervisor("instanceID", time.Nanosecond, 10, sqlStore, &mockEventsProducer{}, logger)

		installation := &model.Installation{
			OwnerID:  "blah",
			Version:  "version",
			Name:     "dns",
			Size:     mmv1alpha1.Size100String,
			Affinity: model.InstallationAffinityIsolated,
			State:    model.InstallationStateDeletionPending,
		}

		err := sqlStore.CreateInstallation(installation, nil, testutil.DNSForInstallation("dns.example.com"))
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		installation.DeletionPendingExpiry = model.GetMillis() - 1
		err = sqlStore.UpdateInstallation(installation)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		supervisor.Supervise(installation)
		installation, err = sqlStore.GetInstallation(installation.ID, false, false)
		require.NoError(t, err)
		require.Equal(t, model.InstallationStateDeletionRequested, installation.State)
	})

	t.Run("scheduled deletion, before scheduled deletion time", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		supervisor := supervisor.NewInstallationDeletionSupervisor("instanceID", time.Nanosecond, 10, sqlStore, &mockEventsProducer{}, logger)

		installation := &model.Installation{
			OwnerID:               "blah",
			Version:               "version",
			Name:                  "dns",
			Size:                  mmv1alpha1.Size100String,
			Affinity:              model.InstallationAffinityIsolated,
			State:                 model.InstallationStateStable,
			ScheduledDeletionTime: model.GetMillisAtTime(time.Now().Add(time.Hour)),
		}

		err := sqlStore.CreateInstallation(installation, nil, testutil.DNSForInstallation("dns.example.com"))
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		supervisor.Supervise(installation)
		installation, err = sqlStore.GetInstallation(installation.ID, false, false)
		require.NoError(t, err)
		require.Equal(t, model.InstallationStateStable, installation.State)
	})

	t.Run("scheduled deletion, past scheduled deletion time", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		supervisor := supervisor.NewInstallationDeletionSupervisor("instanceID", time.Nanosecond, 10, sqlStore, &mockEventsProducer{}, logger)

		installation := &model.Installation{
			OwnerID:               "blah",
			Version:               "version",
			Name:                  "dns",
			Size:                  mmv1alpha1.Size100String,
			Affinity:              model.InstallationAffinityIsolated,
			State:                 model.InstallationStateStable,
			ScheduledDeletionTime: model.GetMillis() - 1,
		}

		err := sqlStore.CreateInstallation(installation, nil, testutil.DNSForInstallation("dns.example.com"))
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		supervisor.Supervise(installation)
		installation, err = sqlStore.GetInstallation(installation.ID, false, false)
		require.NoError(t, err)
		require.Equal(t, model.InstallationStateDeletionPendingRequested, installation.State)
	})
}
