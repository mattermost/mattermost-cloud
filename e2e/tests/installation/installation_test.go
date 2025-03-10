// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//go:build e2e
// +build e2e

package installation

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-cloud/e2e/tests/shared"
	"github.com/mattermost/mattermost-cloud/e2e/workflow"

	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	shared.TestMain(m)
}

func SetupInstallationLifecycleTest() (*shared.Test, error) {
	test, err := shared.SetupTestWithDefaults("installation-lifecycle")
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup test environment")
	}

	return test, nil
}

func Test_InstallationLifecycle(t *testing.T) {
	ctx := context.Background()
	rand.Seed(time.Now().UnixNano())

	test, err := SetupInstallationLifecycleTest()
	require.NoError(t, err)

	err = test.EventsRecorder.Start(test.ProvisionerClient, test.Logger)
	require.NoError(t, err)
	defer test.EventsRecorder.ShutDown(test.ProvisionerClient)
	defer test.CleanupTest(t)

	cases := []struct {
		Name              string
		WorkflowStepsFunc func(*workflow.ClusterSuite, *workflow.InstallationSuite) []*workflow.Step
	}{
		{
			Name:              "Create installation, populate sample data, then delete installation",
			WorkflowStepsFunc: basicCreateDeleteInstallationSteps,
		},
		{
			Name:              "Create and delete installation with a versioned s3 bucket",
			WorkflowStepsFunc: versionedS3BucketInstallationLifecycleSteps,
		},
		{
			Name:              "Create and delete installation with a custom provisioner size",
			WorkflowStepsFunc: largeInstallationSizeLifecycleSteps,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			testWorkflowSteps := c.WorkflowStepsFunc(test.ClusterSuite, test.InstallationSuite)
			test.Workflow = workflow.NewWorkflow(testWorkflowSteps)
			test.Steps = testWorkflowSteps
			defer test.InstallationSuite.Cleanup(ctx)

			err = test.Run()
			checkTestError(t, test, err)
			time.Sleep(time.Second * 1)

			expectedEvents := workflow.GetExpectedEvents(test.Steps)
			err = test.EventsRecorder.VerifyInOrder(expectedEvents)
			checkTestError(t, test, err)
		})
	}
}

func checkTestError(t *testing.T, test *shared.Test, err error) {
	if err != nil {
		// Log errors with test logger for easier troubleshooting.
		test.Logger.WithError(err).Error("Test failure")
	}
	require.NoError(t, err)
}
