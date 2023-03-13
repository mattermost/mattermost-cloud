// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import "github.com/pkg/errors"

// PGBouncerConfig contains the configuration for the PGBouncer utility.
// //////////////////////////////////////////////////////////////////////////////
//   - MaxDatabaseConnectionsPerPool is the maximum number of connections per
//     logical database pool when using proxy databases.
//   - MinPoolSize is the minimum pool size.
//   - DefaultPoolSize is the default pool size per user.
//   - ReservePoolSize is the default pool size per user.
//   - MaxClientConnections is the maximum client connections.
//   - ServerIdleTimeout is the server idle timeout.
//   - ServerLifetime is the server lifetime.
//   - ServerResetQueryAlways is boolean 0 or 1 whether server_reset_query should
//     be run in all pooling modes.
//
// //////////////////////////////////////////////////////////////////////////////
type PGBouncerConfig struct {
	MinPoolSize                   int
	DefaultPoolSize               int
	ReservePoolSize               int
	MaxClientConnections          int
	MaxDatabaseConnectionsPerPool int
	ServerIdleTimeout             int
	ServerLifetime                int
	ServerResetQueryAlways        int
}

// Validate validates a PGBouncerConfig.
func (c *PGBouncerConfig) Validate() error {
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
