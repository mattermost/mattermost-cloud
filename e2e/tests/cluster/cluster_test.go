// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//go:build e2e
// +build e2e

package cluster

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/e2e/pkg"
	"github.com/mattermost/mattermost-cloud/e2e/tests/state"
	"github.com/mattermost/mattermost-cloud/e2e/workflow"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	webhookSuccessfulMessage      = "Provisioner E2E tests passed successfully"
	webhookFailedMessage          = `Provisioner E2E tests failed`
	webhookSuccessEmoji           = "large_green_circle"
	webhookFailedEmoji            = "red_circle"
	webhookAttachmentColorSuccess = "#009E60"
	webhookAttachmentColorError   = "#FF0000"
)

func TestMain(m *testing.M) {
	// This is mainly used to send a notification when tests are finished to a mattermost webhook
	// provided with the WEBHOOOK_URL environment variable.
	state.StartTime = time.Now()
	code := m.Run()
	state.EndTime = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var err error
	if code != 0 {
		err = pkg.SendE2EResult(ctx, webhookFailedEmoji, webhookFailedMessage, webhookAttachmentColorError)
	} else {
		err = pkg.SendE2EResult(ctx, webhookSuccessEmoji, webhookSuccessfulMessage, webhookAttachmentColorSuccess)
	}

	if err != nil {
		fmt.Printf("error sending webhook: %s", err)
	}

	os.Exit(code)
}

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
			err := test.ClusterSuite.Cleanup(context.Background())
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
