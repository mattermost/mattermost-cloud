// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package kubectl

import (
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

// RunGenericCommand runs any given kubectl command.
func (c *Cmd) RunGenericCommand(arg ...string) error {
	_, _, err := c.run(arg...)
	if err != nil {
		return errors.Wrap(err, "failed to invoke kubectl command")
	}

	return nil
}

// RunCommandRaw runs any given kubectl command returning raw output.
func (c *Cmd) RunCommandRaw(arg ...string) ([]byte, error) {
	cmd := exec.Command(c.kubectlPath, arg...)
	return cmd.Output()
}

// Version invokes kubectl version and returns the value.
func (c *Cmd) Version() (string, error) {
	stdout, _, err := c.run("version", "--client", "true")
	minimized := strings.ReplaceAll(strings.TrimSuffix(string(stdout), "\n"), "\n", " | ")
	if err != nil {
		return minimized, errors.Wrap(err, "failed to invoke kubectl version")
	}

	return minimized, nil
}
