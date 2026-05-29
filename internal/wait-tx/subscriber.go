package waittx

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	tmtypes "github.com/cometbft/cometbft/types"
)

// subscriberSeq generates unique CometBFT subscriber IDs per process. A
// constant subscriberID across concurrent callers risked client-side
// collisions in cometbft's WSEvents.subscriptions[query] map. Even with one
// rpchttp.Client per call (today), rotation keeps the design forward-safe if
// callers ever share clients (see RCA item 6b).
var subscriberSeq uint64

func newSubscriberID() string {
	return fmt.Sprintf("sdk-go-wait-%d-%d", os.Getpid(), atomic.AddUint64(&subscriberSeq, 1))
}

// unsubscribeTimeout bounds the lifetime of the deferred Unsubscribe call so
// teardown cannot block on an unresponsive server forever.
const unsubscribeTimeout = 2 * time.Second

type subscriber struct {
	endpoint string
}

func newSubscriber(endpoint string) Source {
	return &subscriber{endpoint: endpoint}
}

func (s *subscriber) Wait(ctx context.Context, txHash string) (Result, error) {
	client, err := rpchttp.New(s.endpoint, "/websocket")
	if err != nil {
		return Result{}, fmt.Errorf("tm client init: %w", err)
	}
	if err := client.Start(); err != nil {
		return Result{}, fmt.Errorf("tm client start: %w", err)
	}
	// Always stop the rpchttp client, even if Subscribe fails or ctx fires.
	// This is the single point that closes the underlying gorilla WS conn
	// (WSEvents.OnStop -> ws.Stop). Without this, on any error path the
	// websocket to lumerad's :26657 stays ESTABLISHED until the OS or peer
	// times out (~hours). See ops RCA on lumera-devnet-1 val3 leak (~2562
	// sockets / 11 days).
	defer func() { _ = client.Stop() }()

	id := newSubscriberID()
	query := fmt.Sprintf("tm.event='Tx' AND tx.hash='%s'", formatTMHash(txHash))
	ch, err := client.Subscribe(ctx, id, query)
	if err != nil {
		return Result{}, fmt.Errorf("subscribe: %w", err)
	}
	// Bounded Unsubscribe context: the outer ctx may already be cancelled by
	// the time this defer runs, and Background() with no timeout could let a
	// slow server block teardown indefinitely. 2s is plenty for an unsubscribe
	// JSON-RPC round-trip; if it fails we still proceed to client.Stop() which
	// tears down the socket regardless.
	defer func() {
		unsubCtx, cancel := context.WithTimeout(context.Background(), unsubscribeTimeout)
		defer cancel()
		_ = client.Unsubscribe(unsubCtx, id, query)
	}()

	for {
		select {
		case <-ctx.Done():
			return Result{}, ctx.Err()
		case ev := <-ch:
			txev, ok := ev.Data.(tmtypes.EventDataTx)
			if !ok {
				continue
			}
			flat := make(map[string][]string)
			for _, e := range txev.Result.Events {
				for _, a := range e.Attributes {
					key := e.Type + "." + string(a.Key)
					flat[key] = append(flat[key], string(a.Value))
				}
			}
			return Result{Code: uint32(txev.Result.Code), Events: flat}, nil
		}
	}
}

func formatTMHash(h string) string {
	h = strings.TrimPrefix(h, "0x")
	return "0x" + strings.ToUpper(h)
}
