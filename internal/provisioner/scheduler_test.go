package provisioner_test

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/provisioner"
	"github.com/stretchr/testify/assert"
)

func TestScheduler(t *testing.T) {
	t.Run("manual only", func(t *testing.T) {
		t.Parallel()

		doer := &testDoer{
			calls: make(chan bool, 1),
		}
		scheduler := provisioner.NewScheduler(doer, 0*time.Second)
		defer scheduler.Close()

		scheduler.Do()

		select {
		case <-doer.calls:
		case <-time.After(5 * time.Second):
			assert.Fail(t, "doer not invoked within 5 seconds")
		}
	})

	t.Run("periodic only", func(t *testing.T) {
		t.Parallel()

		doer := &testDoer{
			calls: make(chan bool, 1),
		}
		scheduler := provisioner.NewScheduler(doer, 100*time.Millisecond)
		defer scheduler.Close()

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
		scheduler := provisioner.NewScheduler(doer, 30*time.Second)
		defer scheduler.Close()

		scheduler.Do()

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
		scheduler := provisioner.NewScheduler(doer, 30*time.Second)
		scheduler.Close()

		scheduler.Do()

		select {
		case <-doer.calls:
			assert.Fail(t, "doer should not have been invoked")
		case <-time.After(500 * time.Millisecond):
		}
	})
}
