package model

import "encoding/json"

// AWSMetadata is the provider metadata stored in a model.Cluster.
type AWSMetadata struct {
	Zones []string
}

// NewAWSMetadata creates an instance of AWSMetadata given the raw provider metadata.
func NewAWSMetadata(providerMetadata []byte) (*AWSMetadata, error) {
	awsMetadata := AWSMetadata{}

	if providerMetadata == nil {
		return &awsMetadata, nil
	}

	err := json.Unmarshal(providerMetadata, &awsMetadata)
	if err != nil {
		return nil, err
	}

	return &awsMetadata, nil
}
