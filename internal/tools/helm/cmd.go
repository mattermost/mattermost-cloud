package helm

import (
	"os/exec"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Cmd is the helm command to execute.
type Cmd struct {
	helmPath string
	logger   log.FieldLogger
}

// New creates a new instance of Cmd through which to execute helm.
func New(logger log.FieldLogger) (*Cmd, error) {
	helmPath, err := exec.LookPath("helm")
	if err != nil {
		return nil, errors.Wrap(err, "failed to find helm installed on your PATH")
	}

	return &Cmd{
		helmPath: helmPath,
		logger:   logger,
	}, nil
}

// Close is a no-op.
func (c *Cmd) Close() error {
	return nil
}
