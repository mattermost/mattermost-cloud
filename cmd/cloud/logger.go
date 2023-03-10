// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

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

type stacktrace struct {
}

func enableLogStacktrace() {
	logger.AddHook(&stacktrace{})
}

func enableIndividualClusterLogFiles(path string) {
	logger.Hooks.Add(NewClusterLoggerHook(path, log.AllLevels))
}

func (s *stacktrace) Levels() []log.Level {
	return []log.Level{log.ErrorLevel, log.FatalLevel, log.PanicLevel}
}

func (s *stacktrace) Fire(entry *log.Entry) error {
	files := getLogCaller(7)
	if len(files) == 0 {
		return nil
	}
	entry.Data["stacktrace"] = files
	return nil
}

func getLogCaller(skip int) string {
	pcs := make([]uintptr, 25)
	depth := runtime.Callers(skip, pcs)
	frames := runtime.CallersFrames(pcs[:depth])

	var files []string
	acceptedPrefix := "github.com/mattermost/mattermost-cloud/"

	for f, again := frames.Next(); again; f, again = frames.Next() {
		pkg := getPackageName(f.Function)
		fileName := getFileName(f.File)
		if strings.HasPrefix(pkg, acceptedPrefix) {
			files = append(files, fmt.Sprintf("%s:%d", strings.TrimPrefix(fileName, acceptedPrefix), f.Line))
		}
	}

	return strings.Join(files, "; ")
}

func getFileName(file string) string {
	parts := strings.Split(file, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if strings.HasSuffix(parts[i], ".com") {
			return strings.Join(parts[i:], "/")
		}
	}

	return file
}

func getPackageName(f string) string {
	for {
		lastPeriod := strings.LastIndex(f, ".")
		lastSlash := strings.LastIndex(f, "/")
		if lastPeriod > lastSlash {
			f = f[:lastPeriod]
		} else {
			break
		}
	}

	return f
}
