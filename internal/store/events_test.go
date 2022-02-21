// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ClaimSubscriptionsAndGetDeliveries(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)
	instanceID := model.NewID()

	sub1 := &model.Subscription{
		URL:                   "test1",
		EventType:             model.ResourceStateChangeEventType,
		LastDeliveryStatus:    model.SubscriptionDeliverySucceeded,
		LastDeliveryAttemptAt: 100,
	}
	sub2 := &model.Subscription{
		URL:                   "test2",
		EventType:             model.ResourceStateChangeEventType,
		LastDeliveryStatus:    model.SubscriptionDeliverySucceeded,
		LastDeliveryAttemptAt: 101,
	}
	sub3 := &model.Subscription{
		URL:                   "test3",
		EventType:             "unknown event type",
		LastDeliveryStatus:    model.SubscriptionDeliverySucceeded,
		LastDeliveryAttemptAt: 103,
	}
	sub4 := &model.Subscription{
		URL:                   "test4",
		EventType:             model.ResourceStateChangeEventType,
		LastDeliveryStatus:    model.SubscriptionDeliverySucceeded,
		LastDeliveryAttemptAt: 104,
	}

	err := sqlStore.CreateSubscription(sub1)
	require.NoError(t, err)
	err = sqlStore.CreateSubscription(sub2)
	require.NoError(t, err)
	err = sqlStore.CreateSubscription(sub3)
	require.NoError(t, err)
	err = sqlStore.CreateSubscription(sub4)
	require.NoError(t, err)

	// Deleted subscription should not be picked.
	err = sqlStore.DeleteSubscription(sub4.ID)
	require.NoError(t, err)

	eventData1 := &model.StateChangeEventData{
		Event: model.Event{
			EventType: model.ResourceStateChangeEventType,
			Timestamp: model.GetMillis(),
		},
		StateChange: model.StateChangeEvent{
			OldState:     "old",
			NewState:     "new",
			ResourceID:   "installation1",
			ResourceType: "installation",
		},
	}
	time.Sleep(1 * time.Millisecond) // Make sure timestamps differ
	eventData2 := &model.StateChangeEventData{
		Event: model.Event{
			EventType: model.ResourceStateChangeEventType,
			Timestamp: model.GetMillis(),
		},
		StateChange: model.StateChangeEvent{
			OldState:     "old",
			NewState:     "new",
			ResourceID:   "installation1",
			ResourceType: "installation",
		},
	}
	err = sqlStore.CreateStateChangeEvent(eventData1)
	require.NoError(t, err)
	err = sqlStore.CreateStateChangeEvent(eventData2)
	require.NoError(t, err)

	// Claim first subscription
	subscription, err := sqlStore.ClaimUpToDateSubscription(instanceID)
	require.NoError(t, err)
	assert.Equal(t, sub1.ID, subscription.ID)

	subscription, err = sqlStore.GetSubscription(subscription.ID)
	require.NoError(t, err)
	require.NotNil(t, subscription)
	assert.True(t, subscription.LockAcquiredAt > 0)
	assert.Equal(t, instanceID, *subscription.LockAcquiredBy)

	// Claim second subscription
	subscription2, err := sqlStore.ClaimUpToDateSubscription(instanceID)
	require.NoError(t, err)
	assert.Equal(t, sub2.ID, subscription2.ID)

	subscription2, err = sqlStore.GetSubscription(subscription2.ID)
	require.NoError(t, err)
	require.NotNil(t, subscription2)
	assert.True(t, subscription2.LockAcquiredAt > 0)
	assert.Equal(t, instanceID, *subscription2.LockAcquiredBy)

	// Do not claim any if all are already claimed
	_, err = sqlStore.ClaimUpToDateSubscription(instanceID)
	require.Error(t, err)
	assert.Equal(t, ErrNoSubscriptionsToProcess, err)

	// Mark event deliveries for firs subscription as success and release lock
	deliveries, err := sqlStore.GetStateChangeEventsToProcess(sub1.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, len(deliveries))

	// Assert event IDs are set properly and events order is correct
	assert.Equal(t, eventData1.Event.ID, deliveries[0].EventData.Event.ID)
	assert.Equal(t, eventData1.StateChange.ID, deliveries[0].EventData.StateChange.ID)
	assert.NotEqual(t, deliveries[0].EventData.Event.ID, deliveries[0].EventData.StateChange.ID)
	assert.True(t, deliveries[0].EventData.Event.Timestamp < deliveries[1].EventData.Event.Timestamp)

	deliveries[0].EventDelivery.LastAttempt = model.GetMillis()
	deliveries[0].EventDelivery.Attempts = 1
	deliveries[0].EventDelivery.Status = model.EventDeliveryDelivered
	deliveries[1].EventDelivery.Status = model.EventDeliveryRetrying

	err = sqlStore.UpdateEventDeliveryStatus(&deliveries[0].EventDelivery)
	require.NoError(t, err)
	err = sqlStore.UpdateEventDeliveryStatus(&deliveries[1].EventDelivery)
	require.NoError(t, err)

	sub1.LastDeliveryStatus = model.SubscriptionDeliveryFailed
	err = sqlStore.UpdateSubscriptionStatus(sub1)
	require.NoError(t, err)

	unlocked, err := sqlStore.UnlockSubscription(sub1.ID, instanceID, false)
	require.NoError(t, err)
	assert.True(t, unlocked)

	// Claim first subscription again and get retrying delivery to process
	sub1, err = sqlStore.ClaimRetryingSubscription(instanceID, 10)
	require.NoError(t, err)
	assert.Equal(t, subscription.ID, sub1.ID)

	deliveries, err = sqlStore.GetStateChangeEventsToProcess(sub1.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, len(deliveries))

	deliveries[0].EventDelivery.Status = model.EventDeliveryDelivered
	err = sqlStore.UpdateEventDeliveryStatus(&deliveries[0].EventDelivery)
	require.NoError(t, err)

	unlocked, err = sqlStore.UnlockSubscription(sub1.ID, instanceID, false)
	require.NoError(t, err)
	assert.True(t, unlocked)

	// Claim no subscription as no events to process
	_, err = sqlStore.ClaimUpToDateSubscription(instanceID)
	require.Error(t, err)
	assert.Equal(t, ErrNoSubscriptionsToProcess, err)
	_, err = sqlStore.ClaimRetryingSubscription(instanceID, 10)
	require.Error(t, err)
	assert.Equal(t, ErrNoSubscriptionsToProcess, err)
}

func TestGetStateChangeEvent(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	eventData1 := &model.StateChangeEventData{
		Event: model.Event{
			EventType: model.ResourceStateChangeEventType,
			Timestamp: model.GetMillis(),
			ExtraData: model.EventExtraData{Fields: map[string]string{
				"key1": "val1",
				"key2": "val2",
			}},
		},
		StateChange: model.StateChangeEvent{
			OldState:     "old",
			NewState:     "new",
			ResourceID:   "installation1",
			ResourceType: "installation",
		},
	}
	err := sqlStore.CreateStateChangeEvent(eventData1)
	require.NoError(t, err)

	event, err := sqlStore.GetStateChangeEvent(eventData1.Event.ID)
	require.NoError(t, err)

	assert.Equal(t, eventData1.Event.ID, event.Event.ID)
	assert.Equal(t, eventData1.Event.EventType, event.Event.EventType)
	assert.Equal(t, eventData1.Event.Timestamp, event.Event.Timestamp)
	assert.Equal(t, eventData1.Event.ExtraData, event.Event.ExtraData)
	assert.Equal(t, eventData1.StateChange.ID, event.StateChange.ID)
	assert.Equal(t, eventData1.StateChange.EventID, event.StateChange.EventID)
	assert.Equal(t, eventData1.StateChange.OldState, event.StateChange.OldState)
	assert.Equal(t, eventData1.StateChange.NewState, event.StateChange.NewState)
	assert.Equal(t, eventData1.StateChange.ResourceID, event.StateChange.ResourceID)
	assert.Equal(t, eventData1.StateChange.ResourceType, event.StateChange.ResourceType)

	t.Run("should not found", func(t *testing.T) {
		event, err := sqlStore.GetStateChangeEvent(model.NewID())
		require.NoError(t, err)
		assert.Nil(t, event)
	})
}

func TestGetStateChangeEvents(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	events := []*model.StateChangeEventData{
		{
			Event: model.Event{EventType: "test", Timestamp: 1000},
			StateChange: model.StateChangeEvent{ResourceType: model.TypeInstallation, ResourceID: "inst1",
				OldState: model.InstallationStateHibernationInProgress, NewState: model.InstallationStateHibernating},
		},
		{
			Event: model.Event{EventType: "test", Timestamp: 2000},
			StateChange: model.StateChangeEvent{ResourceType: model.TypeInstallation, ResourceID: "inst2",
				OldState: model.InstallationStateDeletionInProgress, NewState: model.InstallationStateDeleted},
		},
		{
			Event: model.Event{EventType: "test", Timestamp: 3000},
			StateChange: model.StateChangeEvent{ResourceType: model.TypeCluster, ResourceID: "cluster1",
				OldState: model.ClusterStateCreationRequested, NewState: model.ClusterStateStable},
		},
	}

	for i := range events {
		err := sqlStore.CreateStateChangeEvent(events[i])
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond)
	}

	for _, testCase := range []struct {
		description string
		filter      *model.StateChangeEventFilter
		fetched     int
	}{
		{
			description: "fetch all for installations",
			filter:      &model.StateChangeEventFilter{ResourceType: model.TypeInstallation, Paging: model.AllPagesNotDeleted()},
			fetched:     2,
		},
		{
			description: "fetch all for specific ID",
			filter:      &model.StateChangeEventFilter{ResourceID: "inst2", Paging: model.AllPagesNotDeleted()},
			fetched:     1,
		},
		{
			description: "fetch all installations with old state of deletion-in-progress",
			filter:      &model.StateChangeEventFilter{OldStates: []string{model.InstallationStateDeletionInProgress}, Paging: model.AllPagesNotDeleted()},
			fetched:     1,
		},
		{
			description: "fetch all installations with new state of deletion-in-progress",
			filter:      &model.StateChangeEventFilter{NewStates: []string{model.InstallationStateDeletionInProgress}, Paging: model.AllPagesNotDeleted()},
			fetched:     0,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			fetchedSubs, err := sqlStore.GetStateChangeEvents(testCase.filter)
			require.NoError(t, err)
			assert.Equal(t, testCase.fetched, len(fetchedSubs))
		})
	}
}
