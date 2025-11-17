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
