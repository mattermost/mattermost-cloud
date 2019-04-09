package kops

import (
	"fmt"

	"github.com/pkg/errors"
)

// CreateCluster invokes kops create cluster, using the context of the created Cmd.
func (c *Cmd) CreateCluster(name, cloud string, zones []string) error {
	if len(zones) == 0 {
		return fmt.Errorf("must supply at least one zone")
	}

	_, _, err := c.run(
		"create", "cluster",
		arg("name", name),
		arg("cloud", cloud),
		arg("state", "s3://", c.s3StateStore),
		commaArg("zones", zones),
		arg("target", "terraform"),
		arg("out", c.outputDir),
		arg("output", "json"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to invoke kops create cluster")
	}

	return nil
}

// UpdateCluster invokes kops update cluster, using the context of the created Cmd.
func (c *Cmd) UpdateCluster(name string) error {
	_, _, err := c.run(
		"update",
		"cluster",
		arg("name", name),
		arg("state", "s3://", c.s3StateStore),
		"--yes",
		arg("target", "terraform"),
		arg("out", c.outputDir),
	)
	if err != nil {
		return errors.Wrap(err, "failed to invoke kops update cluster")
	}

	return nil
}

// DeleteCluster invokes kops delete cluster, using the context of the created Cmd.
func (c *Cmd) DeleteCluster(name string) error {
	_, _, err := c.run(
		"delete",
		"cluster",
		arg("name", name),
		arg("state", "s3://", c.s3StateStore),
		"--yes",
	)
	if err != nil {
		return errors.Wrap(err, "failed to invoke kops delete cluster")
	}

	return nil
}
