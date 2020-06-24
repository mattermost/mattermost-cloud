// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package terraform

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/exechelper"
	log "github.com/sirupsen/logrus"
)

func arg(key string, values ...string) string {
	if !strings.HasPrefix(key, "-") {
		key = fmt.Sprintf("-%s", key)
	}

	if len(values) == 0 {
		return key
	}

	value := strings.Join(values, "")

	return fmt.Sprintf("%s=%s", key, value)
}

func outputLogger(line string, logger log.FieldLogger) {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return
	}

	logger.Infof("[terraform] %s", line)
}

func (c *Cmd) run(arg ...string) ([]byte, []byte, error) {
	cmd := exec.Command(c.terraformPath, append(arg, "-no-color")...)
	cmd.Dir = c.dir
	cmd.Env = append(os.Environ(), "TF_IN_AUTOMATION=1")

	return exechelper.Run(cmd, c.logger, outputLogger)
}
