// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package kops

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

// InstanceGroup is a kops instance group.
type InstanceGroup struct {
	Metadata InstanceGroupMetadata `json:"metadata"`
	Spec     InstanceGroupSpec     `json:"spec"`
}

// InstanceGroupMetadata is the metadata of a kops instance group.
type InstanceGroupMetadata struct {
	Name string `json:"name"`
}

// InstanceGroupSpec is the spec of a kops instance group.
type InstanceGroupSpec struct {
	Role        string `json:"role"`
	Image       string `json:"image"`
	MachineType string `json:"machineType"`
	MinSize     int64  `json:"minSize"`
	MaxSize     int64  `json:"maxSize"`
}

// UpdateMetadata updates KopsMetadata with the current values from kops state
// store. This can be a bit tricky. We are attempting to correlate multiple kops
// instance groups into a simplified set of metadata information. To do so, we
// assume and check the following:
// - There is one worker node instance group.
// - There is one or more master instance groups.
// - All of the cluster hosts are running the same AMI.
// - All of the master nodes are running the same instance type.
// Note:
// If any violations are found, we don't return an error as that is beyond the
// scope of updating the metadata. Instead, warnings for each violation are
// returned and stored.
func (c *Cmd) UpdateMetadata(metadata *model.KopsMetadata) error {
	instanceGroups, err := c.GetInstanceGroupsJSON(metadata.Name)
	if err != nil {
		return err
	}

	metadata.MasterInstanceGroups = make(model.KopsInstanceGroupsMetadata)
	metadata.NodeInstanceGroups = make(model.KopsInstanceGroupsMetadata)
	metadata.CustomInstanceGroups = make(model.KopsInstanceGroupsMetadata)

	var masterIGCount, nodeIGCount, nodeMinCount, nodeMaxCount int64
	var masterMachineType, AMI string
	for _, ig := range instanceGroups {
		switch ig.Spec.Role {
		case "Master":
			if AMI == "" {
				AMI = ig.Spec.Image
			} else if AMI != ig.Spec.Image {
				warning := fmt.Sprintf("Expected all hosts to be running same AMI, but instance group %s has AMI %s", ig.Metadata.Name, ig.Spec.Image)
				metadata.AddWarning(warning)
				c.logger.WithField("kops-metadata-error", warning).Warn("Encountered a kops metadata validation error")
			}

			if masterMachineType == "" {
				masterMachineType = ig.Spec.MachineType
			} else if masterMachineType != ig.Spec.MachineType {
				warning := fmt.Sprintf("Expected all master hosts to be running same machine type, but instance group %s has type %s", ig.Metadata.Name, ig.Spec.MachineType)
				metadata.AddWarning(warning)
				c.logger.WithField("kops-metadata-error", warning).Warn("Encountered a kops metadata validation error")
			}

			masterIGCount++
			metadata.MasterInstanceGroups[ig.Metadata.Name] = model.KopsInstanceGroupMetadata{
				NodeInstanceType: ig.Spec.MachineType,
				NodeMinCount:     ig.Spec.MinSize,
				NodeMaxCount:     ig.Spec.MaxSize,
			}
		case "Node":
			if AMI == "" {
				AMI = ig.Spec.Image
			} else if AMI != ig.Spec.Image {
				warning := fmt.Sprintf("Expected all hosts to be running same AMI, but instance group %s has AMI %s", ig.Metadata.Name, ig.Spec.Image)
				metadata.AddWarning(warning)
				c.logger.WithField("kops-metadata-error", warning).Warn("Encountered a kops metadata validation error")
			}

			if strings.HasPrefix(ig.Metadata.Name, "nodes") {
				nodeIGCount++
				nodeMinCount += ig.Spec.MinSize
				nodeMaxCount += ig.Spec.MaxSize
				metadata.NodeInstanceGroups[ig.Metadata.Name] = model.KopsInstanceGroupMetadata{
					NodeInstanceType: ig.Spec.MachineType,
					NodeMinCount:     ig.Spec.MinSize,
					NodeMaxCount:     ig.Spec.MaxSize,
				}
			} else {
				metadata.CustomInstanceGroups[ig.Metadata.Name] = model.KopsInstanceGroupMetadata{
					NodeInstanceType: ig.Spec.MachineType,
					NodeMinCount:     ig.Spec.MinSize,
					NodeMaxCount:     ig.Spec.MaxSize,
				}
			}
		default:
			warning := fmt.Sprintf("Instance group %s has unknown role %s", ig.Metadata.Name, ig.Spec.Role)
			metadata.AddWarning(warning)
			c.logger.WithField("kops-metadata-error", warning).Warn("Encountered a kops metadata validation error")
		}
	}

	if masterIGCount == 0 {
		warning := "Failed to find any master instance groups"
		metadata.AddWarning(warning)
		c.logger.WithField("kops-metadata-error", warning).Warn("Encountered a kops metadata validation error")
	}
	if nodeIGCount == 0 {
		warning := "Failed to find any node instance groups"
		metadata.AddWarning(warning)
		c.logger.WithField("kops-metadata-error", warning).Warn("Encountered a kops metadata validation error")
	}

	metadata.AMI = AMI
	metadata.MasterInstanceType = masterMachineType
	metadata.MasterCount = masterIGCount
	metadata.NodeMinCount = nodeMinCount
	metadata.NodeMaxCount = nodeMaxCount

	return nil
}

// GetInstanceGroupsJSON invokes kops get instancegroup, using the context of the
// created Cmd, and returns the unmarshaled response as []InstanceGroup.
func (c *Cmd) GetInstanceGroupsJSON(clusterName string) ([]InstanceGroup, error) {
	stdout, _, err := c.run(
		"get",
		"instancegroup",
		arg("name", clusterName),
		arg("state", "s3://", c.s3StateStore),
		arg("output", "json"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to invoke kops get instancegroup")
	}

	var instanceGroupList []InstanceGroup
	err = json.Unmarshal(stdout, &instanceGroupList)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal JSON output from kops get instancegroup")
	}
	if len(instanceGroupList) == 0 {
		return nil, errors.New("expected at least one instance group, but found none")
	}
	for _, ig := range instanceGroupList {
		if len(ig.Metadata.Name) == 0 {
			return nil, errors.New("an instance group name value was empty")
		}
		if len(ig.Spec.Image) == 0 {
			return nil, errors.New("an instance group image value was empty")
		}
	}

	return instanceGroupList, nil
}

// GetInstanceGroupYAML invokes kops get instancegroup, using the context of the
// created Cmd, and returns the YAML stdout.
func (c *Cmd) GetInstanceGroupYAML(clusterName, igName string) (string, error) {
	stdout, _, err := c.run(
		"get",
		"instancegroup",
		arg("name", clusterName),
		arg("state", "s3://", c.s3StateStore),
		igName,
		arg("output", "yaml"),
	)
	trimmed := strings.TrimSuffix(string(stdout), "\n")
	if err != nil {
		return trimmed, errors.Wrap(err, "failed to invoke kops get instancegroup")
	}

	return trimmed, nil
}
