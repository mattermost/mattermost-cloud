package terraform

import (
	log "github.com/sirupsen/logrus"
)

const defaultTerraformPath = "/usr/local/bin/terraform"

// Cmd is the terraform command to execute.
type Cmd struct {
	terraformPath string
	dir           string
	logger        log.FieldLogger
}

// New creates a new instance of Cmd through which to execute terraform.
func New(dir string, logger log.FieldLogger) *Cmd {
	return &Cmd{
		terraformPath: defaultTerraformPath,
		dir:           dir,
		logger:        logger,
	}
}

// Close is a no-op.
func (c *Cmd) Close() error {
	return nil
}
