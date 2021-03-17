// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"

	"github.com/mattermost/rotator/rotator"
	"github.com/pkg/errors"
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
	RotatorRequest     *RotatorMetadata            `json:"RotatorRequest,omitempty"`
	Warnings           []string                    `json:"Warnings,omitempty"`
	Networking         string                      `json:"networking,omitempty"`
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
	Networking         string `json:"networking,omitempty"`
}

// RotatorMetadata is the metadata for the Rotator tool
type RotatorMetadata struct {
	Status *rotator.RotatorMetadata
	Config *RotatorConfig
}

// RotatorConfig is the config setup for the Rotator tool run
type RotatorConfig struct {
	UseRotator           *bool `json:"use-rotator,omitempty"`
	MaxScaling           *int  `json:"max-scaling,omitempty"`
	MaxDrainRetries      *int  `json:"max-drain-retries,omitempty"`
	EvictGracePeriod     *int  `json:"evict-grace-period,omitempty"`
	WaitBetweenRotations *int  `json:"wait-between-rotations,omitempty"`
	WaitBetweenDrains    *int  `json:"wait-between-drains,omitempty"`
}

// ValidateChangeRequest ensures that the ChangeRequest has at least one
// actionable value.
func (km *KopsMetadata) ValidateChangeRequest() error {
	if km.ChangeRequest == nil {
		return errors.New("the KopsMetadata ChangeRequest is nil")
	}

	if len(km.ChangeRequest.Version) == 0 &&
		len(km.ChangeRequest.AMI) == 0 &&
		len(km.ChangeRequest.MasterInstanceType) == 0 &&
		len(km.ChangeRequest.NodeInstanceType) == 0 &&
		km.MasterCount == 0 &&
		km.NodeMinCount == 0 &&
		km.NodeMaxCount == 0 {
		return errors.New("the KopsMetadata ChangeRequest has no change values set")
	}

	return nil
}

// ClearChangeRequest clears the kops metadata change request.
func (km *KopsMetadata) ClearChangeRequest() {
	km.ChangeRequest = nil
}

// ClearRotatorRequest clears the kops metadata rotator request.
func (km *KopsMetadata) ClearRotatorRequest() {
	km.RotatorRequest = nil
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
