// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import "encoding/json"

// AWSMetadata is the provider metadata stored in a model.Cluster.
type AWSMetadata struct {
	Zones []string
}

// NewAWSMetadata creates an instance of AWSMetadata given the raw provider metadata.
func NewAWSMetadata(metadataBytes []byte) (*AWSMetadata, error) {
	if metadataBytes == nil || string(metadataBytes) == "null" {
		// TODO: remove "null" check after sqlite is gone.
		return nil, nil
	}

	var awsMetadata AWSMetadata
	err := json.Unmarshal(metadataBytes, &awsMetadata)
	if err != nil {
		return nil, err
	}

	return &awsMetadata, nil
}
