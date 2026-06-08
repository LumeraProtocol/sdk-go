package waittx

import (
	"context"
	"fmt"
	"time"

	txtypes "cosmossdk.io/api/cosmos/tx/v1beta1"
	clientconfig "github.com/LumeraProtocol/sdk-go/client/config"
)

// Querier fetches transactions over gRPC.
type Querier interface {
	GetTx(ctx context.Context, req *txtypes.GetTxRequest) (*txtypes.GetTxResponse, error)
}

// Waiter coordinates a subscriber (WS) and poller (gRPC) to observe a tx.
type Waiter struct {
	subscriber Source
	poller     Source
	setupDelay time.Duration
}

// New creates a waiter based on the provided config and querier.
func New(cfg clientconfig.WaitTxConfig, rpcEndpoint string, querier Querier) (*Waiter, error) {
	if querier == nil {
		return nil, fmt.Errorf("querier is required")
	}

	normalized := cfg
	clientconfig.ApplyWaitTxDefaults(&normalized)

	poller := newPoller(querier, normalized)

	var sub Source
	if rpcEndpoint != "" {
		sub = newSubscriber(rpcEndpoint)
	}

	return &Waiter{subscriber: sub, poller: poller, setupDelay: normalized.SubscriberSetupTimeout}, nil
}

// Wait blocks until the transaction reaches a final state or the context ends.
func (w *Waiter) Wait(ctx context.Context, txHash string, timeout time.Duration) (Result, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	var subErr error
	if w.subscriber != nil {
		// Bound the subscriber lifetime to setupDelay. If the WS handshake
		// or first event delivery has not landed by then we fall through to
		// the gRPC poller. Critically, the spawned goroutine MUST receive
		// subCtx (not the outer ctx) so that when this select returns via
		// the subCtx.Done() arm, the subscriber.Wait call inside the
		// goroutine unwinds promptly and its deferred client.Stop() runs.
		// Passing the outer ctx here was the root cause of the WS-socket
		// leak observed on lumera-devnet-1 val3 (~2562 sockets / 11 days):
		// subCtx fired after 5s, the caller fell through to the poller,
		// but the goroutine kept blocking on <-ch for the lifetime of the
		// outer ctx (often unbounded), pinning one rpchttp client open.
		subCtx, cancel := context.WithTimeout(ctx, w.setupDelay)
		defer cancel()
		resCh := make(chan Result, 1)
		errCh := make(chan error, 1)
		go func() {
			res, err := w.subscriber.Wait(subCtx, txHash)
			if err != nil {
				errCh <- err
				return
			}
			resCh <- res
		}()

		select {
		case <-subCtx.Done():
		case subErr = <-errCh:
		case res := <-resCh:
			return res, nil
		}
		cancel()
	}

	if w.poller == nil {
		return Result{}, fmt.Errorf("poller is required")
	}
	res, err := w.poller.Wait(ctx, txHash)
	// If the subscriber failed early and the poller also failed, surface both:
	// the subscriber error is often the more diagnostic root cause (e.g. WS
	// connection refused) and would otherwise be silently discarded.
	if err != nil && subErr != nil {
		return res, fmt.Errorf("subscriber failed (%v); poller failed: %w", subErr, err)
	}
	return res, err
}
