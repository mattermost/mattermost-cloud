package kops

import (
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const defaultKopsPath = "/usr/local/bin/kops"

// Cmd is the kops command to execute.
type Cmd struct {
	kopsPath     string
	s3StateStore string
	outputDir    string
	logger       log.FieldLogger
}

// New creates a new instance of Cmd through which to execute kops.
func New(s3StateStore string, logger log.FieldLogger) (*Cmd, error) {
	outputDir, err := ioutil.TempDir("", "kops")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temporary directory for output")
	}

	return &Cmd{
		kopsPath:     defaultKopsPath,
		s3StateStore: s3StateStore,
		outputDir:    outputDir,
		logger:       logger,
	}, nil
}

// GetOutputDirectory returns the temporary output directory used by kops.
func (c *Cmd) GetOutputDirectory() string {
	return c.outputDir
}

// Close cleans up the temporary output directory used by kops.
func (c *Cmd) Close() error {
	return os.RemoveAll(c.outputDir)
}
