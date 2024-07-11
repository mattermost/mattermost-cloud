// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//go:build e2e
// +build e2e

package cluster

import (
	"math/rand"
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-cloud/e2e/tests/shared"
	"github.com/mattermost/mattermost-cloud/e2e/workflow"

	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// This is mainly used to send a notification when tests are finished to a mattermost webhook
	// provided with the WEBHOOOK_URL environment variable.
	shared.TestMain(m)
}

// SetupClusterLifecycleTest sets up cluster lifecycle test.
func SetupClusterLifecycleTest() (*shared.Test, error) {
	test, err := shared.SetupTestWithDefaults("cluster-lifecycle", "argocd")
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup test environment")
	}

	return test, nil
}

func Test_ClusterLifecycle(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	t.Parallel()

	test, err := SetupClusterLifecycleTest()
	require.NoError(t, err)
	testWorkflowSteps := clusterLifecycleSteps(test.ClusterSuite, test.InstallationSuite)

	test.Workflow = workflow.NewWorkflow(testWorkflowSteps)
	test.Steps = testWorkflowSteps

	err = test.EventsRecorder.Start(test.ProvisionerClient, test.Logger)
	require.NoError(t, err)
	defer test.EventsRecorder.ShutDown(test.ProvisionerClient)
	defer test.CleanupTest(t)

	err = test.Run()
	require.NoError(t, err)

	// Make sure we wait for all subscription events
	time.Sleep(time.Second * 1)

	// Make sure that expected events occurred in correct order.
	expectedEvents := workflow.GetExpectedEvents(test.Steps)
	err = test.EventsRecorder.VerifyInOrder(expectedEvents)
	require.NoError(t, err)
}
