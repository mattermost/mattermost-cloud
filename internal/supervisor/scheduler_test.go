// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor_test

import (
	log "github.com/sirupsen/logrus"
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/stretchr/testify/assert"
)

func TestScheduler(t *testing.T) {
	logger := testlib.MakeLogger(t)

	t.Run("disabled", func(t *testing.T) {
		t.Parallel()

		doer := &testDoer{
			calls: make(chan bool, 1),
		}
		scheduler := supervisor.NewScheduler(doer, 0*time.Second, logger)
		defer scheduler.Close()

		scheduler.Do()

		select {
		case <-doer.calls:
			assert.Fail(t, "doer should not have been invoked")
		case <-time.After(500 * time.Millisecond):
		}
	})

	t.Run("periodic only", func(t *testing.T) {
		t.Parallel()

		doer := &testDoer{
			calls: make(chan bool, 1),
		}
		scheduler := supervisor.NewScheduler(doer, 100*time.Millisecond, logger)
		defer func() {
			if err := scheduler.Close(); err != nil {
				log.WithError(err).Error("failed to close scheduler")
			}
		}()

		for i := 0; i < 5; i++ {
			select {
			case <-doer.calls:
			case <-time.After(5 * time.Second):
				assert.Fail(t, "doer not invoked within 5 seconds")
			}
		}
	})

	t.Run("periodic and manual", func(t *testing.T) {
		t.Parallel()

		doer := &testDoer{
			calls: make(chan bool, 1),
		}
		scheduler := supervisor.NewScheduler(doer, 30*time.Second, logger)
		defer func() {
			if err := scheduler.Close(); err != nil {
				log.WithError(err).Error("failed to close scheduler")
			}
		}()

		if err := scheduler.Do(); err != nil {
			log.WithError(err).Error("supervisor task failed")
		}

		select {
		case <-doer.calls:
		case <-time.After(5 * time.Second):
			assert.Fail(t, "doer not invoked within 5 seconds")
		}
	})

	t.Run("after close", func(t *testing.T) {
		t.Parallel()

		doer := &testDoer{
			calls: make(chan bool, 1),
		}
		scheduler := supervisor.NewScheduler(doer, 30*time.Second, logger)
		defer func() {
			if err := scheduler.Close(); err != nil {
				log.WithError(err).Error("failed to close scheduler")
			}
		}()

		if err := scheduler.Do(); err != nil {
			log.WithError(err).Error("supervisor task failed")
		}

		select {
		case <-doer.calls:
			assert.Fail(t, "doer should not have been invoked")
		case <-time.After(500 * time.Millisecond):
		}
	})

	t.Run("while busy", func(t *testing.T) {
		t.Parallel()

		doer := &testDoer{
			calls: make(chan bool),
		}
		scheduler := supervisor.NewScheduler(doer, 30*time.Second, logger)
		defer func() {
			if err := scheduler.Close(); err != nil {
				log.WithError(err).Error("failed to close scheduler")
			}
		}()

		if err := scheduler.Do(); err != nil {
			log.WithError(err).Error("supervisor task failed")
		}

		time.Sleep(1 * time.Second)

		// Second call should be non-blocking
		if err := scheduler.Do(); err != nil {
			log.WithError(err).Error("supervisor task failed")
		}

		select {
		case <-doer.calls:
		case <-time.After(5 * time.Second):
			assert.Fail(t, "doer not invoked within 5 seconds")
		}

		<-doer.calls
	})
}
