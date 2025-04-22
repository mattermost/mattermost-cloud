// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//go:build e2e
// +build e2e

package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mattermost/mattermost-cloud/model"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// installationWaitTimeout is the maximum time to wait for an installation to be ready.
const installationWaitTimeout = 15 * time.Minute

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
	err := WaitForFunc(NewWaitConfig(installationWaitTimeout, 10*time.Second, 2, log), func() (bool, error) {
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

// WaitForInstallationToBeStable waits until installation reaches Stable state.
func WaitForInstallationToBeStable(ctx context.Context, installationID string, whChan <-chan *model.WebhookPayload, log logrus.FieldLogger) error {
	waitCtx, cancel := context.WithTimeout(ctx, installationWaitTimeout)
	defer cancel()

	whWaiter := webhookWaiter{
		whChan:        whChan,
		resourceType:  model.TypeInstallation.String(),
		desiredState:  model.InstallationStateStable,
		failureStates: []string{model.InstallationStateCreationFailed, model.InstallationStateCreationNoCompatibleClusters},
		logger: log.WithFields(map[string]interface{}{
			"installation":  installationID,
			"desired-state": model.InstallationStateStable,
		}),
	}

	return whWaiter.waitForState(waitCtx, installationID)
}

// WaitForInstallationDeletion waits until Installation reaches Deleted state.
func WaitForInstallationDeletion(client *model.Client, installationID string, log logrus.FieldLogger) error {
	err := WaitForFunc(NewWaitConfig(10*time.Minute, 10*time.Second, 2, log), func() (bool, error) {
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

// WaitForInstallationToBeDeletionPending waits until installation is in the deletion pending state.
func WaitForInstallationToBeDeletionPending(ctx context.Context, installationID string, whChan <-chan *model.WebhookPayload, log logrus.FieldLogger) error {
	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	whWaiter := webhookWaiter{
		whChan:        whChan,
		resourceType:  model.TypeInstallation.String(),
		desiredState:  model.InstallationStateDeletionPending,
		failureStates: []string{model.InstallationStateDeletionCancellationRequested, model.InstallationStateDeletionFailed},
		logger: log.WithFields(map[string]interface{}{
			"installation":  installationID,
			"desired-state": model.InstallationStateDeletionPending,
		}),
	}

	return whWaiter.waitForState(waitCtx, installationID)
}

// WaitForInstallationToBeDeleted waits until installation is deleted.
func WaitForInstallationToBeDeleted(ctx context.Context, installationID string, whChan <-chan *model.WebhookPayload, log logrus.FieldLogger) error {
	waitCtx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()

	whWaiter := webhookWaiter{
		whChan:        whChan,
		resourceType:  model.TypeInstallation.String(),
		desiredState:  model.InstallationStateDeleted,
		failureStates: []string{model.InstallationStateDeletionFailed},
		logger: log.WithFields(map[string]interface{}{
			"installation":  installationID,
			"desired-state": model.InstallationStateDeleted,
		}),
	}

	return whWaiter.waitForState(waitCtx, installationID)
}

// WaitForClusterInstallationReadyStatus pings installation until it responds successfully.
func WaitForClusterInstallationReadyStatus(client *model.Client, clusterInstallationID string, log logrus.FieldLogger) error {
	err := WaitForFunc(NewWaitConfig(5*time.Minute, 20*time.Second, 0, log), func() (bool, error) {
		status, err := client.GetClusterInstallationStatus(clusterInstallationID)
		if err != nil {
			return false, errors.Wrap(err, "while waiting for cluster installation to be ready")
		}

		if status.Replicas != nil && status.ReadyLocalServer != nil {
			if *status.Replicas == *status.ReadyLocalServer {
				return true, nil
			}

			log.Infof("Cluster installation %s not ready: %d/%d", clusterInstallationID, *status.ReadyLocalServer, *status.Replicas)
			statusByte, _ := json.Marshal(status)
			log.Debug(string(statusByte))
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
