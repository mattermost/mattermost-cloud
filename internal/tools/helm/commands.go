// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package helm

import (
	"github.com/pkg/errors"
	"os/exec"
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
