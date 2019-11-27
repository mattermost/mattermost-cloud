package model

import (
	"encoding/json"
)

// KopsMetadata is the provisioner metadata stored in a model.Cluster.
type KopsMetadata struct {
	Name    string
	Version string
	AMI     string
}

// NewKopsMetadata creates an instance of KopsMetadata given the raw provisioner metadata.
func NewKopsMetadata(provisionerMetadata []byte) (*KopsMetadata, error) {
	kopsMetadata := KopsMetadata{}

	// Check if length of metadata is 0 as opposed to if the value is nil. This
	// is done to avoid an issue encountered where the metadata value provided
	// had a length of 0, but had non-zero capacity.
	if len(provisionerMetadata) == 0 {
		return &kopsMetadata, nil
	}

	err := json.Unmarshal(provisionerMetadata, &kopsMetadata)
	if err != nil {
		return nil, err
	}

	return &kopsMetadata, nil
}
