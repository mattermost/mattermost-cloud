package terraform

import (
	"os/exec"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Cmd is the terraform command to execute.
type Cmd struct {
	terraformPath string
	dir           string
	logger        log.FieldLogger
}

// New creates a new instance of Cmd through which to execute terraform.
func New(dir string, logger log.FieldLogger) (*Cmd, error) {
	terraformPath, err := exec.LookPath("terraform")
	if err != nil {
		return nil, errors.Wrap(err, "failed to find terraform installed on your PATH")
	}

	return &Cmd{
		terraformPath: terraformPath,
		dir:           dir,
		logger:        logger,
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
