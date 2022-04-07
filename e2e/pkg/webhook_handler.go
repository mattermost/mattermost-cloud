// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//+build e2e

package pkg

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/mattermost/mattermost-cloud/model"
)

// SetupTestWebhook sets up webhook used by test.
func SetupTestWebhook(client *model.Client, address, owner string, logger logrus.FieldLogger) (<-chan *model.WebhookPayload, func() error, error) {
	whURL, err := url.Parse(address)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse webhook address")
	}
	wh, err := registerWebhook(client, address, owner, logger)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to register webhook")
	}

	whChan := make(chan *model.WebhookPayload)

	shutdown := startWebhookListener(whURL.Port(), whChan, logger.WithField("handler", "webhook"))

	cleanup := func() error {
		shutdown()
		return cleanupWebhook(client, wh.ID)
	}

	return whChan, cleanup, nil
}

func registerWebhook(client *model.Client, address, owner string, logger logrus.FieldLogger) (*model.Webhook, error) {
	existingWh, err := client.GetWebhooks(&model.GetWebhooksRequest{
		Paging:  model.AllPagesNotDeleted(),
		OwnerID: owner,
	})
	if err != nil {
		logger.WithError(err).Error("failed to get e2e webhooks")
		// We do not want to fail over this, we will try to register.
	}
	for _, wh := range existingWh {
		if wh.URL == address {
			logger.Infof("Found existing webhook %q", wh.ID)
			return wh, nil
		}
	}
	logger.Infof("Webhook not found, registering...")
	
	whReq := &model.CreateWebhookRequest{
		OwnerID: owner,
		URL:     address,
	}

	wh, err := client.CreateWebhook(whReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to register webhook")
	}

	return wh, nil
}

func cleanupWebhook(client *model.Client, id string) error {
	return client.DeleteWebhook(id)
}

// startWebhookListener starts webhook listener.
func startWebhookListener(port string, c chan *model.WebhookPayload, logger logrus.FieldLogger) func() {
	logger.Infof("Starting cloud webhook listener on port %s", port)

	srv := &http.Server{Addr: fmt.Sprintf(":%s", port)}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		webhookHandler(w, r, c, logger)
	})

	go func() {
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to run webhook listener")
		}
	}()

	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			logger.WithError(err).Error("Failed to gracefully shutdown webhook listener")
		}
	}

	return shutdown
}

func webhookHandler(w http.ResponseWriter, r *http.Request, c chan *model.WebhookPayload, logger logrus.FieldLogger) {
	webhook, err := model.WebhookPayloadFromReader(r.Body)
	if err != nil {
		logger.WithError(err).Error("Failed to parse webhook")
		return
	}
	if len(webhook.ID) == 0 {
		return
	}

	c <- webhook

	logger.Debugf("[ %s | %s ] %s -> %s", webhook.Type, webhook.ID[0:4], webhook.OldState, webhook.NewState)

	w.WriteHeader(http.StatusOK)
}

type webhookWaiter struct {
	whChan        <-chan *model.WebhookPayload
	resourceType  string
	desiredState  string
	failureStates []string
	logger        logrus.FieldLogger
}

func (s *webhookWaiter) waitForState(ctx context.Context, id string) error {
	for {
		select {
		case payload := <-s.whChan:
			if payload.Type.String() != s.resourceType || payload.ID != id {
				continue
			}
			if payload.NewState == s.desiredState {
				return nil
			}
			if Contains(payload.NewState, s.failureStates) {
				return errors.Errorf("error waiting for state: resource reached a failure state %q", payload.NewState)
			}
			s.logger.Infof("Resource %q with ID %q transitioned to %q", s.resourceType, id, payload.NewState)
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for %q state for resource %q with ID %q", s.desiredState, s.resourceType, id)
		}
	}
}

func Contains(s string, all []string) bool {
	for _, str := range all {
		if s == str {
			return true
		}
	}
	return false
}
