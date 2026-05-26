package waittx

import (
	"context"
	"errors"
	"testing"
	"time"

	clientconfig "github.com/LumeraProtocol/sdk-go/client/config"
	abcipb "cosmossdk.io/api/cosmos/base/abci/v1beta1"
	txtypes "cosmossdk.io/api/cosmos/tx/v1beta1"
)

type stubSource struct {
	res   Result
	err   error
	calls int
}

func (s *stubSource) Wait(ctx context.Context, txHash string) (Result, error) {
	s.calls++
	return s.res, s.err
}

func TestWaiterPrefersSubscriber(t *testing.T) {
	w := &Waiter{
		poller:     &stubSource{res: Result{Code: 1}},
		subscriber: &stubSource{res: Result{Code: 0}},
		setupDelay: 50 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := w.Wait(ctx, "hash", 0)
	if err != nil {
		t.Fatalf("wait error: %v", err)
	}
	if res.Code != 0 {
		t.Fatalf("expected subscriber result")
	}

	if w.poller.(*stubSource).calls != 0 {
		t.Fatalf("poller should not be used when subscriber succeeds")
	}
}

func TestWaiterFallsBackToPoller(t *testing.T) {
	poller := &stubSource{res: Result{Code: 2}}
	sub := &stubSource{err: errors.New("boom")}

	w := &Waiter{poller: poller, subscriber: sub, setupDelay: 10 * time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := w.Wait(ctx, "hash", 0)
	if err != nil {
		t.Fatalf("wait error: %v", err)
	}
	if res.Code != 2 {
		t.Fatalf("expected poller result")
	}
	if poller.calls == 0 {
		t.Fatalf("poller should have been invoked")
	}
}

type waiterStubQuerier struct {
	resp  *txtypes.GetTxResponse
	err   error
	calls int
}

func (s *waiterStubQuerier) GetTx(ctx context.Context, req *txtypes.GetTxRequest) (*txtypes.GetTxResponse, error) {
	s.calls++
	return s.resp, s.err
}

func TestNewSetsDefaults(t *testing.T) {
	resp := &txtypes.GetTxResponse{TxResponse: &abcipb.TxResponse{Txhash: "hash"}}
	q := &waiterStubQuerier{resp: resp}

	w, err := New(clientconfig.DefaultWaitTxConfig(), "ws://localhost:26657", q)
	if err != nil {
		t.Fatalf("new error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if _, err := w.Wait(ctx, "hash", 0); err != nil {
		t.Fatalf("unexpected wait error: %v", err)
	}
}

// blockingSource records the ctx it was called with and blocks on <-ctx.Done()
// without ever returning a Result. Used to verify the subscriber goroutine
// receives subCtx (so it unwinds promptly when setupDelay fires) rather than
// the outer ctx (which would leave it blocking until the outer ctx expires —
// the root-cause shape of the lumera-devnet-1 val3 WS-socket leak, RCA on file).
type blockingSource struct {
	received chan context.Context
	done     chan struct{}
}

func newBlockingSource() *blockingSource {
	return &blockingSource{received: make(chan context.Context, 1), done: make(chan struct{}, 1)}
}

func (b *blockingSource) Wait(ctx context.Context, _ string) (Result, error) {
	b.received <- ctx
	<-ctx.Done() // unwinds only when the supplied ctx fires
	b.done <- struct{}{}
	return Result{}, ctx.Err()
}

// TestWaiterSubscriberReceivesSubCtxNotOuter is the regression test for the
// WS-socket leak fix. Before the fix, the goroutine inside Waiter.Wait was
// invoked with the OUTER ctx; once setupDelay fired the caller fell through
// to the poller but the goroutine kept blocking on the outer ctx (often
// unbounded), pinning a CometBFT rpchttp client open and leaking a TCP socket
// to :26657 for the lifetime of the outer ctx.
//
// This test gives the subscriber a blockingSource that records its incoming
// ctx and asserts that, within a small multiple of setupDelay, the recorded
// ctx is Done — which is only true if the goroutine was bound to subCtx.
func TestWaiterSubscriberReceivesSubCtxNotOuter(t *testing.T) {
	sub := newBlockingSource()
	poller := &stubSource{res: Result{Code: 9}}
	w := &Waiter{
		subscriber: sub,
		poller:     poller,
		setupDelay: 25 * time.Millisecond,
	}

	// Outer ctx has a generous timeout. The test passes only if the
	// goroutine's ctx is cancelled when setupDelay (subCtx) fires, NOT when
	// the outer ctx fires.
	outer, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := w.Wait(outer, "hash", 0); err != nil {
		t.Fatalf("Wait should fall through to poller on subscriber timeout, got err=%v", err)
	}

	var recvCtx context.Context
	select {
	case recvCtx = <-sub.received:
	case <-time.After(time.Second):
		t.Fatal("subscriber.Wait was never invoked")
	}

	// Within ~10x setupDelay the goroutine's ctx MUST be Done. If it isn't,
	// the goroutine is leaking on the outer ctx (the bug).
	select {
	case <-recvCtx.Done():
		// good
	case <-time.After(250 * time.Millisecond):
		t.Fatal("subscriber goroutine's ctx never fired within 10x setupDelay; goroutine is bound to outer ctx (leak)")
	}

	// And the goroutine itself must have unwound (defer client.Stop() ran).
	select {
	case <-sub.done:
		// good
	case <-time.After(time.Second):
		t.Fatal("subscriber goroutine did not unwind after its ctx fired")
	}

	if poller.calls != 1 {
		t.Fatalf("poller should have been invoked exactly once; got %d", poller.calls)
	}
}
