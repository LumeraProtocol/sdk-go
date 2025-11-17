//go:build ignore
// +build ignore

package waittx

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/LumeraProtocol/network-maker/pkg/log"
)

type TMClient interface {
	Start() error
	Stop() error
	Subscribe(ctx context.Context, subscriber, query string, outCapacity ...int) (<-chan ctypes.ResultEvent, error)
	Unsubscribe(ctx context.Context, subscriber, query string) error
}

// NewHTTPClient returns an HTTP+WS client for an endpoint like "http://127.0.0.1:26657".
func NewHTTPClient(endpoint string) (TMClient, error) {
	// rpchttp.New takes (httpURL, wsPath) and dials WS at httpURL + wsPath.
	return rpchttp.New(endpoint, "/websocket")
}

// formatTMHash ensures Tendermint v0.38-friendly tx.hash format: hex with "0x" prefix.
// Uppercase is fine; Tendermint matches case-insensitively for hex, but we normalize anyway.
func formatTMHash(h string) string {
	h = strings.TrimPrefix(h, "0x")
	return "0x" + strings.ToUpper(h)
}

// Subscriber listens for Tx events over WS and returns on first match (or ctx done).
type Subscriber struct {
	logger        log.Logger
	endpoint      string // e.g., "http://127.0.0.1:26657"
	subscriberID  string // unique name per subscription
	clientFactory func() (TMClient, error)
}

func NewSubscriber(logger log.Logger, endpoint, subscriberID string, f func() (TMClient, error)) *Subscriber {
	return &Subscriber{
		logger:        logger,
		endpoint:      endpoint,
		subscriberID:  subscriberID,
		clientFactory: f,
	}
}

func (s *Subscriber) Wait(ctx context.Context, txHash string) (Result, error) {
	newClient := s.clientFactory
	if newClient == nil {
		newClient = func() (TMClient, error) { return NewHTTPClient(s.endpoint) }
	}

	cl, err := newClient()
	if err != nil {
		return Result{}, fmt.Errorf("tm client init: %w", err)
	}
	if err := cl.Start(); err != nil {
		return Result{}, fmt.Errorf("tm client start: %w", err)
	}
	defer cl.Stop()

	// Tendermint v0.38.17: tx.hash must be hex **with** 0x prefix.
	q := fmt.Sprintf("tm.event='Tx' AND tx.hash='%s'", formatTMHash(txHash))

	evCh, err := cl.Subscribe(ctx, s.subscriberID, q)
	if err != nil {
		return Result{}, fmt.Errorf("subscribe: %w", err)
	}
	defer cl.Unsubscribe(context.Background(), s.subscriberID, q)

	for {
		select {
		case <-ctx.Done():
			return Result{}, ctx.Err()

		case ev := <-evCh:
			// v0.38.x: Data is tmtypes.EventDataTx
			txev, ok := ev.Data.(tmtypes.EventDataTx)
			if !ok {
				// Unexpected (e.g., NewBlock event)—ignore and keep waiting.
				continue
			}
			// log txev json for debugging
			txevJSON, _ := json.Marshal(txev)
			s.logger.Infof("Received Tx event: %s", string(txevJSON))

			// ABCI code lives under TxResult.Result.Code
			code := uint32(txev.TxResult.Result.Code)

			// Flatten ABCI events (TxResult.Result.Events) into map[type.key] = []values
			flat := make(map[string][]string)
			for _, e := range txev.TxResult.Result.Events {
				for _, a := range e.Attributes {
					key := e.Type + "." + string(a.Key)
					flat[key] = append(flat[key], string(a.Value))
				}
			}

			return Result{Code: code, Events: flat}, nil
		}
	}
}
