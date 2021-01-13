// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package kops

import (
	"fmt"
	"path"
	"strings"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

// CreateCluster invokes kops create cluster, using the context of the created Cmd.
func (c *Cmd) CreateCluster(name, cloud string, kopsRequest *model.KopsMetadataRequestedState, zones, privateSubnetIds, publicSubnetIds, masterSecurityGroups, workerSecurityGroups, allowSSHCIDRS []string) error {
	if len(zones) == 0 {
		return fmt.Errorf("must supply at least one zone")
	}

	args := []string{
		"create", "cluster",
		arg("name", name),
		arg("cloud", cloud),
		arg("state", "s3://", c.s3StateStore),
		commaArg("zones", zones),
		arg("node-count", fmt.Sprintf("%d", kopsRequest.NodeMinCount)),
		arg("node-size", kopsRequest.NodeInstanceType),
		arg("master-count", fmt.Sprintf("%d", kopsRequest.MasterCount)),
		arg("master-size", kopsRequest.MasterInstanceType),
		arg("target", "terraform"),
		arg("out", c.GetOutputDirectory()),
		arg("ssh-access", strings.Join(allowSSHCIDRS, ",")),
		arg("output", "json"),
	}

	if kopsRequest.Version != "latest" && kopsRequest.Version != "" {
		args = append(args,
			arg("kubernetes-version", kopsRequest.Version),
		)
	}
	if kopsRequest.AMI != "" {
		args = append(args, arg("image", kopsRequest.AMI))
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

// Replace invokes kops replace, using the context of the created Cmd, and
// returns the stdout. The filename passed in is expected to be in the root temp
// dir of this kops command.
func (c *Cmd) Replace(name string) (string, error) {
	stdout, _, err := c.run(
		"replace",
		arg("filename", path.Join(c.GetTempDir(), name)),
		arg("state", "s3://", c.s3StateStore),
	)
	trimmed := strings.TrimSuffix(string(stdout), "\n")
	if err != nil {
		return trimmed, errors.Wrap(err, "failed to invoke kops get instancegroup")
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
