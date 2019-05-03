package main

import (
	log "github.com/sirupsen/logrus"
)

var logger log.FieldLogger

func init() {
	logger = log.New()
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
