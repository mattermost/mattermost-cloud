package kops

import (
	"encoding/json"
	"strings"

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
	Image string `json:"image"`
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
