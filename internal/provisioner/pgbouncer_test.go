package provisioner

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePGBouncerConfig(t *testing.T) {
	var testCases = []struct {
		description string
		config      *PGBouncerConfig
		valid       bool
	}{
		{
			"valid",
			NewPGBouncerConfig("SELECT usename, passwd FROM tablename", 5, 10, 5, 2000, 20, 10, 20, 0),
			true,
		},
		{
			"no auth query",
			NewPGBouncerConfig("", 5, 10, 5, 2000, 20, 10, 20, 0),
			false,
		},
		{
			"MaxDatabaseConnectionsPerPool is too low",
			NewPGBouncerConfig("SELECT usename, passwd FROM tablename", 5, 10, 5, 2000, 0, 10, 20, 0),
			false,
		},
		{
			"DefaultPoolSize is too low",
			NewPGBouncerConfig("SELECT usename, passwd FROM tablename", 5, 0, 5, 2000, 20, 10, 20, 0),
			false,
		},
		{
			"ServerResetQueryAlways is invalid",
			NewPGBouncerConfig("SELECT usename, passwd FROM tablename", 5, 10, 5, 2000, 20, 10, 20, 2),
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
	config := NewPGBouncerConfig("SELECT usename, passwd FROM tablename", 5, 10, 54, 2000, 63, 11, 44, 0)
	require.NoError(t, config.Validate())

	ini := config.generatePGBouncerBaseIni()
	assert.Contains(t, ini, "[pgbouncer]")
	assert.Contains(t, ini, config.AuthQuery)

	// Most of the other values are integers so just spot check a few
	assert.Contains(t, ini, fmt.Sprintf("%d", config.MaxDatabaseConnectionsPerPool))
	assert.Contains(t, ini, fmt.Sprintf("%d", config.DefaultPoolSize))
	assert.Contains(t, ini, fmt.Sprintf("%d", config.ServerLifetime))
	assert.Contains(t, ini, fmt.Sprintf("%d", config.ServerIdleTimeout))
	assert.Contains(t, ini, fmt.Sprintf("%d", config.MinPoolSize))
}
