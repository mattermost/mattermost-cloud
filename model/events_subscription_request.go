// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
	"net/url"
	"time"

	"github.com/pkg/errors"
)

// CreateSubscriptionRequest represents a request to create Subscription.
type CreateSubscriptionRequest struct {
	Name             string
	URL              string
	OwnerID          string
	EventType        EventType
	FailureThreshold time.Duration
	Headers          Headers
}

// ToSubscription validates request and converts it to subscription
func (r CreateSubscriptionRequest) ToSubscription() (Subscription, error) {
	_, err := url.Parse(r.URL)
	if err != nil {
		return Subscription{}, errors.Wrap(err, "failed to parse subscription URL")
	}
	if r.EventType == "" {
		return Subscription{}, errors.New("event type is required when registering subscription")
	}
	if r.OwnerID == "" {
		return Subscription{}, errors.New("owner ID is required when registering subscription")
	}
	if r.FailureThreshold < 0 || r.FailureThreshold > 72*time.Hour {
		return Subscription{}, errors.New("failure threshold need to be between 0 and 72 hours")
	}

	return Subscription{
		Name:                  r.Name,
		URL:                   r.URL,
		OwnerID:               r.OwnerID,
		EventType:             r.EventType,
		LastDeliveryStatus:    SubscriptionDeliveryNone,
		LastDeliveryAttemptAt: 0,
		FailureThreshold:      r.FailureThreshold,
		Headers:               r.Headers,
	}, nil
}

// NewCreateSubscriptionRequestFromReader will create a CreateSubscriptionRequest from an
// io.Reader with JSON data.
func NewCreateSubscriptionRequestFromReader(reader io.Reader) (*CreateSubscriptionRequest, error) {
	subRequest := CreateSubscriptionRequest{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&subRequest)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &subRequest, nil
}

// ListSubscriptionsRequest represents a request data for querying subscriptions.
type ListSubscriptionsRequest struct {
	Paging
	Owner     string
	EventType EventType
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *ListSubscriptionsRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	q.Add("owner", request.Owner)
	q.Add("event_type", string(request.EventType))
	request.Paging.AddToQuery(q)

	u.RawQuery = q.Encode()
}
