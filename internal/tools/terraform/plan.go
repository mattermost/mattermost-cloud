package terraform

import (
	"strings"

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

// Output invokes terraform output and returns the value.
func (c *Cmd) Output(variable string) (string, error) {
	stdout, _, err := c.run(
		"output",
		variable,
	)
	trimmed := strings.TrimSuffix(string(stdout), "\n")
	if err != nil {
		return trimmed, errors.Wrap(err, "failed to invoke terraform output")
	}

	return trimmed, nil
}

// Version invokes terraform version and returns the value.
func (c *Cmd) Version() (string, error) {
	stdout, _, err := c.run("version")
	trimmed := strings.TrimSuffix(string(stdout), "\n")
	if err != nil {
		return trimmed, errors.Wrap(err, "failed to invoke terraform version")
	}

	return trimmed, nil
}
