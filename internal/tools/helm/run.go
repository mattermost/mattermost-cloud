// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package helm

import (
	"os"
	"os/exec"
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/exechelper"
	log "github.com/sirupsen/logrus"
)

const helmLoggingEnvironmentVariable string = "MM_CLOUD_VERBOSE_HELM_OUTPUT"

// SetVerboseHelmLogging controls an environment variable which
// signals whether or not the stdout output from Helm should be
// included as DEBUG level logs or not. True turns them on, false
// turns them off.
func SetVerboseHelmLogging(enable bool) {
	if enable {
		os.Setenv(helmLoggingEnvironmentVariable, "true")
	} else {
		os.Setenv(helmLoggingEnvironmentVariable, "")
	}
}

func outputLogger(line string, logger log.FieldLogger) {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return
	}

	if os.Getenv("MM_CLOUD_VERBOSE_HELM_OUTPUT") != "" {
		logger.Debugf("[helm] %s", line)
	}
}

func (c *Cmd) run(arg ...string) ([]byte, []byte, error) {
	cmd := exec.Command(c.helmPath, arg...)

	return exechelper.Run(cmd, c.logger, outputLogger)
}
