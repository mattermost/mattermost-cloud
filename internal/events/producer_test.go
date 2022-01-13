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
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ProduceAndDeliverEvents(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	instanceID := model.NewID()

	installation := &model.Installation{
		DNS:   "test.installation.com",
		State: model.InstallationStateStable,
	}
	err := sqlStore.CreateInstallation(installation, nil)
	require.NoError(t, err)

	eventChan := make(chan *model.StateChangeEventPayload)
	webhookChan := make(chan *model.WebhookPayload)

	r := mux.NewRouter()
	r.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		payload, err := model.WebhookPayloadFromReader(r.Body)
		require.NoError(t, err)
		webhookChan <- payload
	})
	r.HandleFunc("/event", func(w http.ResponseWriter, r *http.Request) {
		payload, err := model.NewStateChangeEventPayloadFromReader(r.Body)
		require.NoError(t, err)
		eventChan <- payload
	})

	consumerSever := httptest.NewServer(r)
	webhook := &model.Webhook{
		URL: fmt.Sprintf("%s/webhook", consumerSever.URL),
	}
	err = sqlStore.CreateWebhook(webhook)
	require.NoError(t, err)

	subscription := &model.Subscription{
		URL:                fmt.Sprintf("%s/event", consumerSever.URL),
		EventType:          model.ResourceStateChangeEventType,
		LastDeliveryStatus: model.SubscriptionDeliveryNone,
	}
	err = sqlStore.CreateSubscription(subscription)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := DelivererConfig{
		MaxBurstWorkers: 5,
	}
	eventsDeliverer := NewDeliverer(ctx, sqlStore, instanceID, logger, cfg)

	eventProducer := NewProducer(sqlStore, eventsDeliverer, "test", logger)

	err = eventProducer.ProduceInstallationStateChangeEvent(installation, model.InstallationStateUpdateInProgress)
	require.NoError(t, err)

	webhookPayload, eventPayload, err := awaitWebhookAndEvent(webhookChan, eventChan)
	require.NoError(t, err)
	assert.Equal(t, eventPayload.EventID, webhookPayload.EventID)

	// Assert Event
	assert.Equal(t, installation.ID, eventPayload.ResourceID)
	assert.Equal(t, model.TypeInstallation, eventPayload.ResourceType)
	assert.Equal(t, model.InstallationStateStable, eventPayload.NewState)
	assert.Equal(t, model.InstallationStateUpdateInProgress, eventPayload.OldState)
	assert.NotEmpty(t, eventPayload.EventID)
	assert.NotEmpty(t, eventPayload.Timestamp)
	assert.Equal(t, "test", eventPayload.ExtraData["Environment"])
	assert.Equal(t, installation.DNS, eventPayload.ExtraData["DNS"])

	// Asset Webhook
	assert.Equal(t, installation.ID, webhookPayload.ID)
	assert.Equal(t, model.TypeInstallation, webhookPayload.Type)
	assert.Equal(t, model.InstallationStateStable, webhookPayload.NewState)
	assert.Equal(t, model.InstallationStateUpdateInProgress, webhookPayload.OldState)
	assert.NotEmpty(t, webhookPayload.EventID)
	assert.NotEmpty(t, webhookPayload.Timestamp)
	assert.Equal(t, "test", webhookPayload.ExtraData["Environment"])
	assert.Equal(t, installation.DNS, webhookPayload.ExtraData["DNS"])
}

func awaitWebhookAndEvent(webhookChan <-chan *model.WebhookPayload, eventChan <-chan *model.StateChangeEventPayload) (*model.WebhookPayload, *model.StateChangeEventPayload, error) {
	gotWebhook := false
	gotEvent := false

	var eventPayload *model.StateChangeEventPayload
	var webhookPayload *model.WebhookPayload

	timeout := time.After(3 * time.Second)

	for !gotWebhook || !gotEvent {
		select {
		case eventPayload = <-eventChan:
			gotEvent = true
		case webhookPayload = <-webhookChan:
			gotWebhook = true

		case <-timeout:
			return nil, nil, errors.New("timeout waiting for event and webhook")
		}
	}
	return webhookPayload, eventPayload, nil
}
