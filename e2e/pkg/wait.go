// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//+build e2e

package pkg

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"time"

	"github.com/pkg/errors"
)

// WaitConfig contains configuration for WaitForFunc.
type WaitConfig struct {
	Timeout        time.Duration
	Interval       time.Duration
	TolerateErrors int
	Logger         logrus.FieldLogger
}

// NewWaitConfig create new WaitConfig.
func NewWaitConfig(timeout, interval time.Duration, tolerateErrs int, log logrus.FieldLogger) WaitConfig {
	return WaitConfig{
		Timeout:        timeout,
		Interval:       interval,
		TolerateErrors: tolerateErrs,
		Logger:         log,
	}
}

// WaitForFunc waits until `isReady` returns `true`, error is returned or timeout reached.
func WaitForFunc(cfg WaitConfig, isReady func() (bool, error)) error {
	done := time.After(cfg.Timeout)
	errsCount := 0

	for {
		ready, err := isReady()
		if err != nil {
			if cfg.Logger != nil {
				cfg.Logger.WithError(err).Error("error while waiting for condition")
			}
			errsCount++
			if errsCount > cfg.TolerateErrors {
				return errors.Wrap(err, "while checking if condition is ready")
			}
		} else {
			if ready {
				return nil
			}
			if cfg.Logger != nil {
				cfg.Logger.Debug("condition not ready")
			}
			errsCount = 0
		}

		select {
		case <-done:
			return fmt.Errorf("timeout waiting for condition")
		default:
			time.Sleep(cfg.Interval)
		}
	}
}
