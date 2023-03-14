package model

import (
	"encoding/json"
	"github.com/pkg/errors"
)

const (
	ProvisionerEKS = "eks"
)

// EKSMetadata is metadata for EKS cluster and node groups.
type EKSMetadata struct {
	Name                  string
	Version               string
	AMI                   string
	VPC                   string
	Networking            string
	ClusterRoleARN        string
	NodeRoleARN           string
	MaxPodsPerNode        int64
	NodeInstanceType      string
	NodeMinCount          int64
	NodeMaxCount          int64
	NodeInstanceGroups    EKSInstanceGroupsMetadata
	ChangeRequest         *EKSMetadataRequestedState `json:"ChangeRequest,omitempty"`
	Warnings              []string                   `json:"Warnings,omitempty"`
	WorkerName            string
	LaunchTemplateVersion *int64
}

// EKSInstanceGroupsMetadata is a map of instance group names to their metadata.
type EKSInstanceGroupsMetadata map[string]EKSInstanceGroupMetadata

// EKSInstanceGroupMetadata is the metadata of an instance group.
type EKSInstanceGroupMetadata struct {
	NodeInstanceType string
	NodeMinCount     int64
	NodeMaxCount     int64
}

// EKSMetadataRequestedState is the requested state for eks metadata.
type EKSMetadataRequestedState struct {
	Version               string `json:"Version,omitempty"`
	AMI                   string `json:"AMI,omitempty"`
	NodeInstanceType      string `json:"NodeInstanceType,omitempty"`
	NodeMinCount          int64  `json:"NodeMinCount,omitempty"`
	NodeMaxCount          int64  `json:"NodeMaxCount,omitempty"`
	MaxPodsPerNode        int64  `json:"MaxPodsPerNode,omitempty"`
	Networking            string `json:"Networking,omitempty"`
	VPC                   string `json:"VPC,omitempty"`
	ClusterRoleARN        string `json:"ClusterRoleARN,omitempty"`
	NodeRoleARN           string `json:"NodeRoleARN,omitempty"`
	LaunchTemplateVersion *int64 `json:"LaunchTemplateVersion,omitempty"`
	WorkerName            string `json:"WorkerName,omitempty"`
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
		em.ChangeRequest = changes
	}

	return applied
}

func (em *EKSMetadata) ApplyClusterSizePatch(patchRequest *PatchClusterSizeRequest) bool {
	changes := &EKSMetadataRequestedState{}

	var applied bool
	if patchRequest.NodeInstanceType != nil && *patchRequest.NodeInstanceType != em.NodeInstanceType {
		applied = true
		changes.NodeInstanceType = *patchRequest.NodeInstanceType
	}
	if patchRequest.NodeMinCount != nil && *patchRequest.NodeMinCount != em.NodeMinCount {
		applied = true
		changes.NodeMinCount = *patchRequest.NodeMinCount
	}
	if patchRequest.NodeMaxCount != nil && *patchRequest.NodeMaxCount != em.NodeMaxCount {
		applied = true
		changes.NodeMaxCount = *patchRequest.NodeMaxCount
	}

	if applied {
		em.ChangeRequest = changes
	}

	return applied
}

// ValidateChangeRequest ensures that the ChangeRequest has at least one
// actionable value.
func (em *EKSMetadata) ValidateChangeRequest() error {
	if em.ChangeRequest == nil {
		return errors.New("the EKS Metadata ChangeRequest is nil")
	}

	if len(em.ChangeRequest.Version) == 0 &&
		len(em.ChangeRequest.AMI) == 0 &&
		len(em.ChangeRequest.NodeInstanceType) == 0 &&
		em.ChangeRequest.NodeMinCount == 0 &&
		em.ChangeRequest.NodeMaxCount == 0 &&
		em.ChangeRequest.MaxPodsPerNode == 0 {
		return errors.New("the EKS Metadata ChangeRequest has no change values set")
	}

	return nil
}

// ApplyChangeRequest applies change request values to the KopsMetadata that are
// not reflected by calling refreshKopsMetadata().
func (em *EKSMetadata) ApplyChangeRequest() {
	if em.ChangeRequest != nil {
		if em.ChangeRequest.AMI != "" {
			em.AMI = em.ChangeRequest.AMI
		}
		if em.ChangeRequest.MaxPodsPerNode != 0 {
			em.MaxPodsPerNode = em.ChangeRequest.MaxPodsPerNode
		}
		if em.ChangeRequest.VPC != "" {
			em.VPC = em.ChangeRequest.VPC
		}
		if em.ChangeRequest.Version != "" {
			em.Version = em.ChangeRequest.Version
		}
		if em.ChangeRequest.LaunchTemplateVersion != nil {
			em.LaunchTemplateVersion = em.ChangeRequest.LaunchTemplateVersion
		}
	}
}

func (em *EKSMetadata) GetCommonMetadata() ProvisionerMetadata {
	return ProvisionerMetadata{
		Name:             em.Name,
		Version:          em.Version,
		AMI:              em.AMI,
		NodeInstanceType: em.NodeInstanceType,
		NodeMinCount:     em.NodeMinCount,
		NodeMaxCount:     em.NodeMaxCount,
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
