// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"encoding/json"
	"fmt"
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

	resp, err := http.Get(fmt.Sprintf("https://%s/api/v4/system/ping?get_server_status=true", installation.DNS))
	if err != nil {
		return errors.Wrap(err, "failed to run enhanced ping test")
	}
	if resp.StatusCode != http.StatusOK {
		logger.Warn("Installation failed health ping check")
	} else {
		logger.Info("Installation passed health check")
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
