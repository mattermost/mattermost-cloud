// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"net/http"

	"github.com/mattermost/mattermost-cloud/model"
	cloud "github.com/mattermost/mattermost-cloud/model"
)

func webhookHandler(w http.ResponseWriter, r *http.Request, c chan *model.WebhookPayload) {
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
