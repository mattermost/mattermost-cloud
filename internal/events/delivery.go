// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package events

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	contentTypeApplicationJSON = "application/json"

	workerIdleDelay = 6 * time.Second

	retryDelay = 20 * time.Second
)

type delivererStore interface {
	ClaimUpToDateSubscription(instanceID string) (*model.Subscription, error)
	ClaimRetryingSubscription(instanceID string, cooldown time.Duration) (*model.Subscription, error)
	UpdateSubscriptionStatus(sub *model.Subscription) error
	UnlockSubscription(subID, lockerID string, force bool) (bool, error)
	CountSubscriptionsForEvent(eventType model.EventType) (int64, error)
	GetStateChangeEventsToProcess(subID string) ([]*model.StateChangeEventDeliveryData, error)
	UpdateEventDeliveryStatus(delivery *model.EventDelivery) error
}

// EventDeliverer is responsible for delivering events.
type EventDeliverer struct {
	ctx        context.Context
	store      delivererStore
	client     *http.Client
	instanceID string
	config     DelivererConfig
	logger     logrus.FieldLogger
	// TODO: Make retry delay exponential. This will require storing the last delay on Subscription.
	// For now we just use const.
	retryDelay time.Duration
}

// DelivererConfig is config of EventDeliverer component.
type DelivererConfig struct {
	RetryWorkers    int
	UpToDateWorkers int
	MaxBurstWorkers int
}

// NewDeliverer creates new EventDeliverer component.
func NewDeliverer(ctx context.Context, store delivererStore, instanceID string, logger logrus.FieldLogger, cfg DelivererConfig) *EventDeliverer {
	delivery := &EventDeliverer{
		ctx:        ctx,
		store:      store,
		client:     &http.Client{Timeout: 15 * time.Second},
		instanceID: instanceID,
		config:     cfg,
		logger:     logger.WithField("component", "eventsDelivery"),
		retryDelay: retryDelay,
	}

	for i := 0; i < delivery.config.RetryWorkers; i++ {
		go delivery.newWorker().ProcessRetrying(ctx)
	}

	for i := 0; i < delivery.config.UpToDateWorkers; i++ {
		go delivery.newWorker().ProcessUpToDate(ctx)
	}

	return delivery
}

type token struct{}

// SignalNewEvents attempts to try/retry to send to all subscriptions for particular type of event.
// It will throttle concurrent deliveries.
func (d *EventDeliverer) SignalNewEvents(eventType model.EventType) {
	if d.config.MaxBurstWorkers == 0 {
		return
	}

	burst, err := d.store.CountSubscriptionsForEvent(eventType)
	if err != nil {
		d.logger.WithField("eventType", eventType).WithError(err).Error("Failed to count subscriptions for event")
		burst = int64(d.config.MaxBurstWorkers)
	}

	// This will attempt to send to at most MaxBurstWorkers at a time
	// until all subscriptions subscribed to the event where attempted
	// or a worker did not find any subscription to process.
	semaphore := make(chan token, d.config.MaxBurstWorkers)
	done := make(chan struct{}, 1)
	wg := &sync.WaitGroup{}

loop:
	for i := int64(0); i < burst; i++ {
		// In case any of burst workers did not find any more work stop early.
		select {
		case <-done:
			d.logger.Debug("No more subscriptions to process, stopping burst early")
			break loop
		default:
			semaphore <- token{}
		}

		wg.Add(1)
		go func() {
			defer func() {
				<-semaphore
				wg.Done()
			}()

			if !d.newWorker().ProcessUpToDateOnce() {
				select {
				case done <- struct{}{}:
				default:
					return
				}
			}
		}()
	}
	close(semaphore)
	wg.Wait()
	close(done)
}

// sender is a helper struct for separation of logic.
// It uses HTTP client of EventDeliverer.
type sender struct {
	store      delivererStore
	client     *http.Client
	instanceID string
	logger     logrus.FieldLogger
}

func (d *EventDeliverer) newWorker() *sender {
	return &sender{
		store:      d.store,
		client:     d.client,
		instanceID: d.instanceID,
		logger:     d.logger.WithField("worker", model.NewID()),
	}
}

// ProcessUpToDateOnce attempts to process single up-to-date subscription once.
func (s *sender) ProcessUpToDateOnce() bool {
	return s.processSubscriptionEvents(s.store.ClaimUpToDateSubscription)
}

// ProcessUpToDate starts a worker processing up-to-date subscriptions.
func (s *sender) ProcessUpToDate(ctx context.Context) {
	s.process(ctx, s.store.ClaimUpToDateSubscription)
}

// ProcessRetrying starts a worker processing retrying subscriptions.
func (s *sender) ProcessRetrying(ctx context.Context) {
	s.process(ctx, func(instanceID string) (*model.Subscription, error) {
		return s.store.ClaimRetryingSubscription(instanceID, retryDelay)
	})
}

func (s *sender) process(ctx context.Context, claimFunc func(instanceID string) (*model.Subscription, error)) {
	s.logger.Info("Worker is starting processing")

	subscriptionProcessed := true
	for {
		select {
		case <-ctx.Done():
		default:
			// If last time we did not get subscription to process, wait before trying again.
			if !subscriptionProcessed {
				time.Sleep(workerIdleDelay)
			}
			subscriptionProcessed = s.processSubscriptionEvents(claimFunc)
		}
	}
}

// processSubscriptionEvents attempts to claim subscription and process its events.
// Returns true if the subscription was claimed, false if there are no more
// subscriptions awaiting delivery or failed to claim.
func (s *sender) processSubscriptionEvents(claimFunc func(instanceID string) (*model.Subscription, error)) bool {
	subscription, err := claimFunc(s.instanceID)
	if err != nil {
		if err == store.ErrNoSubscriptionsToProcess {
			return false
		}
		s.logger.WithError(err).Error("Failed to claim subscription to process")
		return false
	}
	log := s.logger.WithField("subscription", subscription.ID)
	defer s.unlockSubscription(subscription.ID, log)

	s.logger.Debug("Processing events delivery for subscription")

	// If amount of events will grow substantially we might consider
	// fetching in batches. For now fetching all should be good.
	eventDeliveries, err := s.store.GetStateChangeEventsToProcess(subscription.ID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get events to process")
		return true
	}

	s.processDeliveries(subscription, eventDeliveries, log)
	return true
}

func (s *sender) processDeliveries(sub *model.Subscription, deliveries []*model.StateChangeEventDeliveryData, log logrus.FieldLogger) {
	var subDeliveryStatus model.SubscriptionDeliveryStatus
	var continueSending bool
	for _, d := range deliveries {
		subDeliveryStatus, continueSending = s.processDelivery(sub, d, log)
		if !continueSending {
			break
		}
	}
	sub.LastDeliveryStatus = subDeliveryStatus
	sub.LastDeliveryAttemptAt = model.GetMillis()

	err := s.store.UpdateSubscriptionStatus(sub)
	if err != nil {
		s.logger.WithError(err).Error("Failed to update subscription status after delivery")
		return
	}
}

func (s *sender) processDelivery(sub *model.Subscription, delivery *model.StateChangeEventDeliveryData, logger logrus.FieldLogger) (model.SubscriptionDeliveryStatus, bool) {
	delivery.EventDelivery.Attempts++
	delivery.EventDelivery.LastAttempt = model.GetMillis()

	log := logger.WithFields(map[string]interface{}{
		"event":         delivery.EventData.Event.ID,
		"eventDelivery": delivery.EventDelivery.ID,
	})

	s.logger.Debug("Attempting to deliver event...")

	var subDeliveryStatus model.SubscriptionDeliveryStatus

	delivery.EventHeaders = sub.Headers
	err := s.sendEvent(sub.ID, sub.URL, delivery, log)
	if err != nil {
		s.logger.WithError(err).Error("Failed to deliver event")

		// We abort delivery on the subscription only if the event will be retried
		// otherwise we mark event as failed and continue.
		if delivery.EventData.Event.Timestamp+sub.FailureThreshold.Milliseconds() < delivery.EventDelivery.LastAttempt {
			delivery.EventDelivery.Status = model.EventDeliveryFailed
		} else {
			subDeliveryStatus = model.SubscriptionDeliveryFailed
			delivery.EventDelivery.Status = model.EventDeliveryRetrying
		}
	} else {
		delivery.EventDelivery.Status = model.EventDeliveryDelivered
	}

	err = s.store.UpdateEventDeliveryStatus(&delivery.EventDelivery)
	if err != nil {
		s.logger.WithError(err).Error("Failed to update event delivery state")
		return subDeliveryStatus, false
	}

	return subDeliveryStatus, subDeliveryStatus != model.SubscriptionDeliveryFailed
}

func (s *sender) sendEvent(subscription_id, url string, data *model.StateChangeEventDeliveryData, log logrus.FieldLogger) error {
	payload, err := json.Marshal(data.EventData.ToEventPayload())
	if err != nil {
		return errors.Wrap(err, "failed to marshal event payload")
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		s.logger.WithField("subscription_url", url).WithError(err).Error("Unable to create request")
		return errors.Wrap(err, "unable to create request from payload")
	}
	req.Header.Set("Content-Type", contentTypeApplicationJSON)
	headers, err := model.ParseHeadersFromStringMap(data.EventHeaders)
	if err != nil {
		// If there's an error parsing the headers, log it but continue execution so the subscription
		// event is sent, `model.ParseHeadersFromStringMap` should take care of not disclosing any
		// unset environment variables into resulting headers.
		s.logger.WithFields(logrus.Fields{
			"subscription": subscription_id,
		}).WithError(err).Error()
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to deliver event")
	}
	defer drainBody(resp.Body)
	if resp.StatusCode >= 500 {
		return errors.Errorf(
			"consumer failed to receive event, got %d response code, body: %s",
			resp.StatusCode, attemptToReadBody(resp.Body),
		)
	}
	if resp.StatusCode != http.StatusOK {
		s.logger.Errorf("Event delivery resulted in status %d. Treating it as consumer error, delivery will not be retried",
			resp.StatusCode)
	}

	return nil
}

func (s *sender) unlockSubscription(subID string, log logrus.FieldLogger) {
	unlocked, err := s.store.UnlockSubscription(subID, s.instanceID, false)
	if err != nil {
		s.logger.WithError(err).Error("failed to unlock subscription")
	} else if !unlocked {
		s.logger.Error("failed to release lock for subscription")
	}
}

func attemptToReadBody(reader io.Reader) string {
	body, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Sprintf("failed to read body: %s", err.Error())
	}
	return string(body)
}

func drainBody(readCloser io.ReadCloser) {
	_, _ = io.ReadAll(readCloser)
	_ = readCloser.Close()
}
