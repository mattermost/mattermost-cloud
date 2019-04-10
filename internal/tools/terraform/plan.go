package terraform

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

type terraformOutput struct {
	Sensitive bool        `json:"sensitive"`
	Type      string      `json:"type"`
	Value     interface{} `json:"value"`
}

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
		"-json",
		variable,
	)
	if err != nil {
		return string(stdout), errors.Wrap(err, "failed to invoke terraform output")
	}

	var output terraformOutput
	err = json.Unmarshal(stdout, &output)
	if err != nil {
		return string(stdout), errors.Wrap(err, "failed to parse terraform output")
	}

	return fmt.Sprintf("%s", output.Value), nil
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
