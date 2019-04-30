package model

import "encoding/json"

// AWSMetadata is the provider metadata stored in a model.Cluster.
type AWSMetadata struct {
	Zones []string
}

// NewAWSMetadata creates an instance of AWSMetadata given the raw provider metadata.
func NewAWSMetadata(providerMetadata []byte) *AWSMetadata {
	awsMetadata := AWSMetadata{}

	if providerMetadata == nil {
		return &awsMetadata
	}

	_ = json.Unmarshal(providerMetadata, &awsMetadata)

	return &awsMetadata
}
