package helm

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

	logger.Debugf("[helm] %s", line)
}

func (c *Cmd) run(arg ...string) ([]byte, []byte, error) {
	cmd := exec.Command(c.helmPath, arg...)

	return exechelper.Run(cmd, c.logger, outputLogger)
}
