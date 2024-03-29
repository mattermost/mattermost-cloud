// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package terraform

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	awsClient "github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/pkg/errors"
)

type terraformOutput struct {
	Sensitive bool            `json:"sensitive"`
	Type      json.RawMessage `json:"type"`
	Value     interface{}     `json:"value"`
}

// Init invokes terraform init.
func (c *Cmd) Init(remoteKey string) error {
	err := os.WriteFile(path.Join(c.dir, backendFilename), []byte(backendFile), 0644)
	if err != nil {
		return errors.Wrap(err, "unable to write terraform backend state file")
	}

	awsRegion := awsClient.GetAWSRegion()

	_, _, err = c.run(
		"init",
		arg("backend-config", fmt.Sprintf("bucket=%s", c.remoteStateBucket)),
		arg("backend-config", fmt.Sprintf("key=%s/%s", remoteStateDirectory, remoteKey)),
		arg("backend-config", fmt.Sprintf("region=%s", awsRegion)),
	)
	if err != nil {
		return errors.Wrap(err, "failed to invoke terraform init")
	}

	return nil
}

// Plan invokes terraform Plan.
func (c *Cmd) Plan() error {
	_, _, err := c.run(
		"plan",
		arg("input", "false"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to invoke terraform plan")
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

// ApplyTarget invokes terraform apply with the given target.
func (c *Cmd) ApplyTarget(target string) error {
	_, _, err := c.run(
		"apply",
		arg("input", "false"),
		arg("target", target),
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

// Output invokes terraform output and returns the named value, true if it exists, and an empty
// string and false if it does not.
func (c *Cmd) Output(variable string) (string, bool, error) {
	stdout, _, err := c.run(
		"output",
		"-json",
	)
	if err != nil {
		return string(stdout), false, errors.Wrap(err, "failed to invoke terraform output")
	}

	var outputs map[string]terraformOutput
	err = json.Unmarshal(stdout, &outputs)
	if err != nil {
		return string(stdout), false, errors.Wrap(err, "failed to parse terraform output")
	}

	value, ok := outputs[variable]

	return fmt.Sprintf("%s", value.Value), ok, nil
}

// Version invokes terraform version and returns the value.
func (c *Cmd) Version(removeUpgradeWarning bool) (string, error) {
	stdout, _, err := c.run("version")
	trimmed := strings.TrimSuffix(string(stdout), "\n")
	if err != nil {
		return trimmed, errors.Wrap(err, "failed to invoke terraform version")
	}

	// The terraform version command will print an upgrade warning if running
	// an older version. Optionally attempt to remove the warning before returning.
	if removeUpgradeWarning {
		trimmed = strings.Split(trimmed, "\n")[0]
	}

	return trimmed, nil
}
