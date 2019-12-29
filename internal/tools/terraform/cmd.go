package terraform

import (
	"os/exec"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	// backendFilename is the backend terraform base configuration for S3 remote
	// state.
	backendFilename = "backend.tf"
	// remoteStateDirectory is the directory inside of the S3 bucket that will
	// contain all of the terraform remote state.
	remoteStateDirectory = "terraform"
)

// Cmd is the terraform command to execute.
type Cmd struct {
	terraformPath     string
	dir               string
	remoteStateBucket string
	logger            log.FieldLogger
}

// New creates a new instance of Cmd through which to execute terraform.
func New(dir, remoteStateBucket string, logger log.FieldLogger) (*Cmd, error) {
	if remoteStateBucket == "" {
		return nil, errors.New("remote state bucket cannot be an empty value")
	}
	terraformPath, err := exec.LookPath("terraform")
	if err != nil {
		return nil, errors.Wrap(err, "failed to find terraform installed on your PATH")
	}

	return &Cmd{
		terraformPath:     terraformPath,
		dir:               dir,
		remoteStateBucket: remoteStateBucket,
		logger:            logger,
	}, nil
}

// GetWorkingDirectory returns the working directory used by terraform.
func (c *Cmd) GetWorkingDirectory() string {
	return c.dir
}

// Close is a no-op.
func (c *Cmd) Close() error {
	return nil
}
