// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package helm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/kubectl"
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
	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("KUBECONFIG=%s", c.kubeconfig),
	)

	return cmd.Output()
}

// RepoAdd invokes helm repo add to add a repo.
func (c *Cmd) RepoAdd(repoName, repoURL string) error {
	_, _, err := c.run("repo", "add", repoName, repoURL)
	if err != nil {
		return errors.Wrap(err, "failed to invoke helm repo add")
	}

	return nil
}

// RepoUpdate invokes helm repo update to update all repo charts.
func (c *Cmd) RepoUpdate() error {
	_, _, err := c.run("repo", "update")
	if err != nil {
		return errors.Wrap(err, "failed to invoke helm repo update")
	}

	return nil
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

// FullyUpgradeLocalChart takes in the arguements necessary to fully upgrade a
// local helm chart. This includes using the kubctl client to apply CRDS.
func (c *Cmd) FullyUpgradeLocalChart(chartName, chartDirectory, namespace, valuesLocation string) error {
	kubectlClient, err := kubectl.New(c.kubeconfig, c.logger)
	if err != nil {
		return errors.Wrap(err, "failed to create kubectl client")
	}
	err = kubectlClient.RunGenericCommand("apply", "-f", filepath.Join(chartDirectory, "crds/"))
	if err != nil {
		return errors.Wrap(err, "failed to apply crds from unpacked helm chart")
	}

	_, _, err = c.run("upgrade", chartName, chartDirectory, "--namespace", namespace, "-f", valuesLocation)
	if err != nil {
		return errors.Wrap(err, "failed to invoke helm upgrade")
	}

	return nil
}

// HelmChartFoundAndDeployed is a helper func that attempts to determine if a
// given chart exists in the cluster and if it was successfully deployed.
func (c *Cmd) HelmChartFoundAndDeployed(releaseName, namespace string) (bool, bool) {
	out, _, err := c.runSilent("status", releaseName, "--namespace", namespace)
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
