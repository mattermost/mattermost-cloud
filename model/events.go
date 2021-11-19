// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

// EventType represents the Provisioners' event type.
type EventType string

const (
	// ResourceStateChangeEventType is event type representing resource state change.
	ResourceStateChangeEventType EventType = "resourceStateChange"
)

// Event represent event produced by Provisioner.
type Event struct {
	ID        string
	EventType EventType
	Timestamp int64
	ExtraData EventExtraData
}

// EventExtraData represents extra data of an Event.
type EventExtraData struct {
	Fields map[string]string `json:"fields"`
}

// EventDeliveryStatus represents status of EventDelivery
type EventDeliveryStatus string

const (
	// EventDeliveryNotAttempted indicates that delivery was not yet tried.
	EventDeliveryNotAttempted EventDeliveryStatus = "not-attempted"
	// EventDeliveryDelivered indicates that delivery was successful.
	EventDeliveryDelivered EventDeliveryStatus = "delivered"
	// EventDeliveryRetrying indicates that delivery failed but will be retired.
	EventDeliveryRetrying EventDeliveryStatus = "retrying"
	// EventDeliveryFailed indicates that delivery failed and will not be retried.
	EventDeliveryFailed EventDeliveryStatus = "failed"
)

// EventDelivery represents delivery status of particular Event to particular Subscription.
type EventDelivery struct {
	ID             string
	Status         EventDeliveryStatus
	LastAttempt    int64
	Attempts       int
	EventID        string
	SubscriptionID string
}

// EventDeliveryForEvent finds first EventDelivery for a particular eventID in collection.
func EventDeliveryForEvent(eventID string, deliveries []*EventDelivery) (*EventDelivery, bool) {
	for _, ed := range deliveries {
		if ed.EventID == eventID {
			return ed, true
		}
	}
	return nil, false
}
