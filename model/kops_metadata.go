// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/mattermost/rotator/rotator"
	"github.com/pkg/errors"
)

// KopsMetadata is the provisioner metadata stored in a model.Cluster.
type KopsMetadata struct {
	Name                 string
	Version              string
	AMI                  string
	MasterInstanceType   string
	MasterCount          int64
	NodeInstanceType     string
	NodeMinCount         int64
	NodeMaxCount         int64
	MaxPodsPerNode       int64
	VPC                  string
	Networking           string
	MasterInstanceGroups KopsInstanceGroupsMetadata
	NodeInstanceGroups   KopsInstanceGroupsMetadata
	CustomInstanceGroups KopsInstanceGroupsMetadata  `json:"CustomInstanceGroups,omitempty"`
	ChangeRequest        *KopsMetadataRequestedState `json:"ChangeRequest,omitempty"`
	RotatorRequest       *RotatorMetadata            `json:"RotatorRequest,omitempty"`
	Warnings             []string                    `json:"Warnings,omitempty"`
}

// KopsInstanceGroupsMetadata is a map of instance group names to their metadata.
type KopsInstanceGroupsMetadata map[string]KopsInstanceGroupMetadata

// KopsInstanceGroupMetadata is the metadata of an instance group.
type KopsInstanceGroupMetadata struct {
	NodeInstanceType string
	NodeMinCount     int64
	NodeMaxCount     int64
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
	MaxPodsPerNode     int64  `json:"MaxPodsPerNode,omitempty"`
	Networking         string `json:"Networking,omitempty"`
	VPC                string `json:"VPC,omitempty"`
}

// RotatorMetadata is the metadata for the Rotator tool
type RotatorMetadata struct {
	Status *rotator.RotatorMetadata
	Config *RotatorConfig
}

// RotatorConfig is the config setup for the Rotator tool run
type RotatorConfig struct {
	UseRotator              *bool `json:"use-rotator,omitempty"`
	MaxScaling              *int  `json:"max-scaling,omitempty"`
	MaxDrainRetries         *int  `json:"max-drain-retries,omitempty"`
	EvictGracePeriod        *int  `json:"evict-grace-period,omitempty"`
	WaitBetweenRotations    *int  `json:"wait-between-rotations,omitempty"`
	WaitBetweenDrains       *int  `json:"wait-between-drains,omitempty"`
	WaitBetweenPodEvictions *int  `json:"wait-between-pod-evictions,omitempty"`
}

// Validate validates that the provided attributes for the rotator are set
func (r RotatorConfig) Validate() error {
	if r.UseRotator == nil {
		return errors.Errorf("rotator config use rotator should be set")
	}

	if *r.UseRotator {
		if r.EvictGracePeriod == nil {
			return errors.Errorf("rotator config evict grace period should be set")
		}
		if r.MaxDrainRetries == nil {
			return errors.Errorf("rotator config max drain retries should be set")
		}
		if r.MaxScaling == nil {
			return errors.Errorf("rotator config max scaling should be set")
		}
		if r.WaitBetweenRotations == nil {
			return errors.Errorf("rotator config wait between rotations should be set")
		}
		if r.WaitBetweenDrains == nil {
			return errors.Errorf("rotator config wait between drains should be set")
		}
		if r.WaitBetweenPodEvictions == nil {
			return errors.Errorf("rotator config wait between pod evictions should be set")
		}
	}
	return nil
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

// GetKopsResizeSetActionsFromChanges produces a set of kops set actions that
// should be applied to the instance groups from the provided change data.
func (km *KopsMetadata) GetKopsResizeSetActionsFromChanges(changes KopsInstanceGroupMetadata, igName string) []string {
	kopsSetActions := []string{}

	// There is a bit of complexity with updating min and max instancegroup
	// sizes. The maxSize always needs to be equal or larger than minSize
	// which means we need to apply the changes in a different order
	// depending on if the instance group is scaling up or down.
	if changes.NodeMaxCount >= km.NodeInstanceGroups[igName].NodeMaxCount {
		if changes.NodeMaxCount != km.NodeInstanceGroups[igName].NodeMaxCount {
			kopsSetActions = append(kopsSetActions, fmt.Sprintf("spec.maxSize=%d", changes.NodeMaxCount))
		}
		if changes.NodeMinCount != km.NodeInstanceGroups[igName].NodeMinCount {
			kopsSetActions = append(kopsSetActions, fmt.Sprintf("spec.minSize=%d", changes.NodeMinCount))
		}
	} else {
		if changes.NodeMinCount != km.NodeInstanceGroups[igName].NodeMinCount {
			kopsSetActions = append(kopsSetActions, fmt.Sprintf("spec.minSize=%d", changes.NodeMinCount))
		}
		if changes.NodeMaxCount != km.NodeInstanceGroups[igName].NodeMaxCount {
			kopsSetActions = append(kopsSetActions, fmt.Sprintf("spec.maxSize=%d", changes.NodeMaxCount))
		}
	}

	if changes.NodeInstanceType != km.NodeInstanceGroups[igName].NodeInstanceType {
		kopsSetActions = append(kopsSetActions, fmt.Sprintf("spec.machineType=%s", changes.NodeInstanceType))
	}

	return kopsSetActions
}

// GetWorkerNodesResizeChanges calculates instance group resizing based on the
// current ChangeRequest.
func (km *KopsMetadata) GetWorkerNodesResizeChanges() KopsInstanceGroupsMetadata {
	// Build a new change map.
	changes := make(KopsInstanceGroupsMetadata, len(km.NodeInstanceGroups))
	for k, v := range km.NodeInstanceGroups {
		changes[k] = v
	}

	// Update the AMI if specified.
	if len(km.ChangeRequest.NodeInstanceType) != 0 {
		for k, ig := range changes {
			ig.NodeInstanceType = km.ChangeRequest.NodeInstanceType
			changes[k] = ig
		}
	}

	if km.ChangeRequest.NodeMinCount == 0 {
		return changes
	}

	// Calculate new instance group sizes.
	difference := km.ChangeRequest.NodeMinCount - km.NodeMinCount

	if difference < 0 {
		getDecreasedWorkerNodesResizeChanges(changes, difference)
	}
	if difference > 0 {
		getIncreasedWorkerNodesResizeChanges(changes, difference)
	}

	return changes
}

func getIncreasedWorkerNodesResizeChanges(changes KopsInstanceGroupsMetadata, count int64) {
	orderedKeys := changes.getStableIterationOrder()
	currentBalanceCount := int64(1)

	for {
		for _, key := range orderedKeys {
			ig := changes[key]
			if ig.NodeMinCount >= currentBalanceCount {
				continue
			}

			changes[key] = KopsInstanceGroupMetadata{
				NodeInstanceType: ig.NodeInstanceType,
				NodeMinCount:     ig.NodeMinCount + 1,
				NodeMaxCount:     ig.NodeMinCount + 1,
			}

			count--
			if count == 0 {
				return
			}
		}
		currentBalanceCount++
	}
}

func getDecreasedWorkerNodesResizeChanges(changes KopsInstanceGroupsMetadata, count int64) {
	orderedKeys := changes.getStableIterationOrder()

	// For removing nodes, we want to work our way down starting with the end of
	// of list of IGs. This just seems to make a bit more sense.
	sort.Sort(sort.Reverse(sort.StringSlice(orderedKeys)))

	// Find the current highest IG node count to work backwards from.
	var currentBalanceCount int64
	for _, ig := range changes {
		if ig.NodeMinCount > currentBalanceCount {
			currentBalanceCount = ig.NodeMinCount
		}
	}

	for {
		for _, key := range orderedKeys {
			ig := changes[key]
			if ig.NodeMinCount <= currentBalanceCount {
				continue
			}

			changes[key] = KopsInstanceGroupMetadata{
				NodeInstanceType: ig.NodeInstanceType,
				NodeMinCount:     ig.NodeMinCount - 1,
				NodeMaxCount:     ig.NodeMinCount - 1,
			}

			count++
			if count == 0 {
				return
			}
		}
		currentBalanceCount--
	}
}

// ApplyChangeRequest applies change request values to the KopsMetadata that are
// not reflected by calling RefreshKopsMetadata().
func (km *KopsMetadata) ApplyChangeRequest() {
	if km.ChangeRequest != nil {
		if km.ChangeRequest.NodeMaxCount != 0 {
			km.NodeMaxCount = km.ChangeRequest.NodeMaxCount
		}
	}
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

func (igm *KopsInstanceGroupsMetadata) getStableIterationOrder() []string {
	keys := make([]string, 0, len(*igm))
	for k := range *igm {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys
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
