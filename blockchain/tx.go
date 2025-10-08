package blockchain

import (
	"context"
	"fmt"

	txtypes "cosmossdk.io/api/cosmos/tx/v1beta1"
)

// Simulate runs a gas simulation for a provided tx bytes
func (c *Client) Simulate(ctx context.Context, txBytes []byte) (uint64, error) {
	svc := txtypes.NewServiceClient(c.conn)
	resp, err := svc.Simulate(ctx, &txtypes.SimulateRequest{
		TxBytes: txBytes,
	})
	if err != nil {
		return 0, fmt.Errorf("simulate tx: %w", err)
	}
	if resp == nil || resp.GasInfo == nil {
		return 0, nil
	}
	return resp.GasInfo.GasUsed, nil
}

// Broadcast broadcasts a signed transaction with a chosen broadcast mode
func (c *Client) Broadcast(ctx context.Context, txBytes []byte, mode txtypes.BroadcastMode) (string, error) {
	svc := txtypes.NewServiceClient(c.conn)
	resp, err := svc.BroadcastTx(ctx, &txtypes.BroadcastTxRequest{
		TxBytes: txBytes,
		Mode:    mode,
	})
	if err != nil {
		return "", fmt.Errorf("broadcast tx: %w", err)
	}

	if resp == nil || resp.TxResponse == nil {
		return "", fmt.Errorf("empty tx response")
	}

	if resp.TxResponse.Code != 0 {
		return "", fmt.Errorf("tx failed with code %d: %s", resp.TxResponse.Code, resp.TxResponse.RawLog)
	}

	return resp.TxResponse.GetTxhash(), nil
}
