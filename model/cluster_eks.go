// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import "encoding/json"

// EKSMetadata is metadata for EKS cluster and node groups.
type EKSMetadata struct {
	KubernetesVersion *string
	VPC               string
	Networking        string
	ClusterRoleARN    *string

	EKSNodeGroups EKSNodeGroups
}

// EKSNodeGroups maps node group name to configuration.
type EKSNodeGroups map[string]EKSNodeGroup

// EKSNodeGroup is node group configuration.
type EKSNodeGroup struct {
	RoleARN       *string
	InstanceTypes []string
	AMIVersion    *string
	DesiredSize   *int32
	MinSize       *int32
	MaxSize       *int32
}

// NewEKSMetadata creates an instance of EKSMetadata given the raw provisioner metadata.
func NewEKSMetadata(metadataBytes []byte) (*EKSMetadata, error) {
	// Check if length of metadata is 0 as opposed to if the value is nil. This
	// is done to avoid an issue encountered where the metadata value provided
	// had a length of 0, but had non-zero capacity.
	if len(metadataBytes) == 0 || string(metadataBytes) == "null" {
		// TODO: remove "null" check after sqlite is gone.
		return nil, nil
	}

	eksMetadata := EKSMetadata{}
	err := json.Unmarshal(metadataBytes, &eksMetadata)
	if err != nil {
		return nil, err
	}

	return &eksMetadata, nil
}
