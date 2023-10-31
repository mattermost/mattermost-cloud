// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package kops

import (
	"fmt"

	"github.com/pkg/errors"
)

// ExportKubecfg invokes kops export kubecfg for the named cluster in the context of the created Cmd.
func (c *Cmd) ExportKubecfg(name, ttl string) error {
	_, stderr, err := c.run(
		"export",
		"kubecfg",
		arg("name", name),
		arg("state", "s3://", c.s3StateStore),
		arg("admin", ttl),
	)

	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to invoke kops export kubecfg: %s", string(stderr)))
	}

	return nil
}
