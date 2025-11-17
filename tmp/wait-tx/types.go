//go:build ignore
// +build ignore

package waittx

import (
	"context"
	"time"
)

// Result is what WaitTx returns.
type Result struct {
	Code   uint32
	Events map[string][]string
}

// Source abstracts "something that can wait for a tx".
type Source interface {
	// Wait blocks until the tx is confirmed or ctx ends.
	Wait(ctx context.Context, txHash string) (Result, error)
}

// Backoff defines polling cadence.
type Backoff interface {
	// Next returns the next wait duration (can be constant or exponential).
	Next(attempt int) time.Duration
}
