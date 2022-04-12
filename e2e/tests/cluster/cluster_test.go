// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//+build e2e

package cluster

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/e2e/workflow"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ClusterLifecycle(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	t.Parallel()

	test, err := SetupClusterLifecycleTest()
	require.NoError(t, err)
	// Always cleanup webhook
	defer func() {
		err := test.WebhookCleanup()
		assert.NoError(t, err)
	}()
	if test.Cleanup {
		defer func() {
			err = test.InstallationSuite.Cleanup(context.Background())
			if err != nil {
				test.Logger.WithError(err).Error("Error cleaning up installation")
			}
			err := test.ClusterSuite.DeleteCluster(context.Background())
			if err != nil {
				test.Logger.WithError(err).Error("Error cleaning up cluster")
			}
		}()
	}
	err = test.EventsRecorder.Start(test.ProvisionerClient, test.Logger)
	require.NoError(t, err)
	defer test.EventsRecorder.ShutDown(test.ProvisionerClient)

	err = test.Run()
	require.NoError(t, err)

	// Make sure that expected events occurred in correct order.
	expectedEvents := workflow.GetExpectedEvents(test.Steps)
	err = test.EventsRecorder.VerifyInOrder(expectedEvents)
	require.NoError(t, err)
}
