// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
)

// ClusterInstallation is a single namespace within a cluster composing a potentially larger installation.
type ClusterInstallation struct {
	ID              string
	ClusterID       string
	InstallationID  string
	Namespace       string
	State           string
	CreateAt        int64
	DeleteAt        int64
	APISecurityLock bool
	LockAcquiredBy  *string
	LockAcquiredAt  int64
	IsActive        bool
}

// ClusterInstallationFilter describes the parameters used to constrain a set of cluster installations.
type ClusterInstallationFilter struct {
	Paging
	IDs            []string
	InstallationID string
	ClusterID      string
	IsActive       *bool
}

type ClusterInstallationStatus struct {
	InstallationFound bool   `json:"InstallationFound,omitempty"`
	Replicas          *int32 `json:"Replicas,omitempty"`
	TotalPod          *int32 `json:"TotalPod,omitempty"`
	RunningPod        *int32 `json:"RunningPod,omitempty"`
	ReadyPod          *int32 `json:"ReadyPod,omitempty"`
	StartedPod        *int32 `json:"StartedPod,omitempty"`
	ReadyLocalServer  *int32 `json:"ReadyLocalServer,omitempty"`
}

// MigrateClusterInstallationRequest describes the parameters used to compose migration request between two clusters.
type MigrateClusterInstallationRequest struct {
	InstallationID   string
	SourceClusterID  string
	TargetClusterID  string
	DNSSwitch        bool
	LockInstallation bool
}

const (
	// OperationTypeMigration is used for CIs migration from source to target cluster.
	OperationTypeMigration = "MigratingClusterInstallation"
	// OperationTypeDNS is used for DNS Switch from source to target cluster.
	OperationTypeDNS = "DNSSwitch"
	// OperationTypeSwitchClusterRoles is used for Switching the tags between source & target clusters.
	OperationTypeSwitchClusterRoles = "SwitchClusterRoles"
	// OperationTypeDeletingInActiveCIs is used for Deleting InActive ClusterInstallations from source clusters.
	OperationTypeDeletingInActiveCIs = "DeletingInActiveClusterInstallations"
)

// MigrateClusterInstallationResponse describes the summary of migration between two clusters.
type MigrateClusterInstallationResponse struct {
	SourceClusterID           string
	TargetClusterID           string
	Operation                 string
	TotalClusterInstallations int
}

// Clone returns a deep copy the cluster installation.
func (c *ClusterInstallation) Clone() *ClusterInstallation {
	var clone ClusterInstallation
	data, _ := json.Marshal(c)
	json.Unmarshal(data, &clone)

	return &clone
}

// IsDeleted returns whether the cluster installation was marked as deleted or not.
func (c *ClusterInstallation) IsDeleted() bool {
	return c.DeleteAt != 0
}

// ClusterInstallationFromReader decodes a json-encoded cluster installation from the given io.Reader.
func ClusterInstallationFromReader(reader io.Reader) (*ClusterInstallation, error) {
	clusterInstallation := ClusterInstallation{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&clusterInstallation)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &clusterInstallation, nil
}

// ClusterInstallationsFromReader decodes a json-encoded list of cluster installations from the given io.Reader.
func ClusterInstallationsFromReader(reader io.Reader) ([]*ClusterInstallation, error) {
	clusterInstallations := []*ClusterInstallation{}
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&clusterInstallations)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return clusterInstallations, nil
}

// ClusterInstallationConfigFromReader decodes a json-encoded config from the config io.Reader.
func ClusterInstallationConfigFromReader(reader io.Reader) (map[string]interface{}, error) {
	config := make(map[string]interface{})
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&config)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return config, nil
}

// MigrateClusterInstallationResponseFromReader decodes a json-encoded cluster migration from the given io.Reader.
func MigrateClusterInstallationResponseFromReader(reader io.Reader) (*MigrateClusterInstallationResponse, error) {
	migrateClusterInstallationResponse := MigrateClusterInstallationResponse{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&migrateClusterInstallationResponse)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &migrateClusterInstallationResponse, nil
}

func NewClusterInstallationStatusFromReader(reader io.Reader) (*ClusterInstallationStatus, error) {
	var status ClusterInstallationStatus
	err := json.NewDecoder(reader).Decode(&status)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode ClusterInstallationStatus")
	}

	return &status, nil
}
