package main

import (
	"github.com/mattermost/mattermost-cloud/internal/model"
	log "github.com/sirupsen/logrus"
)

var instanceID string
var logger log.FieldLogger

func init() {
	instanceID = model.NewID()
	logger = log.New().WithField("instance_id", instanceID)
}

type logrusWriter struct {
	logger log.FieldLogger
}

func (w *logrusWriter) Write(b []byte) (int, error) {
	n := len(b)
	if n > 0 && b[n-1] == '\n' {
		b = b[:n-1]
	}

	w.logger.Warning(string(b))
	return n, nil
}
