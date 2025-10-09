package blockchain

import (
	"context"
	"fmt"
	"math"
	"time"

	txtypes "cosmossdk.io/api/cosmos/tx/v1beta1"
	sdkcrypto "github.com/LumeraProtocol/sdk-go/internal/crypto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

// BuildAndSignTx builds a transaction with one message, simulates gas, then signs it.
func (c *Client) BuildAndSignTx(ctx context.Context, msg sdk.Msg, memo string) ([]byte, error) {
	// 1) Tx config and builder
	txCfg := sdkcrypto.NewDefaultTxConfig()
	builder := txCfg.NewTxBuilder()
	if err := builder.SetMsgs(msg); err != nil {
		return nil, fmt.Errorf("set msgs: %w", err)
	}
	if memo != "" {
		builder.SetMemo(memo)
	}

	// 2) Resolve account number/sequence BEFORE simulation
	rec, err := c.keyring.Key(c.keyName)
	if err != nil {
		return nil, fmt.Errorf("load key %q: %w", c.keyName, err)
	}
	accAddr, err := rec.GetAddress()
	if err != nil {
		return nil, fmt.Errorf("get address for %q: %w", c.keyName, err)
	}

	authq := authtypes.NewQueryClient(c.conn)
	acctResp, err := authq.AccountInfo(ctx, &authtypes.QueryAccountInfoRequest{
		Address: accAddr.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("query account info: %w", err)
	}
	if acctResp == nil || acctResp.Info == nil {
		return nil, fmt.Errorf("empty account info response")
	}

	// 3) Build placeholder signature using real sequence
	pk, err := rec.GetPubKey()
	if err != nil {
		return nil, fmt.Errorf("get pubkey for %q: %w", c.keyName, err)
	}
	signMode := txCfg.SignModeHandler().DefaultMode()
	placeholder := signingtypes.SignatureV2{
		PubKey: pk,
		Data: &signingtypes.SingleSignatureData{
			SignMode: signingtypes.SignMode(signMode),
		},
		Sequence: acctResp.Info.Sequence, // use real sequence for simulation
	}
	if err := builder.SetSignatures(placeholder); err != nil {
		return nil, fmt.Errorf("set placeholder signature: %w", err)
	}

	// 4) Simulate with placeholder to get gas
	unsignedBytes, err := txCfg.TxEncoder()(builder.GetTx())
	if err != nil {
		return nil, fmt.Errorf("encode unsigned tx: %w", err)
	}

	gasUsed, err := c.Simulate(ctx, unsignedBytes)
	gas := uint64(0)
	if err == nil && gasUsed > 0 {
		// add a 30% buffer
		gas = uint64(float64(gasUsed) * 1.3)
		if gas == 0 {
			gas = gasUsed
		}
	} else {
		// On simulation failure, proceed with a conservative default gas
		if builder.GetTx().GetGas() == 0 {
			gas = 200000
		}
	}
	builder.SetGasLimit(gas)

	err = builder.SetSignatures() // clear placeholder signature
	if err != nil {
		return nil, fmt.Errorf("clear placeholder signature: %w", err)
	}

	// Ensure a minimum fee to satisfy chain requirements
	fee := int64(math.Ceil(float64(gas) / 40.0)) //the gas price is 0.025
	minFee := sdk.NewCoins(sdk.NewInt64Coin("ulume", fee))
	builder.SetFeeAmount(minFee)

	// 5) Sign with real credentials, overwriting placeholder
	if err := sdkcrypto.SignTxWithKeyring(
		ctx, txCfg, c.keyring, c.keyName, builder,
		c.config.ChainID, acctResp.Info.AccountNumber, acctResp.Info.Sequence, true,
	); err != nil {
		return nil, fmt.Errorf("sign tx: %w", err)
	}

	// 6) Encode signed tx
	signedBytes, err := txCfg.TxEncoder()(builder.GetTx())
	if err != nil {
		return nil, fmt.Errorf("encode signed tx: %w", err)
	}

	return signedBytes, nil
}

// GetTx fetches a transaction by hash via the tx service.
func (c *Client) GetTx(ctx context.Context, hash string) (*txtypes.GetTxResponse, error) {
	svc := txtypes.NewServiceClient(c.conn)
	resp, err := svc.GetTx(ctx, &txtypes.GetTxRequest{Hash: hash})
	if err != nil {
		return nil, fmt.Errorf("get tx: %w", err)
	}
	if resp == nil {
		return nil, fmt.Errorf("empty get tx response")
	}
	return resp, nil
}

// WaitForTxInclusion polls GetTx until the transaction is included in a block.
// It treats codes.NotFound as "not yet included" and continues polling every 1 second.
// Returns the transaction response once it's confirmed with Height > 0.
// Respects context cancellation/timeout.
func (c *Client) WaitForTxInclusion(ctx context.Context, txHash string) (*txtypes.GetTxResponse, error) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			resp, err := c.GetTx(ctx, txHash)
			if err != nil {
				// Check if the error is NotFound - this means tx not yet included
				if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
					continue // Keep polling
				}
				// Any other error should be returned
				return nil, fmt.Errorf("get tx: %w", err)
			}
			// Validate we have a valid response with inclusion proof
			if resp != nil && resp.TxResponse != nil && resp.TxResponse.Height > 0 {
				return resp, nil
			}
		}
	}
}

// ExtractEventAttribute extracts an attribute value from transaction events.
// It searches through TxResponse.Events for the first event matching eventType,
// then returns the value of the first attribute matching attrKey.
// Returns an error if the transaction, events, or matching event/attribute are not found.
func (c *Client) ExtractEventAttribute(tx *txtypes.GetTxResponse, eventType, attrKey string) (string, error) {
	if tx == nil || tx.TxResponse == nil {
		return "", fmt.Errorf("nil tx or tx response")
	}
	events := tx.TxResponse.GetEvents()
	if len(events) == 0 {
		return "", fmt.Errorf("no events in tx response")
	}
	for _, ev := range events {
		if ev == nil {
			continue
		}
		// Note: abci.Event uses GetType_() since 'type' is a reserved field name
		if ev.GetType_() == eventType {
			for _, attr := range ev.GetAttributes() {
				if attr == nil {
					continue
				}
				if attr.GetKey() == attrKey {
					return attr.GetValue(), nil
				}
			}
		}
	}
	return "", fmt.Errorf("attribute %q not found in event type %q", attrKey, eventType)
}
