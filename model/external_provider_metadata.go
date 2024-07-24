// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import "encoding/json"

// ExternalProviderMetadata is the provider metadata stored from external
// clusters.
type ExternalProviderMetadata struct {
	HasAWSInfrastructure bool
}

func (epm *ExternalProviderMetadata) ApplyClusterImportRequest(importRequest *ImportClusterRequest) {
	epm.HasAWSInfrastructure = importRequest.VpcID != ""
}

// NewExternalProviderMetadata creates an instance of ExternalProviderMetadata
// given the raw provider metadata.
func NewExternalProviderMetadata(metadataBytes []byte) (*ExternalProviderMetadata, error) {
	if metadataBytes == nil || string(metadataBytes) == "null" {
		return nil, nil
	}

	var externalProviderMetadata ExternalProviderMetadata
	err := json.Unmarshal(metadataBytes, &externalProviderMetadata)
	if err != nil {
		return nil, err
	}

	return &externalProviderMetadata, nil
}
