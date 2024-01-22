// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package helm

import (
	"fmt"
	"io"
	"os"
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
	logger.Debugf("[helm] %s", line)
}

func (c *Cmd) run(arg ...string) ([]byte, []byte, error) {
	cmd := exec.Command(c.helmPath, arg...)
	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("KUBECONFIG=%s", c.kubeconfig),
	)

	return exechelper.RunWithEnv(cmd, c.logger, outputLogger)
}

func (c *Cmd) runSilent(arg ...string) ([]byte, []byte, error) {
	cmd := exec.Command(c.helmPath, arg...)
	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("KUBECONFIG=%s", c.kubeconfig),
	)

	return exechelper.Run(cmd, silentLogger(), func(string, log.FieldLogger) {})
}

func silentLogger() log.FieldLogger {
	silentLogger := log.New()
	silentLogger.Out = io.Discard

	return silentLogger
}
