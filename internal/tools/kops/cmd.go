package kops

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	defaultKopsPath = "/usr/local/bin/kops"
	outputDirName   = "output"
	kubeConfigName  = "kubeconfig"
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
	tempDir, err := ioutil.TempDir("", "kops-")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temporary kops directory")
	}

	return &Cmd{
		kopsPath:     defaultKopsPath,
		s3StateStore: s3StateStore,
		tempDir:      tempDir,
		logger:       logger,
	}, nil
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
