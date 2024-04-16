// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model_test

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/util"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
)

func TestValidatePGBouncerConfig(t *testing.T) {
	var testCases = []struct {
		description string
		config      *model.PgBouncerConfig
		valid       bool
	}{
		{
			"valid",
			model.NewPgBouncerConfig(5, 10, 5, 2000, 20, 10, 20, 0),
			true,
		},
		{
			"MaxDatabaseConnectionsPerPool is too low",
			model.NewPgBouncerConfig(5, 10, 5, 2000, 0, 10, 20, 0),
			false,
		},
		{
			"DefaultPoolSize is too low",
			model.NewPgBouncerConfig(5, 0, 5, 2000, 20, 10, 20, 0),
			false,
		},
		{
			"ServerResetQueryAlways is invalid",
			model.NewPgBouncerConfig(5, 10, 5, 2000, 20, 10, 20, 2),
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

func TestPgBouncerPatch(t *testing.T) {
	var sizeTests = []struct {
		name     string
		config   *model.PgBouncerConfig
		patch    *model.PatchPgBouncerConfig
		expected *model.PgBouncerConfig
	}{
		{"nil",
			&model.PgBouncerConfig{MinPoolSize: 10},
			nil,
			&model.PgBouncerConfig{MinPoolSize: 10},
		},
		{"one value patched",
			&model.PgBouncerConfig{MinPoolSize: 10},
			&model.PatchPgBouncerConfig{MinPoolSize: util.IToP(11)},
			&model.PgBouncerConfig{MinPoolSize: 11},
		},
		{"full patch",
			&model.PgBouncerConfig{
				MinPoolSize:                   1,
				DefaultPoolSize:               1,
				ReservePoolSize:               1,
				MaxClientConnections:          1,
				MaxDatabaseConnectionsPerPool: 1,
				ServerIdleTimeout:             1,
				ServerLifetime:                1,
				ServerResetQueryAlways:        1,
			},
			&model.PatchPgBouncerConfig{
				MinPoolSize:                   util.IToP(2),
				DefaultPoolSize:               util.IToP(3),
				ReservePoolSize:               util.IToP(4),
				MaxClientConnections:          util.IToP(5),
				MaxDatabaseConnectionsPerPool: util.IToP(6),
				ServerIdleTimeout:             util.IToP(7),
				ServerLifetime:                util.IToP(8),
				ServerResetQueryAlways:        util.IToP(0),
			},
			&model.PgBouncerConfig{
				MinPoolSize:                   2,
				DefaultPoolSize:               3,
				ReservePoolSize:               4,
				MaxClientConnections:          5,
				MaxDatabaseConnectionsPerPool: 6,
				ServerIdleTimeout:             7,
				ServerLifetime:                8,
				ServerResetQueryAlways:        0,
			},
		},
	}

	for _, tt := range sizeTests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.patch != nil {
				assert.NoError(t, tt.patch.Validate())
			}
			tt.config.ApplyPatch(tt.patch)
			assert.Equal(t, tt.config, tt.expected)
		})
	}
}
