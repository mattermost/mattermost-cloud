// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//go:build e2e
// +build e2e

package installation

import (
	"context"
	"github.com/pkg/errors"
	"math/rand"
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/e2e/tests/shared"
	"github.com/mattermost/mattermost-cloud/e2e/workflow"

	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	shared.TestMain(m)
}

func SetupInstallationLifecycleTest() (*shared.Test, error) {

	test, err := shared.SetupTestWithDefaults()

	if err != nil {
		return nil, errors.Wrap(err, "failed to setup test environment")
	}

	return test, nil
}

func Test_InstallationLifecycle(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	test, err := SetupInstallationLifecycleTest()
	require.NoError(t, err)
	testWorkflowSteps := installationLifecycleSteps(test.ClusterSuite, test.InstallationSuite)

	test.Workflow = workflow.NewWorkflow(testWorkflowSteps)
	test.Steps = testWorkflowSteps

	defer test.CleanupTest(t)
	err = test.EventsRecorder.Start(test.ProvisionerClient, test.Logger)
	require.NoError(t, err)
	defer test.EventsRecorder.ShutDown(test.ProvisionerClient)

	err = test.Run()
	require.NoError(t, err)

	// Make sure we wait for all subscription events
	time.Sleep(time.Second * 1)

	// Make sure that expected events occurred in correct order.
	expectedEvents := workflow.GetExpectedEvents(test.Steps)
	err = test.EventsRecorder.VerifyInOrder(expectedEvents)
	require.NoError(t, err)
}
