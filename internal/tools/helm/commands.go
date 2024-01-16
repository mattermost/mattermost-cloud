// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package helm

import (
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

// RunGenericCommand runs any given helm command.
func (c *Cmd) RunGenericCommand(arg ...string) error {
	_, _, err := c.run(arg...)
	if err != nil {
		return errors.Wrap(err, "failed to invoke helm command")
	}

	return nil
}

// RunCommandRaw runs any given helm command returning raw output.
func (c *Cmd) RunCommandRaw(arg ...string) ([]byte, error) {
	cmd := exec.Command(c.helmPath, arg...)
	return cmd.Output()
}

// Version invokes helm version and returns the value.
func (c *Cmd) Version() (string, error) {
	stdout, _, err := c.run("version")
	trimmed := strings.TrimSuffix(string(stdout), "\n")
	if err != nil {
		return trimmed, errors.Wrap(err, "failed to invoke helm version")
	}

	return trimmed, nil
}

// HelmChartFoundAndDeployed is a helper func that attempts to determine if a
// given chart exists in the cluster and if it was successfully deployed.
func (c *Cmd) HelmChartFoundAndDeployed(releaseName, kubeconfigPath string) (bool, bool) {
	out, _, err := c.runSilent("status", releaseName, "--kubeconfig", kubeconfigPath)
	if err == nil {
		if strings.Contains(string(out), "STATUS: deployed") {
			// found and deployed
			return true, true
		}
		// found, but not successfully deployed
		return true, false
	}

	// not found or deployed
	return false, false
}
