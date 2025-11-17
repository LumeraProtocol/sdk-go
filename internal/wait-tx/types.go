package waittx

import (
	"context"
	"time"
)

// Result represents the outcome produced by waiting on a tx.
type Result struct {
	Code   uint32
	Events map[string][]string
}

// Source abstracts a tx wait mechanism (poller, subscriber, etc).
type Source interface {
	Wait(ctx context.Context, txHash string) (Result, error)
}

// Backoff controls polling cadence.
type Backoff interface {
	Next(attempt int) time.Duration
}
