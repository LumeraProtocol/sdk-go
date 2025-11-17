//go:build ignore
// +build ignore

package waittx

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
)

type TxQuerier interface {
	GetTx(ctx context.Context, req *txtypes.GetTxRequest) (*txtypes.GetTxResponse, error)
}

// ConstantBackoff is a simple backoff implementation.
type ConstantBackoff struct{ Every time.Duration }

func (b ConstantBackoff) Next(int) time.Duration { return b.Every }

// ExponentialBackoff grows delays geometrically and optionally applies jitter.
type ExponentialBackoff struct {
	Initial    time.Duration
	Multiplier float64
	Max        time.Duration
	Jitter     float64
	Rand       func() float64
}

func (b ExponentialBackoff) Next(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	initial := b.Initial
	if initial <= 0 {
		initial = 500 * time.Millisecond
	}
	multiplier := b.Multiplier
	if multiplier <= 1 {
		multiplier = 1
	}
	maxDelay := b.Max
	if maxDelay <= 0 {
		maxDelay = 0
	}
	base := float64(initial)
	if multiplier > 1 {
		pow := math.Pow(multiplier, float64(attempt-1))
		base *= pow
	}
	if maxDelay > 0 {
		limit := float64(maxDelay)
		if base > limit {
			base = limit
		}
	}
	if base > float64(math.MaxInt64) {
		base = float64(math.MaxInt64)
	}
	delay := time.Duration(base)
	jitter := b.Jitter
	if jitter < 0 {
		jitter = 0
	}
	if jitter > 1 {
		jitter = 1
	}
	if jitter > 0 {
		randFn := b.Rand
		if randFn == nil {
			randFn = rand.Float64
		}
		factor := 1 + (randFn()*2-1)*jitter
		if factor < 0 {
			factor = 0
		}
		floatDelay := float64(delay) * factor
		if floatDelay > float64(math.MaxInt64) {
			floatDelay = float64(math.MaxInt64)
		}
		delay = time.Duration(floatDelay)
	}
	if delay <= 0 {
		delay = time.Millisecond
	}
	return delay
}

// Poller waits for a tx by periodically calling GetTx.
type Poller struct {
	querier  TxQuerier
	backoff  Backoff
	maxTries int // 0 or <0 means unlimited until ctx timeout
}

func NewPoller(q TxQuerier, b Backoff, maxTries int) *Poller {
	return &Poller{querier: q, backoff: b, maxTries: maxTries}
}

func (p *Poller) Wait(ctx context.Context, txHash string) (Result, error) {
	attempt := 0
	for {
		select {
		case <-ctx.Done():
			return Result{}, ctx.Err()
		default:
		}

		resp, err := p.querier.GetTx(ctx, &txtypes.GetTxRequest{Hash: txHash})
		if err == nil && resp != nil && resp.TxResponse != nil && resp.TxResponse.TxHash != "" {
			// found
			ev := map[string][]string{}
			for _, e := range resp.TxResponse.Events {
				for _, a := range e.Attributes {
					key := e.Type + "." + string(a.Key)
					ev[key] = append(ev[key], string(a.Value))
				}
			}
			return Result{Code: resp.TxResponse.Code, Events: ev}, nil
		}

		attempt++
		if p.maxTries > 0 && attempt >= p.maxTries {
			return Result{}, fmt.Errorf("polling exhausted after %d attempts: %w", attempt, err)
		}
		select {
		case <-ctx.Done():
			return Result{}, ctx.Err()
		case <-sleepCtx(ctx, p.backoff.Next(attempt)):
		}
	}
}

// sleepCtx returns a channel that closes after d or when ctx is done.
func sleepCtx(ctx context.Context, d time.Duration) <-chan struct{} {
	ch := make(chan struct{})
	if d <= 0 {
		close(ch)
		return ch
	}
	go func() {
		t := time.NewTimer(d)
		defer t.Stop()
		select {
		case <-ctx.Done():
		case <-t.C:
		}
		close(ch)
	}()
	return ch
}
