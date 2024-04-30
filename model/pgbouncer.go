// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/pkg/errors"
)

// PgBouncerConfig contains the configuration for the PGBouncer utility.
////////////////////////////////////////////////////////////////////////////////
//   - MaxDatabaseConnectionsPerPool is the maximum number of connections per
//     logical database pool when using proxy databases.
//   - MinPoolSize is the minimum pool size.
//   - DefaultPoolSize controls how many server connections to allow per
//     user/database pair.
//   - ReservePoolSize controls how many additional connections to allow to a
//     pool when the default_pool_size is exhausted
//   - MaxClientConnections is the maximum client connections.
//   - ServerIdleTimeout is the server idle timeout.
//   - ServerLifetime is the server lifetime.
//   - ServerResetQueryAlways is boolean 0 or 1 whether server_reset_query
//     should be run in all pooling modes.
////////////////////////////////////////////////////////////////////////////////

const (
	PgBouncerDefaultMinPoolSize                   int64 = 1
	PgBouncerDefaultDefaultPoolSize               int64 = 5
	PgBouncerDefaultReservePoolSize               int64 = 10
	PgBouncerDefaultMaxClientConnections          int64 = 25000
	PgBouncerDefaultMaxDatabaseConnectionsPerPool int64 = 50
	PgBouncerDefaultServerIdleTimeout             int64 = 30
	PgBouncerDefaultServerLifetime                int64 = 300
	PgBouncerDefaultServerResetQueryAlways        int64 = 0
)

type PgBouncerConfig struct {
	MinPoolSize                   int64
	DefaultPoolSize               int64
	ReservePoolSize               int64
	MaxClientConnections          int64
	MaxDatabaseConnectionsPerPool int64
	ServerIdleTimeout             int64
	ServerLifetime                int64
	ServerResetQueryAlways        int64
}

// NewPgBouncerConfig returns a new PgBouncerConfig with the provided configuration.
func NewPgBouncerConfig(minPoolSize, defaultPoolSize, reservePoolSize, maxClientConnections, maxDatabaseConnectionsPerPool, serverIdleTimeout, serverLifetime, serverResetQueryAlways int64) *PgBouncerConfig {
	return &PgBouncerConfig{
		MinPoolSize:                   minPoolSize,
		DefaultPoolSize:               defaultPoolSize,
		ReservePoolSize:               reservePoolSize,
		MaxClientConnections:          maxClientConnections,
		MaxDatabaseConnectionsPerPool: maxDatabaseConnectionsPerPool,
		ServerIdleTimeout:             serverIdleTimeout,
		ServerLifetime:                serverLifetime,
		ServerResetQueryAlways:        serverResetQueryAlways,
	}
}

// NewPgBouncerConfig returns a new PgBouncerConfig with the default values.
func NewDefaultPgBouncerConfig() *PgBouncerConfig {
	c := &PgBouncerConfig{}
	c.SetDefaults()
	return c
}

// SetDefaults sets default values for empty PgBouncerConfig values.
func (c *PgBouncerConfig) SetDefaults() {
	if c.MinPoolSize == 0 {
		c.MinPoolSize = PgBouncerDefaultMinPoolSize
	}
	if c.DefaultPoolSize == 0 {
		c.DefaultPoolSize = PgBouncerDefaultDefaultPoolSize
	}
	if c.ReservePoolSize == 0 {
		c.ReservePoolSize = PgBouncerDefaultReservePoolSize
	}
	if c.MaxClientConnections == 0 {
		c.MaxClientConnections = PgBouncerDefaultMaxClientConnections
	}
	if c.MaxDatabaseConnectionsPerPool == 0 {
		c.MaxDatabaseConnectionsPerPool = PgBouncerDefaultMaxDatabaseConnectionsPerPool
	}
	if c.ServerIdleTimeout == 0 {
		c.ServerIdleTimeout = PgBouncerDefaultServerIdleTimeout
	}
	if c.ServerLifetime == 0 {
		c.ServerLifetime = PgBouncerDefaultServerLifetime
	}
	// ServerResetQueryAlways default is already 0
}

// Validate validates a PgBouncerConfig.
func (c *PgBouncerConfig) Validate() error {
	if c.MaxDatabaseConnectionsPerPool < 1 {
		return errors.New("MaxDatabaseConnectionsPerPool must be 1 or greater")
	}
	if c.DefaultPoolSize < 1 {
		return errors.New("DefaultPoolSize must be 1 or greater")
	}
	if c.ServerResetQueryAlways != 0 && c.ServerResetQueryAlways != 1 {
		return errors.New("ServerResetQueryAlways must be 0 or 1")
	}

	return nil
}

func (c *PgBouncerConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *PgBouncerConfig) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return errors.New("Could not assert type of PgBouncerConfig")
	}

	var i PgBouncerConfig
	err := json.Unmarshal(source, &i)
	if err != nil {
		return err
	}
	*c = i
	return nil
}

func (c *PgBouncerConfig) FromJSONString(PgBouncerConfigStr string) (*PgBouncerConfig, error) {
	// Unmarshal the JSON into an PgBouncerConfig struct.
	var pgBouncerConfig PgBouncerConfig
	err := json.Unmarshal([]byte(PgBouncerConfigStr), &pgBouncerConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parses JSON")
	}
	return &pgBouncerConfig, nil
}

// ApplyPatch applies patch values to PgBouncerConfig.
func (c *PgBouncerConfig) ApplyPatch(p *PatchPgBouncerConfig) {
	if p == nil {
		return
	}

	if p.MinPoolSize != nil {
		c.MinPoolSize = *p.MinPoolSize
	}
	if p.DefaultPoolSize != nil {
		c.DefaultPoolSize = *p.DefaultPoolSize
	}
	if p.ReservePoolSize != nil {
		c.ReservePoolSize = *p.ReservePoolSize
	}
	if p.MaxClientConnections != nil {
		c.MaxClientConnections = *p.MaxClientConnections
	}
	if p.MaxDatabaseConnectionsPerPool != nil {
		c.MaxDatabaseConnectionsPerPool = *p.MaxDatabaseConnectionsPerPool
	}
	if p.ServerIdleTimeout != nil {
		c.ServerIdleTimeout = *p.ServerIdleTimeout
	}
	if p.ServerLifetime != nil {
		c.ServerLifetime = *p.ServerLifetime
	}
	if p.ServerResetQueryAlways != nil {
		c.ServerResetQueryAlways = *p.ServerResetQueryAlways
	}
}

type PatchPgBouncerConfig struct {
	MinPoolSize                   *int64
	DefaultPoolSize               *int64
	ReservePoolSize               *int64
	MaxClientConnections          *int64
	MaxDatabaseConnectionsPerPool *int64
	ServerIdleTimeout             *int64
	ServerLifetime                *int64
	ServerResetQueryAlways        *int64
}

func (c *PatchPgBouncerConfig) Validate() error {
	if c.MaxDatabaseConnectionsPerPool != nil && *c.MaxDatabaseConnectionsPerPool < 1 {
		return errors.New("MaxDatabaseConnectionsPerPool must be 1 or greater")
	}
	if c.DefaultPoolSize != nil && *c.DefaultPoolSize < 1 {
		return errors.New("DefaultPoolSize must be 1 or greater")
	}
	if c.ServerResetQueryAlways != nil && *c.ServerResetQueryAlways != 0 && *c.ServerResetQueryAlways != 1 {
		return errors.New("ServerResetQueryAlways must be 0 or 1")
	}

	return nil
}
