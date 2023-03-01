// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

// Based on https://github.com/rifflock/lfshook

package main

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/sirupsen/logrus"
)

// Remove colors from the default formatter since we are writting to a file
var fileLogrusFormatter = &logrus.TextFormatter{DisableColors: true}

// ClusterLoggerHook is a hook to handle writing to local log files.
type ClusterLoggerHook struct {
	levels    []logrus.Level
	lock      *sync.Mutex
	formatter logrus.Formatter

	path string
}

// NewClusterLoggerHook returns the hook accepting a parameter to set the path where files should
// be stored.
func NewClusterLoggerHook(path string, levels []logrus.Level) *ClusterLoggerHook {
	hook := &ClusterLoggerHook{
		lock:   new(sync.Mutex),
		levels: levels,
		path:   path,
	}

	// TODO: Ensure paths are created here rather on every write

	hook.formatter = fileLogrusFormatter

	return hook
}

// Fire writes the log file if the specified field is present on the entry
func (hook *ClusterLoggerHook) Fire(entry *logrus.Entry) error {
	hook.lock.Lock()
	defer hook.lock.Unlock()

	if err := hook.fileWrite(entry, "cluster"); err != nil {
		return err
	}

	if err := hook.fileWrite(entry, "installation"); err != nil {
		return err
	}

	return nil
}

// Write a log line directly to a file.
func (hook *ClusterLoggerHook) fileWrite(entry *logrus.Entry, kind string) error {
	dataClusterID, exist := entry.Data[kind]
	if !exist {
		return nil
	}

	clusterID := dataClusterID.(string)

	//  TODO: move this to NewClusterLoggerHook
	dir := filepath.Dir(filepath.Join(hook.path, kind))
	os.MkdirAll(dir, os.ModePerm)

	path := filepath.Join(dir, clusterID+".log")

	fd, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		log.Println("failed to open logfile:", path, err)
		return err
	}
	defer fd.Close()

	// use our formatter instead of entry.String()
	msg, err := hook.formatter.Format(entry)

	if err != nil {
		log.Println("failed to generate string for entry:", err)
		return err
	}
	fd.Write(msg)
	return nil
}

// Levels returns configured log levels.
func (hook *ClusterLoggerHook) Levels() []logrus.Level {
	return hook.levels
}
