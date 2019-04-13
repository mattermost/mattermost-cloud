package kops

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// CreateCluster invokes kops create cluster, using the context of the created Cmd.
func (c *Cmd) CreateCluster(name, cloud string, clusterSize ClusterSize, zones []string) error {
	if len(zones) == 0 {
		return fmt.Errorf("must supply at least one zone")
	}

	_, _, err := c.run(
		"create", "cluster",
		arg("name", name),
		arg("cloud", cloud),
		arg("state", "s3://", c.s3StateStore),
		commaArg("zones", zones),
		arg("node-count", clusterSize.NodeCount),
		arg("node-size", clusterSize.NodeSize),
		arg("master-size", clusterSize.MasterSize),
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

// GetCluster invokes kops get cluster, using the context of the created Cmd, and
// returns the stdout.
func (c *Cmd) GetCluster(name string) (string, error) {
	stdout, _, err := c.run(
		"get",
		"cluster",
		arg("name", name),
		arg("state", "s3://", c.s3StateStore),
	)
	trimmed := strings.TrimSuffix(string(stdout), "\n")
	if err != nil {
		return trimmed, errors.Wrap(err, "failed to invoke kops get cluster")
	}

	return trimmed, nil
}

// Version invokes kops version, using the context of the created Cmd, and
// returns the stdout.
func (c *Cmd) Version() (string, error) {
	stdout, _, err := c.run("version")
	trimmed := strings.TrimSuffix(string(stdout), "\n")
	if err != nil {
		return trimmed, errors.Wrap(err, "failed to invoke kops version")
	}

	return trimmed, nil
}
