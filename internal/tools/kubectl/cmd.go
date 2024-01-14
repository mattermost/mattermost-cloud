// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package kubectl

import (
	"os/exec"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Cmd is the kubectl command to execute.
type Cmd struct {
	kubectlPath string
	logger      log.FieldLogger
}

// New creates a new instance of Cmd through which to execute kubectl.
func New(logger log.FieldLogger) (*Cmd, error) {
	kubectlPath, err := exec.LookPath("kubectl")
	if err != nil {
		return nil, errors.Wrap(err, "failed to find kubectl installed on your PATH")
	}

	return &Cmd{
		kubectlPath: kubectlPath,
		logger:      logger,
	}, nil
}

// Close is a no-op.
func (c *Cmd) Close() error {
	return nil
}
