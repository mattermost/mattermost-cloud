package model

import (
	"encoding/json"
)

// KopsMetadata is the provisioner metadata stored in a model.Cluster.
type KopsMetadata struct {
	Name string
}

// NewKopsMetadata creates an instance of KopsMetadata given the raw provisioner metadata.
func NewKopsMetadata(provisionerMetadata []byte) (*KopsMetadata, error) {
	kopsMetadata := KopsMetadata{}

	if provisionerMetadata == nil {
		return &kopsMetadata, nil
	}

	err := json.Unmarshal(provisionerMetadata, &kopsMetadata)
	if err != nil {
		return nil, err
	}

	return &kopsMetadata, nil
}
