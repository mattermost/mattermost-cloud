// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//+build e2e

package dbmigration

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBMigration_Commit(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	t.Parallel()

	test, err := SetupDBMigrationCommitTest()
	require.NoError(t, err)

	if test.Cleanup {
		defer func() {
			err := test.Suite.InstallationSuite.Cleanup(context.Background())
			assert.NoError(t, err)
		}()
	}

	err = test.Run()
	assert.NoError(t, err)
}

func TestDBMigration_Rollback(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	t.Parallel()

	test, err := SetupDBMigrationRollbackTest()
	require.NoError(t, err)

	if test.Cleanup {
		defer func() {
			err := test.Suite.InstallationSuite.Cleanup(context.Background())
			assert.NoError(t, err)
		}()
	}

	err = test.Run()
	assert.NoError(t, err)
}
