package kops

import "github.com/pkg/errors"

// ExportKubecfg invokes kops export kubecfg for the named cluster in the context of the created Cmd.
func (c *Cmd) ExportKubecfg(name string) error {
	_, _, err := c.run(
		"export",
		"kubecfg",
		arg("name", name),
		arg("state", "s3://", c.s3StateStore),
	)
	if err != nil {
		return errors.Wrap(err, "failed to invoke kops export kubecfg")
	}

	return nil
}
