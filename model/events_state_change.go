// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
	"net/url"
)

// StateChangeEvent contains data specific to StateChangeEvent.
type StateChangeEvent struct {
	ID           string
	EventID      string
	OldState     string
	NewState     string
	ResourceID   string
	ResourceType ResourceType
}

// StateChangeEventData is a combination of StateChangeEvent and Event data.
type StateChangeEventData struct {
	Event       Event
	StateChange StateChangeEvent
}

// StateChangeEventDeliveryData is a combination of StateChangeEventData and EventDelivery.
type StateChangeEventDeliveryData struct {
	EventDelivery EventDelivery
	EventData     StateChangeEventData
	EventHeaders  Headers
}

// StateChangeEventPayload represents payload that is sent to consumers.
type StateChangeEventPayload struct {
	EventID      string       `json:"eventId"`
	Timestamp    int64        `json:"timestamp"`
	ResourceID   string       `json:"resourceId"`
	ResourceType ResourceType `json:"resourceType"`
	NewState     string       `json:"newState"`
	OldState     string       `json:"oldState"`
	ExtraData    map[string]string
}

// ToEventPayload converts StateChangeEventData to StateChangeEventPayload.
func (e *StateChangeEventData) ToEventPayload() StateChangeEventPayload {
	return StateChangeEventPayload{
		EventID:      e.Event.ID,
		Timestamp:    e.Event.Timestamp,
		ResourceID:   e.StateChange.ResourceID,
		ResourceType: e.StateChange.ResourceType,
		NewState:     e.StateChange.NewState,
		OldState:     e.StateChange.OldState,
		ExtraData:    e.Event.ExtraData.Fields,
	}
}

// ToWebhookPayload converts StateChangeEventData to WebhookPayload.
func (e *StateChangeEventData) ToWebhookPayload() WebhookPayload {
	return WebhookPayload{
		EventID:   e.Event.ID,
		Timestamp: e.Event.Timestamp,
		ID:        e.StateChange.ResourceID,
		Type:      e.StateChange.ResourceType,
		NewState:  e.StateChange.NewState,
		OldState:  e.StateChange.OldState,
		ExtraData: e.Event.ExtraData.Fields,
	}
}

// NewStateChangeEventPayloadFromReader will create a StateChangeEventPayload from an
// io.Reader with JSON data.
func NewStateChangeEventPayloadFromReader(reader io.Reader) (*StateChangeEventPayload, error) {
	eventPayload := StateChangeEventPayload{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&eventPayload)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &eventPayload, nil
}

// NewStateChangeEventsDataFromReader will create an array of StateChangeEventData from an
// io.Reader with JSON data.
func NewStateChangeEventsDataFromReader(reader io.Reader) ([]*StateChangeEventData, error) {
	data := []*StateChangeEventData{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&data)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return data, nil
}

// StateChangeEventFilter is a filter for state change event queries.
type StateChangeEventFilter struct {
	Paging
	ResourceType ResourceType
	ResourceID   string
	OldStates    []string
	NewStates    []string
}

// ListStateChangeEventsRequest represents request for state change events query.
type ListStateChangeEventsRequest struct {
	Paging
	ResourceType ResourceType
	ResourceID   string
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *ListStateChangeEventsRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	q.Add("resource_type", string(request.ResourceType))
	q.Add("resource_id", request.ResourceID)
	request.Paging.AddToQuery(q)

	u.RawQuery = q.Encode()
}
