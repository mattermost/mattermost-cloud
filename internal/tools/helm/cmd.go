// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package helm

import (
	"os/exec"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Cmd is the helm command to execute.
type Cmd struct {
	helmPath   string
	kubeconfig string
	logger     log.FieldLogger
}

// New creates a new instance of Cmd through which to execute helm.
func New(kubeconfigPath string, logger log.FieldLogger) (*Cmd, error) {
	helmPath, err := exec.LookPath("helm")
	if err != nil {
		return nil, errors.Wrap(err, "failed to find helm installed on your PATH")
	}

	return &Cmd{
		helmPath:   helmPath,
		kubeconfig: kubeconfigPath,
		logger:     logger.WithField("kubeconfig", kubeconfigPath),
	}, nil
}

// Close is a no-op.
func (c *Cmd) Close() error {
	return nil
}
