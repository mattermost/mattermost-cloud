// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"os"

	log "github.com/sirupsen/logrus"
)

var logger *log.Logger

func init() {
	logger = log.New()
	logger.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	// Output to stdout instead of the default stderr.
	log.SetOutput(os.Stdout)
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
