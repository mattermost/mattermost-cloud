// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package pgbouncer

import (
	"fmt"
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratePGBouncerBaseIni(t *testing.T) {
	config := model.NewPgBouncerConfig(5, 10, 54, 2000, 63, 11, 44, 0)
	require.NoError(t, config.Validate())

	ini := generatePGBouncerBaseIni(config)
	assert.Contains(t, ini, "[pgbouncer]")

	// Most of the other values are integers so just spot check a few
	assert.Contains(t, ini, fmt.Sprintf("%d", config.MaxDatabaseConnectionsPerPool))
	assert.Contains(t, ini, fmt.Sprintf("%d", config.DefaultPoolSize))
	assert.Contains(t, ini, fmt.Sprintf("%d", config.ServerLifetime))
	assert.Contains(t, ini, fmt.Sprintf("%d", config.ServerIdleTimeout))
	assert.Contains(t, ini, fmt.Sprintf("%d", config.MinPoolSize))
}
