// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

const (
	ProvisionerEKS  = "eks"
	NodeGroupWorker = "worker"
)

// EKSMetadata is metadata for EKS cluster and node groups.
type EKSMetadata struct {
	Name           string
	Version        string
	AMI            string
	VPC            string
	Networking     string
	ClusterRoleARN string
	NodeRoleARN    string
	MaxPodsPerNode int64
	NodeGroups     map[string]NodeGroupMetadata
	ChangeRequest  *EKSMetadataRequestedState `json:"ChangeRequest,omitempty"`
	Warnings       []string                   `json:"Warnings,omitempty"`
}

// NodeGroupMetadata is the metadata of an instance group.
type NodeGroupMetadata struct {
	Name              string
	Type              string `json:"Type,omitempty"`
	InstanceType      string `json:"InstanceType,omitempty"`
	MinCount          int64  `json:"MinCount,omitempty"`
	MaxCount          int64  `json:"MaxCount,omitempty"`
	WithPublicSubnet  bool   `json:"WithPublicSubnet,omitempty"`
	WithSecurityGroup bool   `json:"WithSecurityGroup,omitempty"`
}

// EKSMetadataRequestedState is the requested state for eks metadata.
type EKSMetadataRequestedState struct {
	Version        string                       `json:"Version,omitempty"`
	AMI            string                       `json:"AMI,omitempty"`
	MaxPodsPerNode int64                        `json:"MaxPodsPerNode,omitempty"`
	Networking     string                       `json:"Networking,omitempty"`
	VPC            string                       `json:"VPC,omitempty"`
	ClusterRoleARN string                       `json:"ClusterRoleARN,omitempty"`
	NodeRoleARN    string                       `json:"NodeRoleARN,omitempty"`
	NodeGroups     map[string]NodeGroupMetadata `json:"NodeGroups,omitempty"`
}

// CopyMissingFieldsFrom copy empty fields from the given NodeGroupMetadata to the current metadata.
func (ng *NodeGroupMetadata) CopyMissingFieldsFrom(other NodeGroupMetadata) {
	if len(ng.Type) == 0 {
		ng.Type = other.Type
	}
	if ng.InstanceType == "" {
		ng.InstanceType = other.InstanceType
	}
	if ng.MinCount == 0 {
		ng.MinCount = other.MinCount
	}
	if ng.MaxCount == 0 {
		ng.MaxCount = other.MaxCount
	}
	if !ng.WithPublicSubnet {
		ng.WithPublicSubnet = other.WithPublicSubnet
	}
	if !ng.WithSecurityGroup {
		ng.WithSecurityGroup = other.WithSecurityGroup
	}
}

func (em *EKSMetadata) ApplyClusterCreateRequest(createRequest *CreateClusterRequest) bool {

	em.ChangeRequest = &EKSMetadataRequestedState{
		Version:        createRequest.Version,
		AMI:            createRequest.AMI,
		MaxPodsPerNode: createRequest.MaxPodsPerNode,
		VPC:            createRequest.VPC,
		ClusterRoleARN: createRequest.ClusterRoleARN,
		NodeRoleARN:    createRequest.NodeRoleARN,
		NodeGroups:     map[string]NodeGroupMetadata{},
	}

	nodeGroups := createRequest.AdditionalNodeGroups
	if nodeGroups == nil {
		nodeGroups = map[string]NodeGroupMetadata{}
	}

	nodeGroups[NodeGroupWorker] = NodeGroupMetadata{
		InstanceType: createRequest.NodeInstanceType,
		MinCount:     createRequest.NodeMinCount,
		MaxCount:     createRequest.NodeMaxCount,
	}

	for _, ng := range createRequest.NodeGroupWithPublicSubnet {
		nodeGroup := nodeGroups[ng]
		nodeGroup.WithPublicSubnet = true
		nodeGroups[ng] = nodeGroup
	}

	for _, ng := range createRequest.NodeGroupWithSecurityGroup {
		nodeGroup := nodeGroups[ng]
		nodeGroup.WithSecurityGroup = true
		nodeGroups[ng] = nodeGroup
	}

	for name, ng := range nodeGroups {
		em.ChangeRequest.NodeGroups[name] = NodeGroupMetadata{
			Name:              fmt.Sprintf("%s-%s", name, NewNodeGroupSuffix()),
			Type:              name,
			InstanceType:      ng.InstanceType,
			MinCount:          ng.MinCount,
			MaxCount:          ng.MaxCount,
			WithPublicSubnet:  ng.WithPublicSubnet,
			WithSecurityGroup: ng.WithSecurityGroup,
		}
	}

	em.NodeGroups = map[string]NodeGroupMetadata{}

	return true
}

// ApplyUpgradePatch applies the patch to the given cluster's metadata.
func (em *EKSMetadata) ApplyUpgradePatch(patchRequest *PatchUpgradeClusterRequest) bool {
	changes := &EKSMetadataRequestedState{}

	var applied bool
	if patchRequest.Version != nil && *patchRequest.Version != em.Version {
		applied = true
		changes.Version = *patchRequest.Version
	}
	if patchRequest.AMI != nil && *patchRequest.AMI != em.AMI {
		applied = true
		changes.AMI = *patchRequest.AMI
	}
	if patchRequest.MaxPodsPerNode != nil && *patchRequest.MaxPodsPerNode != em.MaxPodsPerNode {
		applied = true
		changes.MaxPodsPerNode = *patchRequest.MaxPodsPerNode
	}

	if applied {
		changes.NodeGroups = map[string]NodeGroupMetadata{}
		for ng := range em.NodeGroups {
			changes.NodeGroups[ng] = NodeGroupMetadata{
				Name: fmt.Sprintf("%s-%s", ng, NewNodeGroupSuffix()),
			}
		}
		em.ChangeRequest = changes
	}

	return applied
}

func (em *EKSMetadata) ValidateClusterSizePatch(patchRequest *PatchClusterSizeRequest) error {
	nodeGroups := patchRequest.NodeGroups

	if len(em.NodeGroups) == 0 {
		return errors.New("no nodegroups available to resize")
	}

	if len(nodeGroups) == 0 {
		if len(em.NodeGroups) > 1 {
			return errors.New("must specify nodegroups to resize")
		}
		for ng := range em.NodeGroups {
			nodeGroups = append(nodeGroups, ng)
		}
	}

	for _, ngToResize := range nodeGroups {
		if _, f := em.NodeGroups[ngToResize]; !f {
			return errors.Errorf("nodegroup %s not found to resize", ngToResize)
		}
	}

	if patchRequest.NodeMinCount != nil && patchRequest.NodeMaxCount != nil {
		if *patchRequest.NodeMinCount > *patchRequest.NodeMaxCount {
			return errors.New("min node count cannot be greater than max node count")
		}
		return nil
	}

	if patchRequest.NodeMinCount != nil {
		for _, ngToResize := range nodeGroups {
			ng := em.NodeGroups[ngToResize]
			nodeMaxCount := ng.MaxCount
			if *patchRequest.NodeMinCount > nodeMaxCount {
				return errors.New("resize patch would set min node count higher than max node count")
			}
		}
	}

	if patchRequest.NodeMaxCount != nil {
		for _, ngToResize := range nodeGroups {
			ng := em.NodeGroups[ngToResize]
			nodeMinCount := ng.MinCount
			if *patchRequest.NodeMaxCount < nodeMinCount {
				return errors.New("resize patch would set max node count lower than min node count")
			}
		}
	}

	return nil
}

func (em *EKSMetadata) ApplyClusterSizePatch(patchRequest *PatchClusterSizeRequest) bool {
	changes := &EKSMetadataRequestedState{
		NodeGroups: map[string]NodeGroupMetadata{},
	}

	var applied bool

	nodeGroupsMeta := patchRequest.NodeGroups
	if len(nodeGroupsMeta) == 0 {
		for ngPrefix := range em.NodeGroups {
			nodeGroupsMeta = append(nodeGroupsMeta, ngPrefix)
		}
	}

	for _, ng := range nodeGroupsMeta {
		ngChangeRequest := NodeGroupMetadata{
			Name: fmt.Sprintf("%s-%s", ng, NewNodeGroupSuffix()),
		}
		if patchRequest.NodeInstanceType != nil {
			applied = true
			ngChangeRequest.InstanceType = *patchRequest.NodeInstanceType
		}
		if patchRequest.NodeMinCount != nil {
			applied = true
			ngChangeRequest.MinCount = *patchRequest.NodeMinCount
		}
		if patchRequest.NodeMaxCount != nil {
			applied = true
			ngChangeRequest.MaxCount = *patchRequest.NodeMaxCount
		}

		changes.NodeGroups[ng] = ngChangeRequest
	}

	if applied {
		em.ChangeRequest = changes
	}

	return applied
}

// ValidateChangeRequest ensures that the ChangeRequest has at least one
// actionable value.
func (em *EKSMetadata) ValidateChangeRequest() error {
	changeRequest := em.ChangeRequest
	if changeRequest == nil {
		return errors.New("the EKS Metadata ChangeRequest is nil")
	}

	changeAllowed := false
	if len(changeRequest.Version) != 0 || len(changeRequest.AMI) != 0 || changeRequest.MaxPodsPerNode != 0 {
		changeAllowed = true
	}

	if changeAllowed {
		return nil
	}

	for _, ng := range changeRequest.NodeGroups {
		if len(ng.InstanceType) != 0 || ng.MinCount != 0 || ng.MaxCount != 0 {
			changeAllowed = true
			break
		}
	}

	if !changeAllowed {
		return errors.New("the EKS Metadata ChangeRequest has no change values set")
	}

	return nil
}

// ApplyChangeRequest applies change request values to the KopsMetadata that are
// not reflected by calling refreshKopsMetadata().
func (em *EKSMetadata) ApplyChangeRequest() {
}

// ValidateNodegroupsCreateRequest ensures that the nodegroups to create do not
// already exist.
func (em *EKSMetadata) ValidateNodegroupsCreateRequest(nodegroups map[string]NodeGroupMetadata) error {
	if len(nodegroups) == 0 {
		return errors.New("must specify at least one nodegroup to create")
	}

	for ng := range nodegroups {
		if _, f := em.NodeGroups[ng]; f {
			return errors.Errorf("nodegroup %s already exists", ng)
		}
	}

	return nil
}

// ValidateNodegroupDeleteRequest ensures that the nodegroup to delete exists.
func (em *EKSMetadata) ValidateNodegroupDeleteRequest(nodegroup string) error {
	if _, f := em.NodeGroups[nodegroup]; !f {
		return errors.Errorf("nodegroup %s not found to delete", nodegroup)
	}

	return nil
}

// ApplyNodegroupsCreateRequest applies the nodegroups to create to the
// KopsMetadata.
func (em *EKSMetadata) ApplyNodegroupsCreateRequest(request *CreateNodegroupsRequest) {
	em.ChangeRequest = &EKSMetadataRequestedState{
		NodeGroups: map[string]NodeGroupMetadata{},
	}

	nodeGroups := request.Nodegroups

	for _, ng := range request.NodeGroupWithPublicSubnet {
		nodeGroup := nodeGroups[ng]
		nodeGroup.WithPublicSubnet = true
		nodeGroups[ng] = nodeGroup
	}

	for _, ng := range request.NodeGroupWithSecurityGroup {
		nodeGroup := nodeGroups[ng]
		nodeGroup.WithSecurityGroup = true
		nodeGroups[ng] = nodeGroup
	}

	for name, ng := range nodeGroups {
		em.ChangeRequest.NodeGroups[name] = NodeGroupMetadata{
			Name:              fmt.Sprintf("%s-%s", name, NewNodeGroupSuffix()),
			Type:              name,
			InstanceType:      ng.InstanceType,
			MinCount:          ng.MinCount,
			MaxCount:          ng.MaxCount,
			WithPublicSubnet:  ng.WithPublicSubnet,
			WithSecurityGroup: ng.WithSecurityGroup,
		}
	}

}

// ApplyNodegroupDeleteRequest applies the nodegroup to delete to the
// KopsMetadata.
func (em *EKSMetadata) ApplyNodegroupDeleteRequest(nodegroup string) {

	if em.NodeGroups == nil {
		return
	}

	em.ChangeRequest = &EKSMetadataRequestedState{
		NodeGroups: map[string]NodeGroupMetadata{
			nodegroup: {
				Name: em.NodeGroups[nodegroup].Name,
			},
		},
	}
}

func (em *EKSMetadata) GetCommonMetadata() ProvisionerMetadata {
	workerNodeGroup := em.NodeGroups[NodeGroupWorker]
	return ProvisionerMetadata{
		Name:             em.Name,
		Version:          em.Version,
		AMI:              em.AMI,
		NodeInstanceType: workerNodeGroup.InstanceType,
		NodeMinCount:     workerNodeGroup.MinCount,
		NodeMaxCount:     workerNodeGroup.MaxCount,
		MaxPodsPerNode:   em.MaxPodsPerNode,
		VPC:              em.VPC,
		Networking:       em.Networking,
	}
}

// ClearChangeRequest clears the kops metadata change request.
func (em *EKSMetadata) ClearChangeRequest() {
	em.ChangeRequest = nil
}

// ClearWarnings clears the kops metadata warnings.
func (em *EKSMetadata) ClearWarnings() {
	em.Warnings = []string{}
}

// NewEKSMetadata creates an instance of EKSMetadata given the raw provisioner metadata.
func NewEKSMetadata(metadataBytes []byte) (*EKSMetadata, error) {
	// Check if length of metadata is 0 as opposed to if the value is nil. This
	// is done to avoid an issue encountered where the metadata value provided
	// had a length of 0, but had non-zero capacity.
	if len(metadataBytes) == 0 || string(metadataBytes) == "null" {
		return nil, nil
	}

	eksMetadata := EKSMetadata{}
	err := json.Unmarshal(metadataBytes, &eksMetadata)
	if err != nil {
		return nil, err
	}

	return &eksMetadata, nil
}
