// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	cloud "github.com/mattermost/mattermost-cloud/model"
)

func webhookHandler(w http.ResponseWriter, r *http.Request, c chan *cloud.WebhookPayload) {
	webhook, err := cloud.WebhookPayloadFromReader(r.Body)
	if err != nil {
		logger.WithError(err).Error("Faled to parse webhook")
		return
	}
	if len(webhook.ID) == 0 {
		return
	}

	wType := "UNKN"
	switch webhook.Type {
	case cloud.TypeCluster:
		wType = "CLSR"
	case cloud.TypeInstallation:
		wType = "INST"
	case cloud.TypeClusterInstallation:
		wType = "CLIN"
	}

	c <- webhook

	logger.Debugf("[ %s | %s ] %s -> %s", wType, webhook.ID[0:4], webhook.OldState, webhook.NewState)

	w.WriteHeader(http.StatusOK)
}

func startWebhookListener(port string, c chan *cloud.WebhookPayload) func() {
	logger.Infof("Starting cloud webhook listener on port %s", port)

	srv := &http.Server{Addr: fmt.Sprintf(":%s", port)}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		webhookHandler(w, r, c)
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
