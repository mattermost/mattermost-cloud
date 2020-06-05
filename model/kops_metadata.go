package model

import (
	"encoding/json"
)

// KopsMetadata is the provisioner metadata stored in a model.Cluster.
type KopsMetadata struct {
	Name               string
	Version            string
	AMI                string
	MasterInstanceType string
	MasterCount        int64
	NodeInstanceType   string
	NodeMinCount       int64
	NodeMaxCount       int64
}

// NewKopsMetadata creates an instance of KopsMetadata given the raw provisioner metadata.
func NewKopsMetadata(metadataBytes []byte) (*KopsMetadata, error) {
	// Check if length of metadata is 0 as opposed to if the value is nil. This
	// is done to avoid an issue encountered where the metadata value provided
	// had a length of 0, but had non-zero capacity.
	if len(metadataBytes) == 0 || string(metadataBytes) == "null" {
		// TODO: remove "null" check after sqlite is gone.
		return nil, nil
	}

	kopsMetadata := KopsMetadata{}
	err := json.Unmarshal(metadataBytes, &kopsMetadata)
	if err != nil {
		return nil, err
	}

	return &kopsMetadata, nil
}
