// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// ExternalDatabaseConfig represents configuration for an externally managed
// database.
type ExternalDatabaseConfig struct {
	SecretName string
}

// ToJSON marshals database configuration to JSON if it is not nil.
func (cfg *ExternalDatabaseConfig) ToJSON() ([]byte, error) {
	if cfg == nil {
		return nil, nil
	}
	return json.Marshal(cfg)
}

// ExternalDatabaseRequest represents requested configuration of an external
// database.
type ExternalDatabaseRequest struct {
	SecretName string
}

// Validate validates an ExternalDatabaseRequest.
func (request *ExternalDatabaseRequest) Validate() error {
	if len(request.SecretName) == 0 {
		return errors.New("no external database secret name providied")
	}

	return nil
}

// ToDBConfig converts ExternalDatabaseRequest to ExternalDatabaseConfig if the
// database type is external.
func (request *ExternalDatabaseRequest) ToDBConfig(database string) *ExternalDatabaseConfig {
	if database != InstallationDatabaseExternal || request == nil {
		return nil
	}

	return &ExternalDatabaseConfig{
		SecretName: request.SecretName,
	}
}
