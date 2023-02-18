// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"

	"github.com/pkg/errors"
)

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

// ClusterResources is a collection of AWS resources that will be used to create a cluster.
type ClusterResources struct {
	VpcID                  string
	VpcCIDR                string
	PrivateSubnetIDs       []string
	PublicSubnetsIDs       []string
	MasterSecurityGroupIDs []string
	WorkerSecurityGroupIDs []string
	CallsSecurityGroupIDs  []string
}

// IsValid returns whether ClusterResources is valid or not.
func (cr *ClusterResources) IsValid() error {
	if cr.VpcID == "" {
		return errors.New("vpc ID is empty")
	}
	if len(cr.PrivateSubnetIDs) == 0 {
		return errors.New("private subnet list is empty")
	}
	if len(cr.PublicSubnetsIDs) == 0 {
		return errors.New("public subnet list is empty")
	}
	if len(cr.MasterSecurityGroupIDs) == 0 {
		return errors.New("master security group list is empty")
	}
	if len(cr.WorkerSecurityGroupIDs) == 0 {
		return errors.New("worker security group list is empty")
	}
	if len(cr.CallsSecurityGroupIDs) == 0 {
		return errors.New("calls security group list is empty")
	}

	return nil
}
