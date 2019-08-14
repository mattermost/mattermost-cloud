package model

import (
	"bytes"
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
		webhook, err := WebhookFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, &Webhook{}, webhook)
	})

	t.Run("invalid request", func(t *testing.T) {
		webhook, err := WebhookFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, webhook)
	})

	t.Run("request", func(t *testing.T) {
		webhook, err := WebhookFromReader(bytes.NewReader([]byte(`{
			"ID":"id",
			"OwnerID":"owner",
			"URL":"https://domain.com",
			"CreateAt":10,
			"DeleteAt":20
		}`)))
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
		webhooks, err := WebhooksFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, []*Webhook{}, webhooks)
	})

	t.Run("invalid request", func(t *testing.T) {
		webhooks, err := WebhooksFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, webhooks)
	})

	t.Run("request", func(t *testing.T) {
		webhooks, err := WebhooksFromReader(bytes.NewReader([]byte(`[
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
		]`)))
		require.NoError(t, err)
		require.Equal(t, []*Webhook{
			&Webhook{
				ID:       "id1",
				OwnerID:  "owner1",
				URL:      "https://domain1.com",
				CreateAt: 10,
				DeleteAt: 20,
			},
			&Webhook{
				ID:       "id2",
				OwnerID:  "owner2",
				URL:      "https://domain2.com",
				CreateAt: 30,
				DeleteAt: 40,
			},
		}, webhooks)
	})
}
