package utils

import (
	"github.com/cenkalti/backoff/v4"
)

type Backoff struct {
	*backoff.ExponentialBackOff
}

func NewExponentialBackoff(expBackoff *backoff.ExponentialBackOff) *Backoff {
	return &Backoff{
		expBackoff,
	}
}

func (b *Backoff) Retry(fn func() error) error {
	return backoff.Retry(fn, b)
}
