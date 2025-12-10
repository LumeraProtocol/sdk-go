package waittx

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	abcipb "cosmossdk.io/api/cosmos/base/abci/v1beta1"
	txtypes "cosmossdk.io/api/cosmos/tx/v1beta1"
	clientconfig "github.com/LumeraProtocol/sdk-go/client/config"
)

type poller struct {
	querier  Querier
	backoff  Backoff
	maxTries int
}

type constantBackoff struct{ every time.Duration }

func (b constantBackoff) Next(int) time.Duration { return b.every }

type exponentialBackoff struct {
	initial    time.Duration
	multiplier float64
	max        time.Duration
	jitter     float64
	randFn     func() float64
}

func (b *exponentialBackoff) Next(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	// Cap values before converting to time.Duration to avoid overflow.
	const safetyMargin = 2048.0
	maxDurationFloat := float64(math.MaxInt64) - safetyMargin

	initial := b.initial
	if initial <= 0 {
		initial = 500 * time.Millisecond
	}
	multiplier := b.multiplier
	if multiplier <= 1 {
		multiplier = 1
	}
	maxDelay := b.max
	base := float64(initial)
	if multiplier > 1 {
		base *= math.Pow(multiplier, float64(attempt-1))
	}
	if maxDelay > 0 {
		maxCap := float64(maxDelay)
		if maxCap > maxDurationFloat {
			maxCap = maxDurationFloat
		}
		if base > maxCap {
			base = maxCap
		}
	}
	if base > maxDurationFloat {
		base = maxDurationFloat
	}
	if base < 0 {
		base = 0
	}
	delay := time.Duration(base)
	jitter := b.jitter
	if jitter > 1 {
		jitter = 1
	}
	if jitter < 0 {
		jitter = 0
	}
	if jitter > 0 {
		randFn := b.randFn
		if randFn == nil {
			randFn = rand.Float64
		}
		factor := 1 + (randFn()*2-1)*jitter
		if factor < 0 {
			factor = 0
		}
		floatDelay := float64(delay) * factor
		if floatDelay > maxDurationFloat {
			floatDelay = maxDurationFloat
		}
		if floatDelay < 0 {
			floatDelay = 0
		}
		delay = time.Duration(floatDelay)
	}
	if delay <= 0 {
		delay = time.Millisecond
	}
	return delay
}

func newPoller(q Querier, cfg clientconfig.WaitTxConfig) *poller {
	return &poller{
		querier:  q,
		backoff:  NewBackoff(cfg),
		maxTries: cfg.PollMaxRetries,
	}
}

// NewBackoff constructs a poller backoff from the WaitTx configuration.
func NewBackoff(cfg clientconfig.WaitTxConfig) Backoff {
	interval := cfg.PollInterval
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	var backoff Backoff = constantBackoff{every: interval}
	if cfg.PollBackoffMultiplier > 1 || cfg.PollBackoffJitter > 0 || (cfg.PollBackoffMaxInterval > 0 && cfg.PollBackoffMaxInterval != interval) {
		backoff = &exponentialBackoff{
			initial:    interval,
			multiplier: cfg.PollBackoffMultiplier,
			max:        cfg.PollBackoffMaxInterval,
			jitter:     cfg.PollBackoffJitter,
		}
	}
	return backoff
}

func (p *poller) Wait(ctx context.Context, txHash string) (Result, error) {
	attempt := 0
	for {
		select {
		case <-ctx.Done():
			return Result{}, ctx.Err()
		default:
		}

		resp, err := p.querier.GetTx(ctx, &txtypes.GetTxRequest{Hash: txHash})
		if err == nil && resp != nil && resp.TxResponse != nil && resp.TxResponse.Txhash != "" {
			return Result{Code: resp.TxResponse.Code, Events: flattenEvents(resp.TxResponse)}, nil
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

func flattenEvents(resp *abcipb.TxResponse) map[string][]string {
	flat := make(map[string][]string)
	for _, e := range resp.Events {
		typeName := e.GetType_()
		for _, a := range e.GetAttributes() {
			key := typeName + "." + a.GetKey()
			flat[key] = append(flat[key], a.GetValue())
		}
	}
	return flat
}

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
