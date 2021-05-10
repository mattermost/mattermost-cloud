// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package components

import (
	"net/http"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/webhook"
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

type installationRestorationStore interface {
	TriggerInstallationRestoration(installation *model.Installation, backup *model.InstallationBackup) (*model.InstallationDBRestorationOperation, error)
	GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error)
}

// TriggerInstallationDBRestoration validates, triggers and reports installation database restoration.
func TriggerInstallationDBRestoration(store installationRestorationStore, installation *model.Installation, backup *model.InstallationBackup, env string, logger log.FieldLogger) (*model.InstallationDBRestorationOperation, error) {
	if err := model.EnsureInstallationReadyForDBRestoration(installation, backup); err != nil {
		return nil, ErrWrap(http.StatusBadRequest, err, "installation cannot be restored")
	}

	oldInstallationState := installation.State

	dbRestoration, err := store.TriggerInstallationRestoration(installation, backup)
	if err != nil {
		return nil, ErrWrap(http.StatusInternalServerError, err, "failed to create Installation DB restoration operation")
	}

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeInstallationDBRestoration,
		ID:        dbRestoration.ID,
		NewState:  string(model.InstallationDBRestorationStateRequested),
		OldState:  "n/a",
		Timestamp: time.Now().UnixNano(),
		ExtraData: map[string]string{"Installation": dbRestoration.InstallationID, "Backup": dbRestoration.BackupID, "Environment": env},
	}
	err = webhook.SendToAllWebhooks(store, webhookPayload, logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		logger.WithError(err).Error("Unable to process and send webhooks")
	}

	installationWebhookPayload := &model.WebhookPayload{
		Type:      model.TypeInstallation,
		ID:        installation.ID,
		NewState:  installation.State,
		OldState:  oldInstallationState,
		Timestamp: time.Now().UnixNano(),
		ExtraData: map[string]string{"DNS": installation.DNS, "Environment": env},
	}
	err = webhook.SendToAllWebhooks(store, installationWebhookPayload, logger.WithField("webhookEvent", installationWebhookPayload.NewState))
	if err != nil {
		logger.WithError(err).Error("Unable to process and send webhooks")
	}

	return dbRestoration, nil
}
