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

	if w.subscriber != nil {
		subCtx, cancel := context.WithTimeout(ctx, w.setupDelay)
		defer cancel()
		resCh := make(chan Result, 1)
		errCh := make(chan error, 1)
		go func() {
			res, err := w.subscriber.Wait(ctx, txHash)
			if err != nil {
				errCh <- err
				return
			}
			resCh <- res
		}()

		select {
		case <-subCtx.Done():
		case <-errCh:
		case res := <-resCh:
			return res, nil
		}
		cancel()
	}

	if w.poller == nil {
		return Result{}, fmt.Errorf("poller is required")
	}
	return w.poller.Wait(ctx, txHash)
}
