// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCachedKopsClient(t *testing.T) {
	logger := testlib.MakeLogger(t)
	provisioner := NewKopsProvisioner("s3statestore", "test", true, []string{}, []string{}, nil, logger, nil)

	// Using &kops.Cmd{} here because kops.New() checks for the binary in your
	// PATH which isn't needed for the test and fails in CI/CD.
	provisioner.kopsCache["test"] = &kops.Cmd{}

	t.Run("get cached client", func(t *testing.T) {
		cachedClient, err := provisioner.getCachedKopsClient("test", logger)
		require.NoError(t, err)
		assert.NotNil(t, cachedClient)
	})

	t.Run("get cached kubecfg", func(t *testing.T) {
		config, err := provisioner.getCachedKopsClusterKubecfg("test", logger)
		require.NoError(t, err)
		assert.NotEmpty(t, config)
	})

	t.Run("invalidate cache", func(t *testing.T) {
		err := provisioner.invalidateCachedKopsClient("test", logger)
		require.NoError(t, err)
		require.Nil(t, provisioner.kopsCache["test"])
	})

	t.Run("invalidate missing cache", func(t *testing.T) {
		err := provisioner.invalidateCachedKopsClient("test1", logger)
		require.Error(t, err)
	})

	provisioner.kopsCache["test"] = &kops.Cmd{}

	t.Run("invalidate cache on error; error is nil", func(t *testing.T) {
		var cacheError error
		provisioner.invalidateCachedKopsClientOnError(cacheError, "test", logger)
		require.NotNil(t, provisioner.kopsCache["test"])
	})

	t.Run("invalidate cache on error; error is not nil", func(t *testing.T) {
		cacheError := errors.New("not nil")
		provisioner.invalidateCachedKopsClientOnError(cacheError, "test", logger)
		require.Nil(t, provisioner.kopsCache["test"])
	})
}
