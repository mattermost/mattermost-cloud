package kops

import (
	"encoding/json"
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

// GetInstanceGroupsJSON invokes kops get instancegroup, using the context of the
// created Cmd, and returns the unmarshaled response as []InstanceGroup.
func (c *Cmd) UpdateMetadata(metadata *model.KopsMetadata) error {
	instanceGroups, err := c.GetInstanceGroupsJSON(metadata.Name)
	if err != nil {
		return err
	}

	// This part is a bit tricky. We are attempting to correlate multiple kops
	// instance groups into a simplified set of metadata information. To do so,
	// we assume and check the following:
	// - There is one worker node instance group.
	// - There is one or more master instance groups.
	// - All of the cluster hosts are running the same AMI.
	// - All of the master nodes are running the same instance type.
	var masterIGCount, NodeIGCount, nodeMinCount, nodeMaxCount int64
	var masterMachineType, nodeInstanceType, AMI string
	for _, ig := range instanceGroups {
		switch ig.Spec.Role {
		case "Master":
			if AMI == "" {
				AMI = ig.Spec.Image
			} else if AMI != ig.Spec.Image {
				return errors.Errorf("expected all hosts to be running same AMI, but found %s and %s", AMI, ig.Spec.Image)
			}

			if masterMachineType == "" {
				masterMachineType = ig.Spec.MachineType
			} else if masterMachineType != ig.Spec.MachineType {
				return errors.Errorf("expected all master hosts to be running same machine type, but found %s and %s", masterMachineType, ig.Spec.MachineType)
			}

			masterIGCount++
		case "Node":
			if AMI == "" {
				AMI = ig.Spec.Image
			} else if AMI != ig.Spec.Image {
				return errors.Errorf("expected all hosts to be running same AMI, but found %s and %s", AMI, ig.Spec.Image)
			}

			NodeIGCount++
			nodeInstanceType = ig.Spec.MachineType
			nodeMinCount = ig.Spec.MinSize
			nodeMaxCount = ig.Spec.MaxSize
		default:
			return errors.Errorf("instance group has unknown role '%s'", ig.Spec.Role)
		}
	}

	if masterIGCount == 0 {
		return errors.New("failed to find any master instance groups")
	}
	if NodeIGCount != 1 {
		return errors.Errorf("expected exactly 1 node instance group, but found %d", NodeIGCount)
	}

	metadata.AMI = AMI
	metadata.MasterInstanceType = masterMachineType
	metadata.MasterCount = masterIGCount
	metadata.NodeInstanceType = nodeInstanceType
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
