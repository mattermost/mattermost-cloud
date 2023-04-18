// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"

	"github.com/aws/smithy-go/ptr"
)

const (
	ProvisionerCrossplane = "crossplane"

	// defaultLaunchTemplateVersion is the default launch template version to use.
	defaultLaunchTemplateVersion = "2"

	// defaultKubernetesVersion is the default Kubernetes version to use.
	defaultKubernetesVersion = "1.23"

	// defaultInstanceType is the default AWS instance type to use.
	defaultInstanceType = "t3.large"

	// defaultNodeCount is the default number of nodes to use.
	defaultNodeCount = 1
)

type CrossplaneMetadata struct {
	ChangeRequest         *CrossplaneMetadataRequestedState
	Name                  string
	AccountID             string
	AMI                   string
	KubernetesVersion     string
	LaunchTemplateVersion *string
	PrivateSubnets        []string
	PublicSubnets         []string
	Region                string
	VPC                   string
	InstanceType          string
	NodeCount             int64
}

func (m *CrossplaneMetadata) ApplyClusterCreateRequest(createRequest *CreateClusterRequest) bool {
	m.ChangeRequest = &CrossplaneMetadataRequestedState{
		AMI:            createRequest.AMI,
		MaxPodsPerNode: createRequest.MaxPodsPerNode,
		Version:        createRequest.Version,
		VPC:            createRequest.VPC,
	}

	return true
}

func (m *CrossplaneMetadata) GetCommonMetadata() ProvisionerMetadata {
	return ProvisionerMetadata{
		Name:             m.Name,
		Version:          m.KubernetesVersion,
		AMI:              m.AMI,
		NodeInstanceType: m.InstanceType,
		NodeMinCount:     m.NodeCount,
		NodeMaxCount:     m.NodeCount,
		MaxPodsPerNode:   m.NodeCount,
		VPC:              m.VPC,
	}
}

func (m *CrossplaneMetadata) SetDefaults() {
	if m.LaunchTemplateVersion == nil {
		m.LaunchTemplateVersion = ptr.String(defaultLaunchTemplateVersion)
	}

	// Safeguard, should be set by the cluster creation request.
	if m.KubernetesVersion == "" {
		m.KubernetesVersion = defaultKubernetesVersion
	}

	if m.InstanceType == "" {
		m.InstanceType = defaultInstanceType
	}

	if m.NodeCount == 0 {
		m.NodeCount = defaultNodeCount
	}
}

// ApplyChangeRequest applies the change request to the metadata if the values are provided.
func (m *CrossplaneMetadata) ApplyChangeRequest() error {
	if m.ChangeRequest == nil {
		return nil
	}

	if m.ChangeRequest.AMI != "" {
		m.AMI = m.ChangeRequest.AMI
	}

	if m.ChangeRequest.Version != "" {
		m.KubernetesVersion = m.ChangeRequest.Version
	}

	if m.ChangeRequest.NodeCount != 0 {
		m.NodeCount = m.ChangeRequest.NodeCount
	}

	if m.ChangeRequest.VPC != "" {
		m.VPC = m.ChangeRequest.VPC
	}

	return nil
}

// CrossplaneMetadataRequestedState is the requested state for crossplane metadata.
type CrossplaneMetadataRequestedState struct {
	AMI                   string
	ClusterRoleARN        string
	LaunchTemplateVersion *string
	MaxPodsPerNode        int64
	NodeCount             int64
	Networking            string
	Version               string
	VPC                   string
}

// NewCrossplaneMetadataFromJSON creates an instance of CrossplaneMetadata given the raw database
// provisioner metadata.
func NewCrossplaneMetadataFromJSON(metadataJSON []byte) (*CrossplaneMetadata, error) {
	// Check if length of metadata is 0 as opposed to if the value is nil. This
	// is done to avoid an issue encountered where the metadata value provided
	// had a length of 0, but had non-zero capacity.
	if len(metadataJSON) == 0 || string(metadataJSON) == "null" {
		// TODO: remove "null" check after sqlite is gone.
		return nil, nil
	}

	metadata := CrossplaneMetadata{}
	err := json.Unmarshal(metadataJSON, &metadata)
	if err != nil {
		return nil, err
	}

	return &metadata, nil
}
