// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
	"time"

	"github.com/pkg/errors"
)

// SubscriptionDeliveryStatus represents delivery status for subscription.
type SubscriptionDeliveryStatus string

const (
	// SubscriptionDeliveryNone indicates no prior delivery for the subscription.
	SubscriptionDeliveryNone SubscriptionDeliveryStatus = ""
	// SubscriptionDeliverySucceeded indicates that last delivery for subscription succeeded.
	SubscriptionDeliverySucceeded SubscriptionDeliveryStatus = "succeeded"
	// SubscriptionDeliveryFailed indicates that last delivery for subscription failed.
	SubscriptionDeliveryFailed SubscriptionDeliveryStatus = "failed"
)

// Subscription is a subscription for Provisioner events.
type Subscription struct {
	ID                    string
	Name                  string
	URL                   string
	OwnerID               string
	EventType             EventType
	LastDeliveryStatus    SubscriptionDeliveryStatus
	LastDeliveryAttemptAt int64
	// FailureThreshold specifies the time, after which undelivered event will be considered failed.
	FailureThreshold time.Duration
	CreateAt         int64
	DeleteAt         int64
	LockAcquiredBy   *string
	LockAcquiredAt   int64
}

// IsDeleted returns true if subscription is deleted.
func (s Subscription) IsDeleted() bool {
	return s.DeleteAt > 0
}

// SubscriptionsFilter is a filter for subscription queries.
type SubscriptionsFilter struct {
	Paging
	Owner     string
	EventType EventType
}

// NewSubscriptionFromReader will create a Subscription from an
// io.Reader with JSON data.
func NewSubscriptionFromReader(reader io.Reader) (*Subscription, error) {
	var subscription Subscription
	err := json.NewDecoder(reader).Decode(&subscription)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode Subscription")
	}

	return &subscription, nil
}

// NewSubscriptionsFromReader will create a slice of Subscriptions from an
// io.Reader with JSON data.
func NewSubscriptionsFromReader(reader io.Reader) ([]*Subscription, error) {
	subscriptions := []*Subscription{}
	err := json.NewDecoder(reader).Decode(&subscriptions)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode Subscriptions")
	}

	return subscriptions, nil
}
