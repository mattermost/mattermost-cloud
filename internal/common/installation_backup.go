// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package common

import (
	"net/http"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/webhook"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type installationBackupStore interface {
	IsInstallationBackupRunning(installationID string) (bool, error)
	CreateInstallationBackup(backup *model.InstallationBackup) error
	GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error)
}

// TriggerInstallationBackup verifies that backup can be started for an Installation and triggers it.
func TriggerInstallationBackup(store installationBackupStore, installation *model.Installation, env string, logger log.FieldLogger) (*model.InstallationBackup, error) {
	err := model.EnsureInstallationReadyForBackup(installation)
	if err != nil {
		return nil, ErrWrap(http.StatusBadRequest, err, "installation cannot be backed up")
	}

	backupRunning, err := store.IsInstallationBackupRunning(installation.ID)
	if err != nil {
		return nil, ErrWrap(http.StatusInternalServerError, err, "failed to check if backup is running for Installation")
	}
	if backupRunning {
		return nil, NewErr(http.StatusBadRequest, errors.New("backup for the installation is already requested or in progress"))
	}

	backup := &model.InstallationBackup{
		InstallationID: installation.ID,
		State:          model.InstallationBackupStateBackupRequested,
	}

	err = store.CreateInstallationBackup(backup)
	if err != nil {
		return nil, ErrWrap(http.StatusInternalServerError, err, "failed to create installation backup")
	}

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeInstallationBackup,
		ID:        backup.ID,
		NewState:  string(backup.State),
		OldState:  "n/a",
		Timestamp: time.Now().UnixNano(),
		ExtraData: map[string]string{"Installation": backup.InstallationID, "Environment": env},
	}
	err = webhook.SendToAllWebhooks(store, webhookPayload, logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		logger.WithError(err).Error("Unable to process and send webhooks")
	}

	return backup, nil
}
