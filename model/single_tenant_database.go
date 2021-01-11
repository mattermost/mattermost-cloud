// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/pkg/errors"
)

// SingleTenantDatabaseConfig represents configuration for the database when used
// in single tenant mode.
type SingleTenantDatabaseConfig struct {
	PrimaryInstanceType string
	ReplicaInstanceType string
	ReplicasCount       int
}

// ToJSON marshals database configuration to JSON if it is not nil.
func (cfg *SingleTenantDatabaseConfig) ToJSON() ([]byte, error) {
	if cfg == nil {
		return nil, nil
	}
	return json.Marshal(cfg)
}

// NewSingleTenantDatabaseConfigurationFromReader will create a SingleTenantDatabaseConfig
// from an io.Reader with JSON data.
func NewSingleTenantDatabaseConfigurationFromReader(reader io.Reader) (*SingleTenantDatabaseConfig, error) {
	singleTenantDBConfig := SingleTenantDatabaseConfig{}
	err := json.NewDecoder(reader).Decode(&singleTenantDBConfig)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode single tenant database configuration")
	}

	return &singleTenantDBConfig, nil
}

// SingleTenantDatabaseRequest represents requested configuration of single tenant database.
type SingleTenantDatabaseRequest struct {
	PrimaryInstanceType string
	ReplicaInstanceType string
	ReplicasCount       int
}

// NewSingleTenantDatabaseRequestFromReader will create a SingleTenantDatabaseRequest from an io.Reader with JSON data.
func NewSingleTenantDatabaseRequestFromReader(reader io.Reader) (*SingleTenantDatabaseRequest, error) {
	var singleTenantDBRequest SingleTenantDatabaseRequest
	err := json.NewDecoder(reader).Decode(&singleTenantDBRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode single tenant database request")
	}

	singleTenantDBRequest.SetDefaults()
	err = singleTenantDBRequest.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "single tenant database request failed validation")
	}

	return &singleTenantDBRequest, nil
}

// SetDefaults sets the default values for a single tenant database configuration request.
func (request *SingleTenantDatabaseRequest) SetDefaults() {
	if len(request.PrimaryInstanceType) == 0 {
		request.PrimaryInstanceType = "db.r5.large"
	}
	if len(request.ReplicaInstanceType) == 0 {
		request.ReplicaInstanceType = "db.r5.large"
	}
}

// Validate validates the values of single tenant database configuration request.
func (request *SingleTenantDatabaseRequest) Validate() error {
	if request.ReplicasCount < 0 || request.ReplicasCount > 15 {
		return fmt.Errorf("single tenant database replicas count must be between 0 and 15")
	}

	return nil
}

// ToDBConfig converts SingleTenantDatabaseRequest to SingleTenantDatabaseConfig
// if database type is single tenant.
func (request *SingleTenantDatabaseRequest) ToDBConfig(database string) *SingleTenantDatabaseConfig {
	if !IsSingleTenantRDS(database) || request == nil {
		return nil
	}

	return &SingleTenantDatabaseConfig{
		PrimaryInstanceType: request.PrimaryInstanceType,
		ReplicaInstanceType: request.ReplicaInstanceType,
		ReplicasCount:       request.ReplicasCount,
	}
}
