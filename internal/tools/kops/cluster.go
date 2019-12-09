package kops

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// CreateCluster invokes kops create cluster, using the context of the created Cmd.
func (c *Cmd) CreateCluster(name, version, ami, cloud string, clusterSize ClusterSize, zones, privateSubnetIds, publicSubnetIds, masterSecurityGroups, workerSecurityGroups []string) error {
	if len(zones) == 0 {
		return fmt.Errorf("must supply at least one zone")
	}

	args := []string{
		"create", "cluster",
		arg("name", name),
		arg("cloud", cloud),
		arg("state", "s3://", c.s3StateStore),
		commaArg("zones", zones),
		arg("node-count", clusterSize.NodeCount),
		arg("node-size", clusterSize.NodeSize),
		arg("master-count", clusterSize.MasterCount),
		arg("master-size", clusterSize.MasterSize),
		arg("target", "terraform"),
		arg("out", c.GetOutputDirectory()),
		arg("output", "json"),
	}

	if version != "latest" && version != "" {
		args = append(args,
			arg("kubernetes-version", version),
		)
	}
	if ami != "" {
		args = append(args, arg("image", ami))
	}
	if cloud == "aws" {
		args = append(args, arg("networking", "amazon-vpc-routed-eni"))
	}

	if len(privateSubnetIds) != 0 {
		args = append(args,
			commaArg("subnets", privateSubnetIds),
			arg("topology", "private"),
			arg("api-loadbalancer-type", "internal"),
		)
	}
	if len(publicSubnetIds) != 0 {
		args = append(args, commaArg("utility-subnets", publicSubnetIds))
	}
	if len(masterSecurityGroups) != 0 {
		args = append(args, commaArg("master-security-groups", masterSecurityGroups))
	}
	if len(workerSecurityGroups) != 0 {
		args = append(args, commaArg("node-security-groups", workerSecurityGroups))
	}

	_, _, err := c.run(args...)
	if err != nil {
		return errors.Wrap(err, "failed to invoke kops create cluster")
	}

	return nil
}

// SetCluster invokes kops set cluster, using the context of the created Cmd.
// Example setValue: spec.kubernetesVersion=1.10.0
func (c *Cmd) SetCluster(name, setValue string) error {
	_, _, err := c.run(
		"set",
		"cluster",
		arg("name", name),
		arg("state", "s3://", c.s3StateStore),
		setValue,
	)
	if err != nil {
		return errors.Wrap(err, "failed to invoke kops set cluster")
	}

	return nil
}

// RollingUpdateCluster invokes kops rolling-update cluster, using the context of the created Cmd.
func (c *Cmd) RollingUpdateCluster(name string) error {
	_, _, err := c.run(
		"rolling-update",
		"cluster",
		arg("name", name),
		arg("state", "s3://", c.s3StateStore),
		"--yes",
	)
	if err != nil {
		return errors.Wrap(err, "failed to invoke kops rolling-update cluster")
	}

	return nil
}

// UpdateCluster invokes kops update cluster, using the context of the created Cmd.
func (c *Cmd) UpdateCluster(name, dir string) error {
	_, _, err := c.run(
		"update",
		"cluster",
		arg("name", name),
		arg("state", "s3://", c.s3StateStore),
		"--yes",
		arg("target", "terraform"),
		arg("out", dir),
	)
	if err != nil {
		return errors.Wrap(err, "failed to invoke kops update cluster")
	}

	return nil
}

// UpgradeCluster invokes kops upgrade cluster, using the context of the created Cmd.
func (c *Cmd) UpgradeCluster(name string) error {
	_, _, err := c.run(
		"upgrade",
		"cluster",
		arg("name", name),
		arg("state", "s3://", c.s3StateStore),
		"--yes",
	)
	if err != nil {
		return errors.Wrap(err, "failed to invoke kops upgrade cluster")
	}

	return nil
}

// ValidateCluster invokes kops validate cluster, using the context of the created Cmd.
func (c *Cmd) ValidateCluster(name string, silent bool) error {
	args := []string{
		"validate",
		"cluster",
		arg("name", name),
		arg("state", "s3://", c.s3StateStore),
	}
	var err error
	if silent {
		_, _, err = c.runSilent(args...)
	} else {
		_, _, err = c.run(args...)
	}

	if err != nil {
		return errors.Wrap(err, "failed to invoke kops validate cluster")
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
