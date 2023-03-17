// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package webhook

import (
	"bytes"
	"net/http"
	"time"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

type webhookStore interface {
	GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error)
}

// SendToAllWebhooks sends a given payload to all webhooks.
func SendToAllWebhooks(store webhookStore, payload *model.WebhookPayload, logger *log.Entry) error {
	hooks, err := store.GetWebhooks(&model.WebhookFilter{
		Paging: model.AllPagesNotDeleted(),
	})
	if err != nil {
		return errors.Wrap(err, "Failed to find webhooks")
	}

	sendWebhooks(hooks, payload, logger)

	return nil
}

// sendWebhooks sends webhooks via fire-and-forget goroutines. The send-webhook
// failures are logged, but not handled.
func sendWebhooks(hooks []*model.Webhook, payload *model.WebhookPayload, logger *log.Entry) {
	if len(hooks) == 0 {
		return
	}

	logger.Debugf("Sending %d webhook(s)", len(hooks))

	for _, hook := range hooks {
		go sendWebhook(hook, payload, logger)
	}
}

func sendWebhook(hook *model.Webhook, payload *model.WebhookPayload, logger *log.Entry) error {
	payloadStr, err := payload.ToJSON()
	if err != nil {
		logger.WithField("webhookURL", hook.URL).WithError(err).Error("Unable to create payload string to send to webhook")
		return errors.Wrap(err, "unable to create payload string to send to webhook")
	}

	req, err := http.NewRequest("POST", hook.URL, bytes.NewBuffer([]byte(payloadStr)))
	if err != nil {
		logger.WithField("webhookURL", hook.URL).WithError(err).Error("Unable to create request")
		return errors.Wrap(err, "unable to create request from payload")
	}
	req.Header.Set("Content-Type", "application/json")
	headers, err := model.ParseHeadersFromStringMap(hook.Headers)
	if err != nil {
		// If there's an error parsing the headers, log it but continue execution so the webhook is
		// sent, `model.ParseHeadersFromStringMap` should take care of not disclosing any unset
		// environment variables into resulting headers.
		logger.WithFields(logrus.Fields{
			"webhook": hook.ID,
		}).WithError(err).Error()
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	_, err = client.Do(req)
	if err != nil {
		logger.WithField("webhookURL", hook.URL).WithError(err).Error("Unable to send webhook")
		return errors.Wrap(err, "unable to send webhook")
	}

	return nil
}
