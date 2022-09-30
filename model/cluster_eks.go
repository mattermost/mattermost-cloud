// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import "encoding/json"

type EKSMetadata struct {
	KubernetesVersion *string
	VPC               string
	Networking        string
	ClusterRoleARN    *string

	EKSNodeGroups EKSNodeGroups
}

type EKSNodeGroups map[string]EKSNodeGroup

type EKSNodeGroup struct {
	RoleARN       *string
	InstanceTypes []*string
	AMIVersion    *string
	DesiredSize   *int64
	MinSize       *int64
	MaxSize       *int64
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
