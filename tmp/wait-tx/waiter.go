//go:build ignore
// +build ignore

package waittx

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Waiter struct {
	sub Source // optional
	pol Source // required (fallback)
	// optional: a short window to attempt subscription setup before we give up
	subSetupDeadline time.Duration
}

type WaiterOption func(*Waiter)

func WithSubscriber(sub Source, setupDeadline time.Duration) WaiterOption {
	return func(w *Waiter) { w.sub = sub; w.subSetupDeadline = setupDeadline }
}
func WithPoller(pol Source) WaiterOption {
	return func(w *Waiter) { w.pol = pol }
}

func NewWaiter(opts ...WaiterOption) (*Waiter, error) {
	w := &Waiter{}
	for _, o := range opts {
		o(w)
	}
	if w.pol == nil {
		return nil, errors.New("poller is required")
	}
	if w.subSetupDeadline <= 0 {
		w.subSetupDeadline = 2 * time.Second
	}
	return w, nil
}

func (w *Waiter) WaitTx(ctx context.Context, txHash string, overallTimeout time.Duration) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, overallTimeout)
	defer cancel()

	// Try subscription first (with a short setup window).
	if w.sub != nil {
		subCtx, scancel := context.WithTimeout(ctx, w.subSetupDeadline)
		defer scancel()

		ch := make(chan Result, 1)
		errCh := make(chan error, 1)

		go func() {
			res, err := w.sub.Wait(ctx, txHash) // important: use parent ctx, so it can run beyond setup time if established
			if err != nil {
				errCh <- err
				return
			}
			ch <- res
		}()

		// If we see a result quickly, great; if we see a setup error, fall back; if time passes, fall back.
		select {
		case <-subCtx.Done():
			// setup window elapsed → fall back
		case err := <-errCh:
			// subscribe failed → fall back
			_ = err // could log
		case res := <-ch:
			return res, nil
		}
	}

	// Fallback to polling until overall ctx ends.
	res, err := w.pol.Wait(ctx, txHash)
	if err != nil {
		return Result{}, fmt.Errorf("fallback polling failed: %w", err)
	}
	return res, nil
}
