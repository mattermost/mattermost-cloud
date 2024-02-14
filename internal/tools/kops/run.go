// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package kops

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/exechelper"
	log "github.com/sirupsen/logrus"
)

func arg(key string, values ...string) string {
	value := strings.Join(values, "")

	if strings.HasPrefix(key, "--") {
		return fmt.Sprintf("%s=%s", key, value)
	}

	return fmt.Sprintf("--%s=%s", key, value)
}

func commaArg(key string, values []string) string {
	return arg(key, strings.Join(values, ","))
}

var glogRe *regexp.Regexp

func init() {
	// See https://chromium.googlesource.com/external/github.com/golang/glog/+/65d674618f712aa808a7d0104131b9206fc3d5ad/glog.go#518.
	glogRe = regexp.MustCompile(`([A-Z])([0-9]{2})([0-9]{2}) ([0-9]{2}):([0-9]{2}):([0-9]{2}).([0-9]{6}) +([0-9]+) ([^ ]+):([0-9]+)\] (.+)`)
}

func outputLogger(line string, logger log.FieldLogger) {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return
	}

	matches := glogRe.FindStringSubmatch(line)
	if matches == nil {
		// Assume non-glog output is a warning.
		logger.Infof("[kops] %s", line)
		return
	}

	level := matches[1]
	msg := fmt.Sprintf("[kops] %s", matches[len(matches)-1])
	switch level {
	case "I":
		logger.Info(msg)
	case "W":
		logger.Warn(msg)
	case "E", "F":
		logger.Error(msg)
	default:
		logger.Warn(msg)
	}
}

func (c *Cmd) run(arg ...string) (stdout []byte, stderr []byte, err error) {
	cmd := exec.Command(c.kopsPath, arg...)
	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("KUBECONFIG=%s", c.GetKubeConfigPath()),
	)

	return exechelper.Run(cmd, c.logger, outputLogger)
}

func (c *Cmd) runSilent(arg ...string) ([]byte, []byte, error) {
	cmd := exec.Command(c.kopsPath, arg...)
	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("KUBECONFIG=%s", c.GetKubeConfigPath()),
	)

	return exechelper.Run(cmd, silentLogger(), func(string, log.FieldLogger) {})
}

func silentLogger() log.FieldLogger {
	silentLogger := log.New()
	silentLogger.Out = io.Discard

	return silentLogger
}
