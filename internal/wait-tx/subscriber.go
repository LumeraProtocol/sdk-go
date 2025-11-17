package waittx

import (
	"context"
	"fmt"
	"strings"

	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	tmtypes "github.com/cometbft/cometbft/types"
)

const subscriberID = "sdk-go-wait"

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
	defer client.Stop() //nolint:errcheck

	query := fmt.Sprintf("tm.event='Tx' AND tx.hash='%s'", formatTMHash(txHash))
	ch, err := client.Subscribe(ctx, subscriberID, query)
	if err != nil {
		return Result{}, fmt.Errorf("subscribe: %w", err)
	}
	defer client.Unsubscribe(context.Background(), subscriberID, query) //nolint:errcheck

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
			for _, e := range txev.TxResult.Result.Events {
				for _, a := range e.Attributes {
					key := e.Type + "." + string(a.Key)
					flat[key] = append(flat[key], string(a.Value))
				}
			}
			return Result{Code: uint32(txev.TxResult.Result.Code), Events: flat}, nil
		}
	}
}

func formatTMHash(h string) string {
	h = strings.TrimPrefix(h, "0x")
	return "0x" + strings.ToUpper(h)
}
