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

func TestValidatePGBouncerConfig(t *testing.T) {
	var testCases = []struct {
		description string
		config      *model.PGBouncerConfig
		valid       bool
	}{
		{
			"valid",
			NewPGBouncerConfig(5, 10, 5, 2000, 20, 10, 20, 0),
			true,
		},
		{
			"MaxDatabaseConnectionsPerPool is too low",
			NewPGBouncerConfig(5, 10, 5, 2000, 0, 10, 20, 0),
			false,
		},
		{
			"DefaultPoolSize is too low",
			NewPGBouncerConfig(5, 0, 5, 2000, 20, 10, 20, 0),
			false,
		},
		{
			"ServerResetQueryAlways is invalid",
			NewPGBouncerConfig(5, 10, 5, 2000, 20, 10, 20, 2),
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			if tc.valid {
				assert.NoError(t, tc.config.Validate())
			} else {
				assert.Error(t, tc.config.Validate())
			}
		})
	}
}

func TestGeneratePGBouncerBaseIni(t *testing.T) {
	config := NewPGBouncerConfig(5, 10, 54, 2000, 63, 11, 44, 0)
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
