// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWebhookIsDeleted(t *testing.T) {
	webhook := &Webhook{
		DeleteAt: 0,
	}

	t.Run("not deleted", func(t *testing.T) {
		require.False(t, webhook.IsDeleted())
	})

	webhook.DeleteAt = 1

	t.Run("deleted", func(t *testing.T) {
		require.True(t, webhook.IsDeleted())
	})
}

func TestWebhookPayloadToJSON(t *testing.T) {
	payload := &WebhookPayload{
		Timestamp: 123456789,
		ID:        "id",
		Type:      "type",
		NewState:  "state1",
		OldState:  "state2",
	}

	expectedStr := `{"timestamp":123456789,"id":"id","type":"type","new_state":"state1","old_state":"state2"}`

	payloadStr, err := payload.ToJSON()
	require.NoError(t, err)
	require.Equal(t, expectedStr, payloadStr)
}

func TestWebhookFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		webhook, err := WebhookFromReader(strings.NewReader(
			``,
		))
		require.NoError(t, err)
		require.Equal(t, &Webhook{}, webhook)
	})

	t.Run("invalid request", func(t *testing.T) {
		webhook, err := WebhookFromReader(strings.NewReader(
			`{test`,
		))
		require.Error(t, err)
		require.Nil(t, webhook)
	})

	t.Run("request", func(t *testing.T) {
		webhook, err := WebhookFromReader(strings.NewReader(`{
			"ID":"id",
			"OwnerID":"owner",
			"URL":"https://domain.com",
			"CreateAt":10,
			"DeleteAt":20
		}`))
		require.NoError(t, err)
		require.Equal(t, &Webhook{
			ID:       "id",
			OwnerID:  "owner",
			URL:      "https://domain.com",
			CreateAt: 10,
			DeleteAt: 20,
		}, webhook)
	})
}

func TestWebhooksFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		webhooks, err := WebhooksFromReader(strings.NewReader(
			``,
		))
		require.NoError(t, err)
		require.Equal(t, []*Webhook{}, webhooks)
	})

	t.Run("invalid request", func(t *testing.T) {
		webhooks, err := WebhooksFromReader(strings.NewReader(
			`{test`,
		))
		require.Error(t, err)
		require.Nil(t, webhooks)
	})

	t.Run("request", func(t *testing.T) {
		webhooks, err := WebhooksFromReader(strings.NewReader(`[
			{
				"ID":"id1",
				"OwnerID":"owner1",
				"URL":"https://domain1.com",
				"CreateAt":10,
				"DeleteAt":20
			},
			{
				"ID":"id2",
				"OwnerID":"owner2",
				"URL":"https://domain2.com",
				"CreateAt":30,
				"DeleteAt":40
			}
		]`))
		require.NoError(t, err)
		require.Equal(t, []*Webhook{
			{
				ID:       "id1",
				OwnerID:  "owner1",
				URL:      "https://domain1.com",
				CreateAt: 10,
				DeleteAt: 20,
			},
			{
				ID:       "id2",
				OwnerID:  "owner2",
				URL:      "https://domain2.com",
				CreateAt: 30,
				DeleteAt: 40,
			},
		}, webhooks)
	})
}

func TestWebhookPayloadFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		payload, err := WebhookPayloadFromReader(strings.NewReader(
			``,
		))
		require.NoError(t, err)
		require.Equal(t, &WebhookPayload{}, payload)
	})

	t.Run("invalid request", func(t *testing.T) {
		payload, err := WebhookPayloadFromReader(strings.NewReader(
			`{test`,
		))
		require.Error(t, err)
		require.Nil(t, payload)
	})

	t.Run("request", func(t *testing.T) {
		payload, err := WebhookPayloadFromReader(strings.NewReader(`{
			"timestamp":1234,
			"id":"id",
			"type":"installation",
			"new_state":"stable",
			"old_state":"creation-in-progress"
		}`))
		require.NoError(t, err)
		require.Equal(t, &WebhookPayload{
			Timestamp: 1234,
			ID:        "id",
			Type:      "installation",
			NewState:  "stable",
			OldState:  "creation-in-progress",
		}, payload)
	})
}
