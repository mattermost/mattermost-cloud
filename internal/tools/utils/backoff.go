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
	expBackoff.InitialInterval = initialInterval
	expBackoff.MaxInterval = maxInterval
	expBackoff.MaxElapsedTime = maxElapsedTime

	if expBackoff.InitialInterval == 0 {
		expBackoff.InitialInterval = backoff.DefaultInitialInterval
	}

	if expBackoff.MaxInterval == 0 {
		expBackoff.MaxInterval = backoff.DefaultMaxInterval
	}

	return &Backoff{
		exp: expBackoff,
	}
}

// Retry is used to invoke a function with constant backoff
func (b *Backoff) Retry(fn func() error) error {
	b.exp.Reset()
	return backoff.Retry(fn, b.exp)
}
