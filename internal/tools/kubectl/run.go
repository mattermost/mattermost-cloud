// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package kubectl

import (
	"os/exec"
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/exechelper"
	log "github.com/sirupsen/logrus"
)

func outputLogger(line string, logger log.FieldLogger) {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return
	}
	logger.Debugf("[kubectl] %s", line)
}

func (c *Cmd) run(arg ...string) ([]byte, []byte, error) {
	cmd := exec.Command(c.kubectlPath, arg...)
	return exechelper.RunWithEnv(cmd, c.logger, outputLogger)
}
