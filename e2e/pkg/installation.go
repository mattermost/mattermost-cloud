// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//+build e2e

package pkg

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mattermost/mattermost-cloud/model"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// WaitForInstallationAvailability pings installation until it responds successfully.
func WaitForInstallationAvailability(dns string, log logrus.FieldLogger) error {
	err := WaitForFunc(NewWaitConfig(20*time.Minute, 20*time.Second, 2, log), func() (bool, error) {
		err := PingInstallation(dns)
		if err != nil {
			log.WithError(err).Errorf("Error while making ping request to Installation %s", dns)
			return false, nil
		}
		return true, nil
	})

	return err
}

// WaitForHibernation waits until Installation is fully hibernated.
func WaitForHibernation(client *model.Client, installationID string, log logrus.FieldLogger) error {
	err := WaitForFunc(NewWaitConfig(5*time.Minute, 10*time.Second, 2, log), func() (bool, error) {
		installation, err := client.GetInstallation(installationID, &model.GetInstallationRequest{})
		if err != nil {
			return false, errors.Wrap(err, "while waiting for hibernation")
		}

		if installation.State == model.InstallationStateHibernating {
			return true, nil
		}
		log.Infof("Installation %s not hibernated: %s", installationID, installation.State)
		return false, nil
	})
	return err
}

// WaitForStable waits until Installation reaches Stable state.
func WaitForStable(client *model.Client, installationID string, log logrus.FieldLogger) error {
	err := WaitForFunc(NewWaitConfig(5*time.Minute, 10*time.Second, 2, log), func() (bool, error) {
		installation, err := client.GetInstallation(installationID, &model.GetInstallationRequest{})
		if err != nil {
			return false, errors.Wrap(err, "while waiting for stable")
		}

		if installation.State == model.InstallationStateStable {
			return true, nil
		}
		log.Infof("Installation %s not stable: %s", installationID, installation.State)
		return false, nil
	})
	return err
}

// WaitForInstallationDeletion waits until Installation reaches Deleted state.
func WaitForInstallationDeletion(client *model.Client, installationID string, log logrus.FieldLogger) error {
	err := WaitForFunc(NewWaitConfig(5*time.Minute, 10*time.Second, 2, log), func() (bool, error) {
		installation, err := client.GetInstallation(installationID, &model.GetInstallationRequest{})
		if err != nil {
			return false, errors.Wrap(err, "while waiting for deletion")
		}

		if installation.State == model.InstallationStateDeleted {
			return true, nil
		}
		log.Infof("Installation %s not deleted: %s", installationID, installation.State)
		if installation.State == model.InstallationStateDeletionFailed {
			return false, errors.New("installation deletion failed")
		}
		return false, nil
	})
	return err
}

// PingInstallation hits Mattermost ping endpoint.
func PingInstallation(dns string) error {
	resp, err := http.Get(pingURL(dns))
	if err != nil {
		return errors.Wrap(err, "failed to ping installation")
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("error pinging installation, received status: %s", resp.Status)
	}
	return nil
}

func pingURL(dns string) string {
	return fmt.Sprintf("https://%s/api/v4/system/ping", dns)
}

