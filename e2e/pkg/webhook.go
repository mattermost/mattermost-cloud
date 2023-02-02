// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"

	"github.com/pkg/errors"
)

type webhookPayload struct {
	Username  string `json:"username"`
	IconURL   string `json:"icon_url"`
	IconEmoji string `json:"icon_emoji"`
	Text      string `json:"text"`
}

// sendWebhook sends a Mattermost webhook to the provided URL.
func sendWebhook(ctx context.Context, webhookURL string, payload *webhookPayload) error {
	if len(payload.Username) == 0 {
		return errors.New("payload username value not set")
	}
	if len(payload.Text) == 0 {
		return errors.New("payload text value not set")
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "failed to marshal payload")
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	_, err = http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send webhook")
	}

	return nil
}

// SendE2EResult sends the webhook with the provided icon and message. Errors on trying to send a
// message or if the webhook URL is not provided properly.
func SendE2EResult(ctx context.Context, icon, text string) error {
	webhookURL := os.Getenv("WEBHOOK_URL")
	_, err := url.ParseRequestURI(webhookURL)
	if err != nil {
		return errors.New("incorrect or empty webhook url")
	}

	payload := webhookPayload{
		Username:  "E2E",
		IconEmoji: icon,
		Text:      text,
	}

	if err := sendWebhook(ctx, webhookURL, &payload); err != nil {
		return errors.Wrap(err, "error sending notification webhook")
	}

	return nil
}
