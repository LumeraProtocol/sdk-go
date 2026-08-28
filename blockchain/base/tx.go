package base

import (
	"context"
	"fmt"
	"strings"
	"time"

	txtypes "cosmossdk.io/api/cosmos/tx/v1beta1"
	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	waittx "github.com/LumeraProtocol/sdk-go/internal/wait-tx"
	sdkcrypto "github.com/LumeraProtocol/sdk-go/pkg/crypto"
)

const defaultSignedTxGasLimit = 200000

var msgEthereumTxTypeURL = sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{})

// TxBuildOptions controls how a transaction is assembled and signed.
type TxBuildOptions struct {
	Messages       []sdk.Msg
	Memo           string
	GasAdjustment  float64
	GasLimit       uint64
	SkipSimulation bool
	AccountNumber  *uint64
	Sequence       *uint64
	FeeAmount      sdk.Coins
	ZeroFee        bool
}

// TxSignerInfo contains the signer account metadata used for signing.
type TxSignerInfo struct {
	AccountNumber uint64
	Sequence      uint64
}

// Simulate runs a gas simulation for a provided tx bytes.
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

// Broadcast broadcasts a signed transaction with a chosen broadcast mode.
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

// BroadcastAndWait broadcasts signed tx bytes, then waits for final inclusion.
func (c *Client) BroadcastAndWait(ctx context.Context, txBytes []byte, mode txtypes.BroadcastMode) (string, *txtypes.GetTxResponse, error) {
	txHash, err := c.Broadcast(ctx, txBytes, mode)
	if err != nil {
		return "", nil, err
	}

	resp, err := c.WaitForTxInclusion(ctx, txHash)
	if err != nil {
		return txHash, nil, err
	}
	if resp == nil || resp.TxResponse == nil {
		return txHash, nil, fmt.Errorf("empty tx response")
	}
	if resp.TxResponse.Code != 0 {
		return txHash, resp, fmt.Errorf("tx failed with code %d: %s", resp.TxResponse.Code, resp.TxResponse.RawLog)
	}

	return txHash, resp, nil
}

// BuildAndSignTx builds a transaction with one message, simulates gas, then signs it.
func (c *Client) BuildAndSignTx(ctx context.Context, msg sdk.Msg, memo string) ([]byte, error) {
	return c.BuildAndSignTxWithOptions(ctx, TxBuildOptions{
		Messages:      []sdk.Msg{msg},
		Memo:          memo,
		GasAdjustment: 1.3,
	})
}

// BuildAndSignTxWithGasAdjustment builds a transaction with one message, simulates gas,
// applies a custom adjustment factor, then signs it.
func (c *Client) BuildAndSignTxWithGasAdjustment(ctx context.Context, msg sdk.Msg, memo string, gasAdjustment float64) ([]byte, error) {
	if gasAdjustment <= 0 {
		gasAdjustment = 1.3
	}
	return c.BuildAndSignTxWithOptions(ctx, TxBuildOptions{
		Messages:      []sdk.Msg{msg},
		Memo:          memo,
		GasAdjustment: gasAdjustment,
	})
}

// BuildAndSignTxWithOptions builds and signs a transaction using explicit options.
func (c *Client) BuildAndSignTxWithOptions(ctx context.Context, opts TxBuildOptions) ([]byte, error) {
	txCfg, builder, signerInfo, err := c.PrepareTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return c.SignPreparedTx(ctx, txCfg, builder, signerInfo)
}

// PrepareTx builds an unsigned tx builder and resolves the signer metadata.
func (c *Client) PrepareTx(ctx context.Context, opts TxBuildOptions) (client.TxConfig, client.TxBuilder, TxSignerInfo, error) {
	txCfg := sdkcrypto.NewDefaultTxConfig()
	builder := txCfg.NewTxBuilder()

	if err := c.validateTxBuildOptions(opts); err != nil {
		return nil, nil, TxSignerInfo{}, err
	}
	if err := builder.SetMsgs(opts.Messages...); err != nil {
		return nil, nil, TxSignerInfo{}, fmt.Errorf("set msgs: %w", err)
	}
	if opts.Memo != "" {
		builder.SetMemo(opts.Memo)
	}

	rec, signerInfo, err := c.resolveSignerInfo(ctx, opts)
	if err != nil {
		return nil, nil, TxSignerInfo{}, err
	}

	pk, err := rec.GetPubKey()
	if err != nil {
		return nil, nil, TxSignerInfo{}, fmt.Errorf("get pubkey for %q: %w", c.keyName, err)
	}
	signMode := txCfg.SignModeHandler().DefaultMode()
	placeholder := signingtypes.SignatureV2{
		PubKey: pk,
		Data: &signingtypes.SingleSignatureData{
			SignMode: signingtypes.SignMode(signMode),
		},
		Sequence: signerInfo.Sequence,
	}
	if err := builder.SetSignatures(placeholder); err != nil {
		return nil, nil, TxSignerInfo{}, fmt.Errorf("set placeholder signature: %w", err)
	}

	gas, err := c.resolveGasLimit(ctx, txCfg, builder, opts)
	if err != nil {
		return nil, nil, TxSignerInfo{}, err
	}
	builder.SetGasLimit(gas)

	if err := builder.SetSignatures(); err != nil {
		return nil, nil, TxSignerInfo{}, fmt.Errorf("clear placeholder signature: %w", err)
	}

	feeAmount, err := c.resolveFeeAmount(gas, opts)
	if err != nil {
		return nil, nil, TxSignerInfo{}, err
	}
	builder.SetFeeAmount(feeAmount)

	return txCfg, builder, signerInfo, nil
}

// SignPreparedTx signs a prepared tx builder using explicit signer info.
func (c *Client) SignPreparedTx(ctx context.Context, txCfg client.TxConfig, builder client.TxBuilder, signerInfo TxSignerInfo) ([]byte, error) {
	if c.keyring == nil {
		return nil, fmt.Errorf("keyring is required")
	}
	if strings.TrimSpace(c.keyName) == "" {
		return nil, fmt.Errorf("key name is required")
	}
	if err := sdkcrypto.SignTxWithKeyring(
		ctx, txCfg, c.keyring, c.keyName, builder,
		c.config.ChainID, signerInfo.AccountNumber, signerInfo.Sequence, true,
	); err != nil {
		return nil, fmt.Errorf("sign tx: %w", err)
	}

	signedBytes, err := txCfg.TxEncoder()(builder.GetTx())
	if err != nil {
		return nil, fmt.Errorf("encode signed tx: %w", err)
	}

	return signedBytes, nil
}

func (c *Client) validateTxBuildOptions(opts TxBuildOptions) error {
	if c.keyring == nil {
		return fmt.Errorf("keyring is required")
	}
	if strings.TrimSpace(c.keyName) == "" {
		return fmt.Errorf("key name is required")
	}
	if len(opts.Messages) == 0 {
		return fmt.Errorf("at least one message is required")
	}
	if strings.TrimSpace(c.config.AccountHRP) == "" {
		return fmt.Errorf("account HRP is required")
	}
	if opts.ZeroFee && !opts.FeeAmount.Empty() {
		return fmt.Errorf("zero fee cannot be combined with explicit fee amount")
	}
	if !opts.ZeroFee && opts.FeeAmount.Empty() {
		if strings.TrimSpace(c.config.FeeDenom) == "" {
			return fmt.Errorf("fee denom is required")
		}
		if c.config.GasPrice.IsNil() || c.config.GasPrice.IsZero() {
			return fmt.Errorf("gas price is required")
		}
	}
	for _, msg := range opts.Messages {
		if sdk.MsgTypeURL(msg) == msgEthereumTxTypeURL {
			return fmt.Errorf("MsgEthereumTx must use EVMClient.SendEthereumTransaction; the cosmos signing pipeline rejects it")
		}
	}
	return nil
}

func (c *Client) resolveSignerInfo(ctx context.Context, opts TxBuildOptions) (*keyring.Record, TxSignerInfo, error) {
	rec, err := c.keyring.Key(c.keyName)
	if err != nil {
		return nil, TxSignerInfo{}, fmt.Errorf("load key %q: %w", c.keyName, err)
	}

	info := TxSignerInfo{}
	if opts.AccountNumber != nil {
		info.AccountNumber = *opts.AccountNumber
	}
	if opts.Sequence != nil {
		info.Sequence = *opts.Sequence
	}
	if opts.AccountNumber != nil && opts.Sequence != nil {
		return rec, info, nil
	}

	accAddr, err := sdkcrypto.AddressFromKey(c.keyring, c.keyName, c.config.AccountHRP)
	if err != nil {
		return nil, TxSignerInfo{}, fmt.Errorf("derive address for %q: %w", c.keyName, err)
	}

	authq := authtypes.NewQueryClient(c.conn)
	acctResp, err := authq.AccountInfo(ctx, &authtypes.QueryAccountInfoRequest{
		Address: accAddr,
	})
	if err != nil {
		return nil, TxSignerInfo{}, fmt.Errorf("query account info: %w", err)
	}
	if acctResp == nil || acctResp.Info == nil {
		return nil, TxSignerInfo{}, fmt.Errorf("empty account info response")
	}
	if opts.AccountNumber == nil {
		info.AccountNumber = acctResp.Info.AccountNumber
	}
	if opts.Sequence == nil {
		info.Sequence = acctResp.Info.Sequence
	}

	return rec, info, nil
}

func (c *Client) resolveGasLimit(ctx context.Context, txCfg client.TxConfig, builder client.TxBuilder, opts TxBuildOptions) (uint64, error) {
	if opts.GasLimit > 0 {
		return opts.GasLimit, nil
	}
	if opts.SkipSimulation {
		return defaultSignedTxGasLimit, nil
	}

	unsignedBytes, err := txCfg.TxEncoder()(builder.GetTx())
	if err != nil {
		return 0, fmt.Errorf("encode unsigned tx: %w", err)
	}

	gasUsed, simErr := c.Simulate(ctx, unsignedBytes)
	if simErr != nil {
		// Don't silently fall back to a fixed gas limit: a failed simulation
		// usually means the tx would fail on-chain, and guessing a default
		// can under-gas it and produce an opaque on-chain error. Surface the
		// reason and let callers opt out via SkipSimulation or an explicit
		// GasLimit.
		return 0, fmt.Errorf("simulate tx for gas estimation (set GasLimit or SkipSimulation to bypass): %w", simErr)
	}
	if gasUsed == 0 {
		return defaultSignedTxGasLimit, nil
	}

	gasAdjustment := opts.GasAdjustment
	if gasAdjustment <= 0 {
		gasAdjustment = 1.3
	}
	gas := uint64(float64(gasUsed) * gasAdjustment)
	if gas == 0 {
		gas = gasUsed
	}
	return gas, nil
}

func (c *Client) resolveFeeAmount(gas uint64, opts TxBuildOptions) (sdk.Coins, error) {
	if opts.ZeroFee {
		return nil, nil
	}
	if !opts.FeeAmount.Empty() {
		return opts.FeeAmount, nil
	}
	feeDec := c.config.GasPrice.MulInt(sdkmath.NewIntFromUint64(gas)).Ceil().TruncateInt()
	return sdk.NewCoins(sdk.NewCoin(c.config.FeeDenom, feeDec)), nil
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

// GetTxsByEvents searches for transactions matching event filters.
func (c *Client) GetTxsByEvents(ctx context.Context, events []string, page, limit uint64) (*txtypes.GetTxsEventResponse, error) {
	if len(events) == 0 {
		return nil, fmt.Errorf("events are required")
	}
	filtered := make([]string, 0, len(events))
	for _, evt := range events {
		evt = strings.TrimSpace(evt)
		if evt != "" {
			filtered = append(filtered, evt)
		}
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf("events are required")
	}
	if page == 0 {
		page = 1
	}
	query := strings.Join(filtered, " AND ")
	req := &txtypes.GetTxsEventRequest{
		Query:   query,
		OrderBy: txtypes.OrderBy_ORDER_BY_DESC,
	}
	req.Page = page
	req.Limit = limit
	svc := txtypes.NewServiceClient(c.conn)
	resp, err := svc.GetTxsEvent(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get txs by events: %w", err)
	}
	if resp == nil {
		return nil, fmt.Errorf("empty get txs by events response")
	}
	return resp, nil
}

// WaitForTxInclusion waits for a transaction to reach a final state using a
// websocket subscriber when possible and falling back to periodic gRPC polling.
// A new waiter (and therefore a new websocket subscription) is created for each
// invocation, so sequential callers should expect a new CometBFT RPC client
// per call. Timeouts are driven entirely by the caller-provided context (the
// waiter timeout argument remains zero intentionally). It respects the context
// for cancellation or deadlines.
func (c *Client) WaitForTxInclusion(ctx context.Context, txHash string) (*txtypes.GetTxResponse, error) {
	w, err := waittx.New(c.config.WaitTx, c.config.RPCEndpoint, txQuerierFunc(func(ctx context.Context, req *txtypes.GetTxRequest) (*txtypes.GetTxResponse, error) {
		return c.GetTx(ctx, req.GetHash())
	}))
	if err != nil {
		return nil, err
	}

	if _, err := w.Wait(ctx, txHash, 0); err != nil {
		return nil, err
	}

	backoff := waittx.NewBackoff(c.config.WaitTx)
	attempt := 0
	maxTries := c.config.WaitTx.PollMaxRetries

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		resp, err := c.GetTx(ctx, txHash)
		if err == nil {
			return resp, nil
		}

		if status.Code(err) != codes.NotFound {
			return nil, err
		}

		attempt++
		if maxTries > 0 && attempt >= maxTries {
			return nil, fmt.Errorf("get tx polling exhausted after %d attempts: %w", attempt, err)
		}

		delay := backoff.Next(attempt)
		if delay <= 0 {
			continue
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

type txQuerierFunc func(ctx context.Context, req *txtypes.GetTxRequest) (*txtypes.GetTxResponse, error)

func (f txQuerierFunc) GetTx(ctx context.Context, req *txtypes.GetTxRequest) (*txtypes.GetTxResponse, error) {
	return f(ctx, req)
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
		// Note: abci.Event uses GetType_() since 'type' is a reserved field name.
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
