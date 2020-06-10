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
	ChangeRequest      *KopsMetadataRequestedState `json:"ChangeRequest,omitempty"`
	Warnings           []string                    `json:"Warnings,omitempty"`
}

// KopsMetadataRequestedState is the requested state for kops metadata.
type KopsMetadataRequestedState struct {
	Version            string `json:"Version,omitempty"`
	AMI                string `json:"AMI,omitempty"`
	MasterInstanceType string `json:"MasterInstanceType,omitempty"`
	MasterCount        int64  `json:"MasterCount,omitempty"`
	NodeInstanceType   string `json:"NodeInstanceType,omitempty"`
	NodeMinCount       int64  `json:"NodeMinCount,omitempty"`
	NodeMaxCount       int64  `json:"NodeMaxCount,omitempty"`
}

// ClearChangeRequest clears the kops metadata change request.
func (km *KopsMetadata) ClearChangeRequest() {
	km.ChangeRequest = nil
}

// ClearWarnings clears the kops metadata warnings.
func (km *KopsMetadata) ClearWarnings() {
	km.Warnings = []string{}
}

// AddWarning adds a warning the kops metadata warning list.
func (km *KopsMetadata) AddWarning(warning string) {
	km.Warnings = append(km.Warnings, warning)
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
