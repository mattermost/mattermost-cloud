package helm

import "github.com/pkg/errors"

// RunGenericCommand runs any given helm command.
func (c *Cmd) RunGenericCommand(arg ...string) error {
	_, _, err := c.run(arg...)
	if err != nil {
		return errors.Wrap(err, "failed to invoke helm command")
	}

	return nil
}
