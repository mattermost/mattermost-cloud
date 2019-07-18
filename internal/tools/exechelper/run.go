// Package exechelper streamlines the running of external commands while both capturing and logging
// their output.
//
// It builds on os/exec, expecting an instance of Cmd to manipulate.
package exechelper

import (
	"bufio"
	"bytes"
	"io"
	"os/exec"
	"sync"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// OutputLogger allows custom logging of the run command output.
type OutputLogger func(line string, logger log.FieldLogger)

func bufferAndLog(reader io.Reader, buffer *bytes.Buffer, logger log.FieldLogger, outputLogger OutputLogger) error {
	scanner := bufio.NewScanner(io.TeeReader(reader, buffer))
	for scanner.Scan() {
		text := scanner.Text()
		if outputLogger == nil {
			logger.Info(scanner.Text())
		} else {
			outputLogger(text, logger)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

// Run invokes cmd.Run, both logging and returning STDOUT and STDERR, optionally transforming the output first.
func Run(cmd *exec.Cmd, logger log.FieldLogger, outputLogger OutputLogger) ([]byte, []byte, error) {
	// Generate a unique identifier for the command invocation by which to group logs.
	runID := model.NewID()

	logger = logger.WithFields(log.Fields{
		"run": runID,
	})

	logger.WithFields(log.Fields{
		"cmd":  cmd.Path,
		"args": cmd.Args,
	}).Info("Invoking command")

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	rStdout, wStdout := io.Pipe()
	rStderr, wStderr := io.Pipe()

	cmd.Stdout = wStdout
	cmd.Stderr = wStderr

	var wg sync.WaitGroup

	// Log and buffer stdout.
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := bufferAndLog(rStdout, stdout, logger, outputLogger); err != nil {
			logger.WithError(err).Error("failed to scan stdout")
		}
	}()

	// Log and buffer stderr.
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := bufferAndLog(rStderr, stderr, logger, outputLogger); err != nil {
			logger.WithError(err).Error("failed to scan stderr")
		}
	}()

	var err error
	go func() {
		err = cmd.Run()
		wStdout.Close()
		wStderr.Close()
	}()

	wg.Wait()

	if err != nil {
		logger.WithError(err).Error("failed invocation")

		return stdout.Bytes(), stderr.Bytes(), errors.Wrap(err, "failed invocation")
	}

	return stdout.Bytes(), stderr.Bytes(), nil
}
