// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//+build e2e

package eventstest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type RecordType int

const (
	RecordInstallation RecordType = 1 << iota
	RecordCluster
	RecordClusterInstallation
)

const (
	RecordAll RecordType = RecordInstallation | RecordCluster | RecordClusterInstallation
)

type EventsRecorder struct {
	client          *model.Client
	owner           string
	listenerAddress string
	logger          log.FieldLogger

	server       *http.Server
	subscription *model.Subscription

	recordResources RecordType

	sync.Mutex
	RecordedEvents []model.StateChangeEventPayload
}

func NewEventsRecorder(owner, listenAddr string, logger log.FieldLogger, recordRes RecordType) *EventsRecorder {
	return &EventsRecorder{
		owner:           owner,
		listenerAddress: listenAddr,
		logger:          logger,
		recordResources: recordRes,
		Mutex:           sync.Mutex{},
		RecordedEvents:  []model.StateChangeEventPayload{},
	}
}

func (ev *EventsRecorder) Start(client *model.Client, logger log.FieldLogger) error {
	err := ev.runServer()
	if err != nil {
		return err
	}

	sub, err := RegisterStateChangeSubscription(client, ev.owner, ev.listenerAddress, logger)
	if err != nil {
		return errors.Wrap(err, "failed to register subscription")
	}
	ev.subscription = sub
	ev.logger.WithField("subscriptionID", sub.ID).Info("Created subscription")

	return nil
}

func (ev *EventsRecorder) runServer() error {
	eventsURL, err := url.Parse(ev.listenerAddress)
	if err != nil {
		return errors.Wrap(err, "failed to parse listener address")
	}

	handler := http.NewServeMux()
	handler.HandleFunc("/", ev.eventsHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", eventsURL.Port()),
		Handler: handler,
	}
	ev.server = server

	go func() {
		ev.logger.WithField("port", eventsURL.Port()).Info("Starting events listener...")
		err := ev.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.WithError(err).Errorf("Error listening")
			return
		}
	}()
	return nil
}

func (ev *EventsRecorder) ShutDown(client *model.Client) {
	err := ev.server.Close()
	if err != nil {
		ev.logger.WithError(err).Error("Error while closing events server")
	}

	err = client.DeleteSubscription(ev.subscription.ID)
	if err != nil {
		ev.logger.WithError(err).Error("Error while deleting subscription")
	}

	return
}

func (ev *EventsRecorder) eventsHandler(w http.ResponseWriter, r *http.Request) {
	payload := model.StateChangeEventPayload{}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		ev.logger.WithError(err).Error("Failed to decode event payload")
		return
	}

	if maskType(payload.ResourceType)&ev.recordResources == 0 {
		ev.logger.Debugf("Event ignored: %s, %s", payload.ResourceType, payload.ResourceID)
		w.WriteHeader(http.StatusOK)
		return
	}

	ev.Lock()
	ev.RecordedEvents = append(ev.RecordedEvents, payload)
	ev.Unlock()

	w.WriteHeader(http.StatusOK)
}

func maskType(resourceType model.ResourceType) RecordType {
	switch resourceType {
	case model.TypeInstallation:
		return RecordInstallation
	case model.TypeCluster:
		return RecordCluster
	case model.TypeClusterInstallation:
		return RecordClusterInstallation
	}
	return 0
}

type EventOccurrence struct {
	ResourceType string
	ResourceID   string
	OldState     string
	NewState     string
}

func (eo EventOccurrence) MatchPayload(payload model.StateChangeEventPayload) bool {
	return eo.ResourceID == payload.ResourceID &&
		eo.ResourceType == payload.ResourceType.String() &&
		eo.NewState == payload.NewState &&
		eo.OldState == payload.OldState
}

func (ev *EventsRecorder) VerifyInOrder(expectedEvents []EventOccurrence) error {
	expEvLen := len(expectedEvents)

	// This might be a little hard to troubleshoot in case of error,
	// so we leave here some Debug logs for such scenarios.
	for i, e := range ev.RecordedEvents {
		ev.logger.WithField("event", e).Debugf("Recorderd event %d", i)
		if len(expectedEvents) == 0 {
			break
		}
		expected := expectedEvents[0]

		// The order of some events might be different depending on which
		// supervisor run earlier therefore we check first 2 events.
		if expected.MatchPayload(e) {
			expectedEvents = expectedEvents[1:]
		} else if len(expectedEvents) > 1 {
			expected = expectedEvents[1]
			if expected.MatchPayload(e) {
				expectedEvents = append(expectedEvents[:1], expectedEvents[2:]...)
			}
		}
	}

	if len(expectedEvents) > 0 {
		return fmt.Errorf("not all expected events appeared in correct order, "+
			"%d expected events occured, "+
			"stuck at expected event: %v", expEvLen-len(expectedEvents), expectedEvents[0])
	}

	return nil
}

func RegisterStateChangeSubscription(client *model.Client, owner, subURL string, logger log.FieldLogger) (*model.Subscription, error) {
	subs, err := client.ListSubscriptions(&model.ListSubscriptionsRequest{
		Paging:    model.AllPagesNotDeleted(),
		Owner:     owner,
	})
	if err != nil {
		logger.WithError(err).Error("Failed to list e2e subscriptions")
		// We do not fail, we will try to register
	}
	for _, sub := range subs {
		if sub.URL == subURL {
			logger.Infof("Found existing subscription %q", sub.ID)
			return sub, nil
		}
	}
	logger.Infof("Subscription not found, registering...")

	createSubReq := model.CreateSubscriptionRequest{
		Name:             "E2e Event Listener",
		URL:              subURL,
		OwnerID:          owner,
		EventType:        model.ResourceStateChangeEventType,
		FailureThreshold: 30 * time.Second, // Should guarantee one retry
	}

	return client.CreateSubscription(&createSubReq)
}
