// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
)

const ProvisionerExternal = "external"

// ExternalClusterMetadata is the provisioner metadata stored in a model.Cluster
// for external clusters.
type ExternalClusterMetadata struct {
	Name       string
	SecretName string
	Version    string
	VPC        string   `json:"VPC,omitempty"`
	Warnings   []string `json:"Warnings,omitempty"`
}

// ClearWarnings clears the kops metadata warnings.
func (ecm *ExternalClusterMetadata) ClearWarnings() {
	ecm.Warnings = []string{}
}

// AddWarning adds a warning the kops metadata warning list.
func (ecm *ExternalClusterMetadata) AddWarning(warning string) {
	ecm.Warnings = append(ecm.Warnings, warning)
}

func (ecm *ExternalClusterMetadata) ApplyClusterImportRequest(importRequest *ImportClusterRequest) bool {
	ecm.SecretName = importRequest.ExternalClusterSecretName
	ecm.VPC = importRequest.VpcID
	return true
}

// NewExternalMetadata creates an instance of ExternalClusterMetadata given the
// raw provisioner metadata.
func NewExternalClusterMetadata(metadataBytes []byte) (*ExternalClusterMetadata, error) {
	// Check if length of metadata is 0 as opposed to if the value is nil. This
	// is done to avoid an issue encountered where the metadata value provided
	// had a length of 0, but had non-zero capacity.
	if len(metadataBytes) == 0 || string(metadataBytes) == "null" {
		return nil, nil
	}

	externalClusterMetadata := ExternalClusterMetadata{}
	err := json.Unmarshal(metadataBytes, &externalClusterMetadata)
	if err != nil {
		return nil, err
	}

	return &externalClusterMetadata, nil
}
