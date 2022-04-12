// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package events

import (
	"time"

	"github.com/mattermost/mattermost-cloud/internal/webhook"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type producerStore interface {
	CreateStateChangeEvent(event *model.StateChangeEventData) error
	GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error)
}

type deliverySignaler interface {
	SignalNewEvents(eventType model.EventType)
}

// DataField represents strings key value pair used for events extra data.
type DataField struct {
	Key   string
	Value string
}

// EventProducer produces Provisioners' state change events.
type EventProducer struct {
	store       producerStore
	signaler    deliverySignaler
	environment string
	logger      logrus.FieldLogger
}

// TODO: ideally we should produce state change events in the same transaction that is updating the state.
// Given that this would require huge refactor, we do not do it for the initial implementation.

// NewProducer creates new EventProducer.
func NewProducer(store producerStore, signaler deliverySignaler, env string, log logrus.FieldLogger) *EventProducer {
	return &EventProducer{
		store:       store,
		signaler:    signaler,
		environment: env,
		logger:      log.WithField("component", "eventsProducer"),
	}
}

// ProduceInstallationStateChangeEvent produces state change event for a given Installation.
func (e *EventProducer) ProduceInstallationStateChangeEvent(installation *model.Installation, oldState string, extraDataFields ...DataField) error {
	stateChangeEvent := model.StateChangeEvent{
		OldState:     oldState,
		NewState:     installation.State,
		ResourceID:   installation.ID,
		ResourceType: model.TypeInstallation,
	}

	extraData := e.initExtraData(extraDataFields)

	// TODO: we can still pass the DNS, tho we would require passing DNS zone
	// to EventProducer.
	//extraData["DNS"] = installation.DNS
	extraData["Name"] = installation.Name

	return e.produceStateChangeEvent(stateChangeEvent, extraData)
}

// ProduceClusterStateChangeEvent produces state change event for a given Cluster.
func (e *EventProducer) ProduceClusterStateChangeEvent(cluster *model.Cluster, oldState string, extraDataFields ...DataField) error {
	stateChangeEvent := model.StateChangeEvent{
		OldState:     oldState,
		NewState:     cluster.State,
		ResourceID:   cluster.ID,
		ResourceType: model.TypeCluster,
	}

	extraData := e.initExtraData(extraDataFields)

	return e.produceStateChangeEvent(stateChangeEvent, extraData)
}

// ProduceClusterInstallationStateChangeEvent produces state change event for a given ClusterInstallation.
func (e *EventProducer) ProduceClusterInstallationStateChangeEvent(clusterInstallation *model.ClusterInstallation, oldState string, extraDataFields ...DataField) error {
	stateChangeEvent := model.StateChangeEvent{
		OldState:     oldState,
		NewState:     clusterInstallation.State,
		ResourceID:   clusterInstallation.ID,
		ResourceType: model.TypeClusterInstallation,
	}

	extraData := e.initExtraData(extraDataFields)
	extraData["ClusterID"] = clusterInstallation.ClusterID

	return e.produceStateChangeEvent(stateChangeEvent, extraData)
}

func (e *EventProducer) produceStateChangeEvent(stateChangeEvent model.StateChangeEvent, extraData map[string]string) error {
	event := model.Event{
		EventType: model.ResourceStateChangeEventType,
		Timestamp: model.GetMillis(),
		ExtraData: model.EventExtraData{Fields: extraData},
	}

	eventData := model.StateChangeEventData{
		Event:       event,
		StateChange: stateChangeEvent,
	}

	err := e.store.CreateStateChangeEvent(&eventData)
	if err != nil {
		return errors.Wrap(err, "failed to create state change event")
	}

	log := e.logger.WithField("event", eventData.Event.ID)

	e.signaler.SignalNewEvents(model.ResourceStateChangeEventType)

	webhookPayload := eventData.ToWebhookPayload()

	e.sendWebhook(&webhookPayload, log)
	return nil
}

func (e *EventProducer) sendWebhook(payload *model.WebhookPayload, log logrus.FieldLogger) {
	// TODO: currently webhooks timestamp is in nano seconds instead of milliseconds as all the other timestamps.
	// We do not want to break compatibility now, but we should align in the future when critical services switch to Events.
	payload.Timestamp = time.Now().UnixNano()

	err := webhook.SendToAllWebhooks(e.store, payload, log.WithField("webhookEvent", payload.NewState))
	if err != nil {
		log.WithError(err).Error("Failed to send webhook")
	}
}

func (e *EventProducer) initExtraData(extraData []DataField) map[string]string {
	base := map[string]string{
		"Environment": e.environment,
	}
	for _, d := range extraData {
		base[d.Key] = d.Value
	}
	return base
}
