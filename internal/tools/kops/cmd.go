// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package kops

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	outputDirName  = "output"
	kubeConfigName = "kubeconfig"
)

// Cmd is the kops command to execute.
type Cmd struct {
	kopsPath     string
	s3StateStore string
	tempDir      string
	logger       log.FieldLogger
}

// New creates a new instance of Cmd through which to execute kops.
func New(s3StateStore string, logger log.FieldLogger) (*Cmd, error) {
	kopsPath, err := exec.LookPath("kops")
	if err != nil {
		return nil, errors.Wrap(err, "failed to find kops installed on your PATH")
	}

	tempDir, err := ioutil.TempDir("", "kops-")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temporary kops directory")
	}

	return &Cmd{
		kopsPath:     kopsPath,
		s3StateStore: s3StateStore,
		tempDir:      tempDir,
		logger:       logger,
	}, nil
}

// SetLogger sets a new logger for kops commands.
func (c *Cmd) SetLogger(logger log.FieldLogger) {
	c.logger = logger
}

// GetTempDir returns the root temporary directory used by kops.
func (c *Cmd) GetTempDir() string {
	return c.tempDir
}

// GetOutputDirectory returns the temporary output directory used by kops.
func (c *Cmd) GetOutputDirectory() string {
	return path.Join(c.tempDir, outputDirName)
}

// GetKubeConfigPath returns the temporary kubeconfig directory used by kops.
func (c *Cmd) GetKubeConfigPath() string {
	return path.Join(c.tempDir, kubeConfigName)
}

// Close cleans up the temporary output directory used by kops.
func (c *Cmd) Close() error {
	return os.RemoveAll(c.tempDir)
}
