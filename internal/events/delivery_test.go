// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package events

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDelivery_Workers(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	instanceID := model.NewID()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := DelivererConfig{
		UpToDateWorkers: 4,
	}
	_ = NewDeliverer(ctx, sqlStore, instanceID, logger, cfg)

	subscriptions := 10
	deliveredEvents := make(chan *model.StateChangeEventPayload, subscriptions)

	r := mux.NewRouter()
	r.HandleFunc("/event", successEventHandler(t, deliveredEvents))
	consumerSever := httptest.NewServer(r)

	createFixedSubscriptions(t, sqlStore, subscriptions, consumerSever.URL)
	eventData := createFixedEventData(t, sqlStore)

	awaitEvents(t, deliveredEvents, eventData.Event.ID, subscriptions)
}

func TestDelivery_WorkersRetrying(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	instanceID := model.NewID()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := DelivererConfig{
		RetryWorkers:    4,
		MaxBurstWorkers: 50,
	}
	eventsDeliverer := NewDeliverer(ctx, sqlStore, instanceID, logger, cfg)
	// Adjust for the purpose of tests.
	eventsDeliverer.retryDelay = 5 * time.Second

	subscriptions := 10
	deliveredEvents := make(chan *model.StateChangeEventPayload, subscriptions)

	r := mux.NewRouter()
	r.HandleFunc("/event", failureEventHandler(t, deliveredEvents))
	consumerSever := httptest.NewServer(r)

	createFixedSubscriptions(t, sqlStore, subscriptions, consumerSever.URL)
	eventData := createFixedEventData(t, sqlStore)

	// Fail delivery
	eventsDeliverer.SignalNewEvents(model.ResourceStateChangeEventType)

	awaitEvents(t, deliveredEvents, eventData.Event.ID, subscriptions)

	// Events should be retried after a couple of seconds
	awaitEvents(t, deliveredEvents, eventData.Event.ID, subscriptions)
}

func TestDelivery_SignalNewEvents(t *testing.T) {

	for _, testCase := range []struct {
		description   string
		subscriptions int
	}{
		{
			description:   "300 subscriptions",
			subscriptions: 300,
		},
		{
			description:   "101 subscriptions",
			subscriptions: 101,
		},
		{
			description:   "5 subscriptions",
			subscriptions: 5,
		},
		{
			description:   "123 subscriptions",
			subscriptions: 123,
		},
		{
			description:   "0 subscriptions",
			subscriptions: 0,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			logger := testlib.MakeLogger(t)
			sqlStore := store.MakeTestSQLStore(t, logger)
			defer store.CloseConnection(t, sqlStore)

			instanceID := model.NewID()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			cfg := DelivererConfig{
				MaxBurstWorkers: 50,
			}
			eventsDeliverer := NewDeliverer(ctx, sqlStore, instanceID, logger, cfg)

			deliveredEvents := make(chan *model.StateChangeEventPayload, testCase.subscriptions)

			r := mux.NewRouter()
			r.HandleFunc("/event", successEventHandler(t, deliveredEvents))
			consumerSever := httptest.NewServer(r)

			createFixedSubscriptions(t, sqlStore, testCase.subscriptions, consumerSever.URL)
			eventData := createFixedEventData(t, sqlStore)

			eventsDeliverer.SignalNewEvents(model.ResourceStateChangeEventType)

			awaitEvents(t, deliveredEvents, eventData.Event.ID, testCase.subscriptions)
		})
	}
}

func TestDelivery_GiveUpWhenThresholdReached(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	instanceID := model.NewID()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := DelivererConfig{
		RetryWorkers:    4,
		MaxBurstWorkers: 50,
	}
	eventsDeliverer := NewDeliverer(ctx, sqlStore, instanceID, logger, cfg)

	deliveredEvents := make(chan *model.StateChangeEventPayload, 1)

	r := mux.NewRouter()
	r.HandleFunc("/event", failureEventHandler(t, deliveredEvents))
	consumerSever := httptest.NewServer(r)

	subscription := &model.Subscription{
		URL:                fmt.Sprintf("%s/event", consumerSever.URL),
		EventType:          model.ResourceStateChangeEventType,
		LastDeliveryStatus: model.SubscriptionDeliveryNone,
		FailureThreshold:   0, // This will guarantee only one try
	}
	err := sqlStore.CreateSubscription(subscription)
	require.NoError(t, err)

	eventData := createFixedEventData(t, sqlStore)

	// Make sure last delivery attempt is not equal event timestamp.
	time.Sleep(1 * time.Millisecond)

	// Init and wait for delivery
	eventsDeliverer.SignalNewEvents(model.ResourceStateChangeEventType)
	<-deliveredEvents

	// Assert that EventDelivery was marked as failed.
	deliveries, err := sqlStore.GetDeliveriesForSubscription(subscription.ID)
	require.NoError(t, err)
	assert.Len(t, deliveries, 1, "expected one delivery")
	assert.Equal(t, eventData.Event.ID, deliveries[0].EventID)
	assert.Equal(t, model.EventDeliveryFailed, deliveries[0].Status)
}

func successEventHandler(t *testing.T, deliveryChan chan<- *model.StateChangeEventPayload) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := model.NewStateChangeEventPayloadFromReader(r.Body)
		require.NoError(t, err)
		deliveryChan <- payload
	}
}

func failureEventHandler(t *testing.T, deliveryChan chan<- *model.StateChangeEventPayload) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := model.NewStateChangeEventPayloadFromReader(r.Body)
		require.NoError(t, err)
		deliveryChan <- payload
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func awaitEvents(t *testing.T, deliveryChan <-chan *model.StateChangeEventPayload, expectedID string, eventsCount int) {
	timeout := time.After(25 * time.Second)
	delivered := 0

	for delivered < eventsCount {
		select {
		case event := <-deliveryChan:
			delivered++
			assert.Equal(t, expectedID, event.EventID)
		case <-timeout:
			t.Fatalf("timeout while waiting for events, delivered: %d", delivered)
		}
	}
}

func createFixedSubscriptions(t *testing.T, sqlStore *store.SQLStore, subsCount int, subBaseURL string) {
	for i := 0; i < subsCount; i++ {
		subscription := &model.Subscription{
			URL:                fmt.Sprintf("%s/event", subBaseURL),
			EventType:          model.ResourceStateChangeEventType,
			LastDeliveryStatus: model.SubscriptionDeliveryNone,
			FailureThreshold:   1 * time.Minute,
		}
		err := sqlStore.CreateSubscription(subscription)
		require.NoError(t, err)
	}
}

func createFixedEventData(t *testing.T, sqlStore *store.SQLStore) model.StateChangeEventData {
	eventData := model.StateChangeEventData{
		Event: model.Event{
			EventType: model.ResourceStateChangeEventType,
			Timestamp: model.GetMillis(),
		},
		StateChange: model.StateChangeEvent{
			OldState:     "old",
			NewState:     "new",
			ResourceID:   "abcd",
			ResourceType: "installation",
		},
	}

	err := sqlStore.CreateStateChangeEvent(&eventData)
	require.NoError(t, err)

	return eventData
}
