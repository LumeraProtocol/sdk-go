//go:build ignore
// +build ignore

package client

import (
	"context"
	"fmt"
	"time"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/LumeraProtocol/network-maker/config"
	waittx "github.com/LumeraProtocol/network-maker/lumera/client/wait-tx"
)

// GetAccount gets the account number and sequence for an address (used in manual flows).
func (c *clientImpl) GetAccount(ctx context.Context, addr string) (number uint64, sequence uint64, err error) {
	if addr == "" {
		return 0, 0, fmt.Errorf("address is empty")
	}
	resp, err := c.acctQ.AccountInfo(ctx, &authtypes.QueryAccountInfoRequest{Address: addr})
	if err != nil {
		return 0, 0, err
	}
	return resp.Info.GetAccountNumber(), resp.Info.GetSequence(), nil
}

// BuildSignBroadcast takes arbitrary msgs (action, approve, finalize, etc.),
// signs the tx with the configured keyring, broadcasts via the tx service,
// and returns a tx response (or error).
func (c *clientImpl) BuildSignBroadcast(ctx context.Context, msgs []sdktypes.Msg) (*sdktypes.TxResponse, error) {
	if len(msgs) == 0 {
		return nil, fmt.Errorf("no messages to send")
	}

	// Fetch account info for the configured signer (helper utility).
	accInfo, err := c.txh.GetAccountInfo(ctx)
	if err != nil {
		return nil, err
	}

	// Run the full flow (simulate → sign → broadcast) via helper.
	br, err := c.txh.ExecuteTransactionWithMsgs(ctx, msgs, accInfo)
	if err != nil {
		return nil, err
	}
	if br.TxResponse == nil {
		return nil, fmt.Errorf("empty tx response from broadcast")
	}
	return br.TxResponse, nil
}

// WaitTx waits for a tx hash to reach a final state or timeout.
// It tries tendermint subscription though WebSocket first (if cfg has an RPC endpoint), then falls back to polling GetTx.
func (c *clientImpl) WaitTx(ctx context.Context, txHash string, timeout time.Duration) (uint32, map[string][]string, error) {
	waitCfg := config.WaitTxOptions{
		PollInitialDelay:       1 * time.Second,
		PollMultiplier:         1.5,
		PollMaxDelay:           20 * time.Second,
		PollJitter:             0.1,
		PollMaxTries:           0,
		SubscriberSetupTimeout: 5 * time.Second,
	}
	if c.cfg != nil && c.cfg.CfgOpts != nil {
		waitCfg = c.cfg.GetWaitTxOptions()
	}

	// 1) Always have a poller fallback (gRPC tx service).
	poller := waittx.NewPoller(
		TxQuerierFunc(func(ctx context.Context, req *sdktx.GetTxRequest) (*sdktx.GetTxResponse, error) {
			return c.txS.GetTx(ctx, req)
		}),
		waittx.ExponentialBackoff{
			Initial:    waitCfg.PollInitialDelay,
			Multiplier: waitCfg.PollMultiplier,
			Max:        waitCfg.PollMaxDelay,
			Jitter:     waitCfg.PollJitter,
		},
		waitCfg.PollMaxTries,
	)

	// 2) Optional subscriber if RPC HTTP is configured (e.g., "http://127.0.0.1:26657").
	var opts []waittx.WaiterOption
	opts = append(opts, waittx.WithPoller(poller))
	if c.cfg != nil && c.cfg.CfgOpts != nil {
		rpcHTTP := c.cfg.CfgOpts.Lumera.RPCHTTP
		if rpcHTTP != "" {
			sub := waittx.NewSubscriber(c.logger, rpcHTTP, "nm-waittx", nil) // nil factory → real client
			opts = append(opts, waittx.WithSubscriber(sub, waitCfg.SubscriberSetupTimeout))
		}
	}

	w, err := waittx.NewWaiter(opts...)
	if err != nil {
		return 0, nil, err
	}

	res, err := w.WaitTx(ctx, txHash, timeout)
	return res.Code, res.Events, err
}

// TxQuerierFunc lets us adapt the gRPC client to the Poller interface.
type TxQuerierFunc func(ctx context.Context, req *sdktx.GetTxRequest) (*sdktx.GetTxResponse, error)

func (f TxQuerierFunc) GetTx(ctx context.Context, req *sdktx.GetTxRequest) (*sdktx.GetTxResponse, error) {
	return f(ctx, req)
}
