package provisioner

import (
	"time"
)

// Scheduler schedules a doer for periodic, serial execution.
type Scheduler struct {
	doer   Doer
	period time.Duration
	notify chan bool
	stop   chan bool
	done   chan bool
}

// NewScheduler creates a new scheduler.
//
// If the period is zero, the doer is only run on demand. Otherwise, the period specifies how
// long to wait after its last successful execution.
func NewScheduler(doer Doer, period time.Duration) *Scheduler {
	s := &Scheduler{
		doer:   doer,
		period: period,
		notify: make(chan bool, 1),
		stop:   make(chan bool),
		done:   make(chan bool),
	}

	go s.run()

	return s
}

// Do requests an execution of the scheduled doer.
//
// If already running, the doer will be run again when done. Multiple calls to Notify while the
// doer is running will only trigger a single additional execution.
func (s *Scheduler) Do() error {
	select {
	case s.notify <- true:
	default:
	}

	return nil
}

// run is the main thread of the scheduler and responsible for triggering the doer as required.
func (s *Scheduler) run() {
	for {
		// Enabling polling only if a non-zero interval was configured.
		var poll <-chan time.Time
		if s.period > 0 {
			poll = time.After(s.period)
		}

		select {
		case <-poll:
			_ = s.doer.Do()
		case <-s.notify:
			_ = s.doer.Do()
		case <-s.stop:
			close(s.done)
			return
		}
	}
}

// Close waits for any active doer to finish, terminates the main thread of the scheduler, and
// ensures the doer is no longer invoked.
func (s *Scheduler) Close() error {
	close(s.stop)
	<-s.done

	return nil
}
