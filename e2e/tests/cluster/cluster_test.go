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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ClusterLifecycle(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	t.Parallel()

	test, err := SetupClusterLifecycleTest()
	require.NoError(t, err)
	if test.Cleanup {
		defer func() {
			err := test.ClusterSuite.DeleteCluster(context.Background())
			if err != nil {
				test.Logger.WithError(err).Error("Error cleaning up cluster")
			}
			err = test.InstallationSuite.Cleanup(context.Background())
			if err != nil {
				test.Logger.WithError(err).Error("Error cleaning up installation")
			}
		}()
	}
	// Always cleanup webhook
	defer func() {
		err := test.WebhookCleanup()
		assert.NoError(t, err)
	}()

	err = test.Run()
	assert.NoError(t, err)
}
