package utils

import (
	"time"

	"github.com/cenkalti/backoff/v4"
)

// Backoff holds exponential backoff settings
type Backoff struct {
	exp *backoff.ExponentialBackOff
}

// NewExponentialBackoff is used to retry a function with exponential backoff
func NewExponentialBackoff(initialInterval, maxInterval, maxElapsedTime time.Duration) *Backoff {
	expBackoff := backoff.NewExponentialBackOff()

	if initialInterval != 0 {
		expBackoff.InitialInterval = initialInterval
	}

	if maxInterval != 0 {
		expBackoff.MaxInterval = maxInterval
	}

	if maxElapsedTime != 0 {
		expBackoff.MaxElapsedTime = maxElapsedTime
	}

	return &Backoff{
		exp: expBackoff,
	}
}

// Retry is used to invoke a function with constant backoff
func (b *Backoff) Retry(fn backoff.Operation) error {
	b.exp.Reset()
	return backoff.Retry(fn, b.exp)
}
