// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStateChangeEventPayloadFromReader(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		stateChangeEventPayload, err := NewStateChangeEventPayloadFromReader(bytes.NewReader([]byte(
			"",
		)))
		require.NoError(t, err)
		require.Equal(t, &StateChangeEventPayload{}, stateChangeEventPayload)
	})

	t.Run("invalid", func(t *testing.T) {
		stateChangeEventPayload, err := NewStateChangeEventPayloadFromReader(bytes.NewReader([]byte(
			"{test",
		)))
		require.Error(t, err)
		require.Nil(t, stateChangeEventPayload)
	})

	t.Run("valid", func(t *testing.T) {
		stateChangeEventPayload, err := NewStateChangeEventPayloadFromReader(bytes.NewReader([]byte(
			`{"eventID":"event-1", "timestamp":1000}`,
		)))
		require.NoError(t, err)
		require.Equal(t, &StateChangeEventPayload{
			EventID:   "event-1",
			Timestamp: 1000,
		}, stateChangeEventPayload)
	})
}

func TestNewStateChangeEventsDataFromReader(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		stateChangeEventDatas, err := NewStateChangeEventsDataFromReader(bytes.NewReader([]byte(
			"",
		)))
		require.NoError(t, err)
		require.Equal(t, []*StateChangeEventData{}, stateChangeEventDatas)
	})

	t.Run("invalid", func(t *testing.T) {
		stateChangeEventDatas, err := NewStateChangeEventsDataFromReader(bytes.NewReader([]byte(
			"{test",
		)))
		require.Error(t, err)
		require.Nil(t, stateChangeEventDatas)
	})

	t.Run("valid", func(t *testing.T) {
		stateChangeEventDatas, err := NewStateChangeEventsDataFromReader(bytes.NewReader([]byte(
			`[
				{"event":{"id":"event-1","timestamp":1000},"stateChange":{"id":"state-change-1","eventID":"event-1"}},
				{"event":{"id":"event-2","timestamp":1000},"stateChange":{"id":"state-change-2","eventID":"event-2"}}
			]`,
		)))
		require.NoError(t, err)
		require.Equal(t, []*StateChangeEventData{
			{
				Event: Event{
					ID:        "event-1",
					Timestamp: 1000,
				},
				StateChange: StateChangeEvent{
					ID:      "state-change-1",
					EventID: "event-1",
				},
			},
			{
				Event: Event{
					ID:        "event-2",
					Timestamp: 1000,
				},
				StateChange: StateChangeEvent{
					ID:      "state-change-2",
					EventID: "event-2",
				},
			},
		}, stateChangeEventDatas)
	})
}

func TestStateChangeEventDataToPayload(t *testing.T) {
	stateChangeEvent := StateChangeEventData{
		Event: Event{
			ID:        "eid",
			EventType: ResourceStateChangeEventType,
			Timestamp: 100,
			ExtraData: EventExtraData{
				Fields: map[string]string{"test": "test"},
			},
		},
		StateChange: StateChangeEvent{
			ID:           "scid",
			EventID:      "eid",
			OldState:     "old",
			NewState:     "new",
			ResourceID:   "resid",
			ResourceType: TypeInstallation,
		},
	}

	eventPayload := stateChangeEvent.ToEventPayload()
	assert.Equal(t, stateChangeEvent.Event.ID, eventPayload.EventID)
	assert.Equal(t, stateChangeEvent.Event.Timestamp, eventPayload.Timestamp)
	assert.Equal(t, stateChangeEvent.Event.ExtraData.Fields, eventPayload.ExtraData)
	assert.Equal(t, stateChangeEvent.StateChange.ResourceType, eventPayload.ResourceType)
	assert.Equal(t, stateChangeEvent.StateChange.ResourceID, eventPayload.ResourceID)
	assert.Equal(t, stateChangeEvent.StateChange.NewState, eventPayload.NewState)
	assert.Equal(t, stateChangeEvent.StateChange.OldState, eventPayload.OldState)

	webhookPayload := stateChangeEvent.ToWebhookPayload()
	assert.Equal(t, stateChangeEvent.Event.ID, webhookPayload.EventID)
	assert.Equal(t, stateChangeEvent.Event.Timestamp, webhookPayload.Timestamp)
	assert.Equal(t, stateChangeEvent.Event.ExtraData.Fields, webhookPayload.ExtraData)
	assert.Equal(t, stateChangeEvent.StateChange.ResourceType, webhookPayload.Type)
	assert.Equal(t, stateChangeEvent.StateChange.ResourceID, webhookPayload.ID)
	assert.Equal(t, stateChangeEvent.StateChange.NewState, webhookPayload.NewState)
	assert.Equal(t, stateChangeEvent.StateChange.OldState, webhookPayload.OldState)
}
