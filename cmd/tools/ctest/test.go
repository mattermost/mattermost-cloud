// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

func runInstallationLifecycleTest(request *model.CreateInstallationRequest, client *model.Client, c chan *model.WebhookPayload) error {
	installation, err := client.CreateInstallation(request)
	if err != nil {
		return errors.Wrap(err, "failed to create installation")
	}

	out, _ := json.Marshal(installation)
	logger.Infof("Installation: %s", string(out))

	logger.Infof("Waiting for installation %s to go stable", installation.ID)
	for {
		payload := <-c
		if payload.ID == installation.ID && payload.NewState == model.InstallationStateStable {
			logger.Infof("Installation %s is now stable", installation.ID)
			break
		}
	}

	resp, err := http.Get(fmt.Sprintf("https://%s/api/v4/system/ping?get_server_status=true", installation.DNSRecords[0].DomainName))
	if err != nil {
		return errors.Wrap(err, "failed to run enhanced ping test")
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.WithError(err).Error("failed to close resp.Body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		var body []byte
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			logger.WithError(err).Warn("Installation failed health ping check and failed to decode response")
		} else {
			logger.Warnf("Installation failed health ping check: %s", string(body))
		}
	} else {
		logger.Info("Installation passed health ping check")
	}

	err = client.DeleteInstallation(installation.ID)
	if err != nil {
		return errors.Wrap(err, "failed to delete installation")
	}

	logger.Infof("Waiting for installation %s to be deleted", installation.ID)
	for {
		payload := <-c
		if payload.ID == installation.ID && payload.NewState == model.InstallationStateDeleted {
			logger.Infof("Installation %s is now deleted", installation.ID)
			break
		}
	}

	return nil
}
