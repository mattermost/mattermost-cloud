// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//+build e2e

package pkg

import (
	"context"
	"time"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/sirupsen/logrus"
)

var (
	ClusterProvisioningFailedStates = []string{model.ClusterStateCreationFailed, model.ClusterStateProvisioningFailed}
)


// WaitForClusterToBeStable waits until Cluster reaches Stable state.
func WaitForClusterToBeStable(ctx context.Context, clusterID string, whChan <-chan *model.WebhookPayload, log logrus.FieldLogger) error {
	waitCtx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()

	whWaiter := webhookWaiter{
		whChan:        whChan,
		resourceType:  model.TypeCluster.String(),
		desiredState:  model.ClusterStateStable,
		failureStates: ClusterProvisioningFailedStates,
		logger: log.WithFields(map[string]interface{}{
			"cluster":       clusterID,
			"desired-state": model.ClusterStateStable,
		}),
	}

	return whWaiter.waitForState(waitCtx, clusterID)
}

// WaitForClusterDeletion waits until Cluster is deleted.
func WaitForClusterDeletion(ctx context.Context, clusterID string, whChan <-chan *model.WebhookPayload, log logrus.FieldLogger) error {
	waitCtx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()

	whWaiter := webhookWaiter{
		whChan:        whChan,
		resourceType:  model.TypeCluster.String(),
		desiredState:  model.ClusterStateDeleted,
		failureStates: []string{model.ClusterStateDeletionFailed},
		logger: log.WithFields(map[string]interface{}{
			"cluster":       clusterID,
			"desired-state": model.ClusterStateDeleted,
		}),
	}

	return whWaiter.waitForState(waitCtx, clusterID)
}
