package waittx

import (
	"context"
	"errors"
	"testing"
	"time"

	abcipb "cosmossdk.io/api/cosmos/base/abci/v1beta1"
	txtypes "cosmossdk.io/api/cosmos/tx/v1beta1"
)

type stubQuerier struct {
	resp  *txtypes.GetTxResponse
	err   error
	calls int
}

func (s *stubQuerier) GetTx(ctx context.Context, req *txtypes.GetTxRequest) (*txtypes.GetTxResponse, error) {
	s.calls++
	return s.resp, s.err
}

func TestPollerStopsAfterMaxRetries(t *testing.T) {
	p := &poller{
		querier:  &stubQuerier{err: errors.New("unavailable")},
		backoff:  constantBackoff{every: time.Millisecond},
		maxTries: 3,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if _, err := p.Wait(ctx, "hash"); err == nil {
		t.Fatalf("expected error when retries exhausted")
	}
}

func TestPollerReturnsSuccessfulResult(t *testing.T) {
	resp := &txtypes.GetTxResponse{TxResponse: &abcipb.TxResponse{
		Txhash: "hash",
		Code:   9,
	}}
	p := &poller{
		querier:  &stubQuerier{resp: resp},
		backoff:  constantBackoff{every: 0},
		maxTries: 1,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := p.Wait(ctx, "hash")
	if err != nil {
		t.Fatalf("wait error: %v", err)
	}
	if res.Code != resp.TxResponse.Code {
		t.Fatalf("unexpected code: %d", res.Code)
	}
}

func TestExponentialBackoffSequence(t *testing.T) {
	b := &exponentialBackoff{
		initial:    time.Second,
		multiplier: 2,
		max:        5 * time.Second,
	}
	want := []time.Duration{
		time.Second,
		2 * time.Second,
		4 * time.Second,
		5 * time.Second,
		5 * time.Second,
	}
	for i, expected := range want {
		if got := b.Next(i + 1); got != expected {
			t.Fatalf("attempt %d: want %v; got %v", i+1, expected, got)
		}
	}
}

func TestExponentialBackoffJitter(t *testing.T) {
	base := time.Second
	b := &exponentialBackoff{
		initial: base,
		jitter:  0.5,
	}
	b.randFn = func() float64 { return 0 }
	if got := b.Next(1); got != base/2 {
		t.Fatalf("jitter low bound: want %v; got %v", base/2, got)
	}
	b.randFn = func() float64 { return 1 }
	if got := b.Next(1); got != base+base/2 {
		t.Fatalf("jitter high bound: want %v; got %v", base+base/2, got)
	}
}

func TestExponentialBackoffDefaults(t *testing.T) {
	b := &exponentialBackoff{}
	if got := b.Next(0); got != 500*time.Millisecond {
		t.Fatalf("default initial: want %v; got %v", 500*time.Millisecond, got)
	}

	b = &exponentialBackoff{multiplier: 0.5}
	if got := b.Next(3); got != 500*time.Millisecond {
		t.Fatalf("multiplier <= 1 should not shrink delay: want %v; got %v", 500*time.Millisecond, got)
	}
}
