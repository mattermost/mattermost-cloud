package terraform

import (
	"github.com/pkg/errors"
)

// Init invokes terraform init.
func (c *Cmd) Init() error {
	_, _, err := c.run("init")
	if err != nil {
		return errors.Wrap(err, "failed to invoke terraform init")
	}

	return nil
}

// Apply invokes terraform apply.
func (c *Cmd) Apply() error {
	_, _, err := c.run(
		"apply",
		arg("input", "false"),
		arg("auto-approve"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to invoke terraform apply")
	}

	return nil
}

// Destroy invokes terraform destroy.
func (c *Cmd) Destroy() error {
	_, _, err := c.run(
		"destroy",
		"-auto-approve",
	)
	if err != nil {
		return errors.Wrap(err, "failed to invoke terraform destroy")
	}

	return nil
}
