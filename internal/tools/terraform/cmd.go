package terraform

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"os/exec"
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

// Close is a no-op.
func (c *Cmd) Close() error {
	return nil
}
