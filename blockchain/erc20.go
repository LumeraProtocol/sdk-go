package blockchain

import (
	"context"
	"fmt"

	txtypes "cosmossdk.io/api/cosmos/tx/v1beta1"
	sdkmath "cosmossdk.io/math"
	sdkcrypto "github.com/LumeraProtocol/sdk-go/pkg/crypto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	erc20types "github.com/cosmos/evm/x/erc20/types"
	"github.com/ethereum/go-ethereum/common"
)

// ERC20Client provides x/erc20 module operations: queries plus opinionated
// tx helpers for the Cosmos<->ERC20 conversion flows.
type ERC20Client struct {
	query  erc20types.QueryClient
	client *Client // backref for tx helpers
}

// ConversionResult captures the outcome of a coin <-> ERC20 conversion.
type ConversionResult struct {
	From   string
	To     string
	Amount sdkmath.Int
	TxHash string
	Height int64
}

// --- queries ---

// TokenPairs returns the registered token pairs with optional pagination.
func (c *ERC20Client) TokenPairs(ctx context.Context, pagination *query.PageRequest) ([]erc20types.TokenPair, *query.PageResponse, error) {
	resp, err := c.query.TokenPairs(ctx, &erc20types.QueryTokenPairsRequest{Pagination: pagination})
	if err != nil {
		return nil, nil, err
	}
	if resp == nil {
		return nil, nil, nil
	}
	return resp.TokenPairs, resp.Pagination, nil
}

// TokenPair returns the registered pair for a given denom or 0x contract.
func (c *ERC20Client) TokenPair(ctx context.Context, token string) (erc20types.TokenPair, error) {
	resp, err := c.query.TokenPair(ctx, &erc20types.QueryTokenPairRequest{Token: token})
	if err != nil {
		return erc20types.TokenPair{}, err
	}
	if resp == nil {
		return erc20types.TokenPair{}, fmt.Errorf("empty token pair response")
	}
	return resp.TokenPair, nil
}

// Params returns the current x/erc20 module parameters.
func (c *ERC20Client) Params(ctx context.Context) (*erc20types.QueryParamsResponse, error) {
	return c.query.Params(ctx, &erc20types.QueryParamsRequest{})
}

// --- message constructors ---

// NewMsgConvertCoin builds a MsgConvertCoin (cosmos coin -> ERC20).
func NewMsgConvertCoin(coin sdk.Coin, receiver common.Address, sender string) *erc20types.MsgConvertCoin {
	return &erc20types.MsgConvertCoin{
		Coin:     coin,
		Receiver: receiver.Hex(),
		Sender:   sender,
	}
}

// NewMsgConvertERC20 builds a MsgConvertERC20 (ERC20 -> cosmos coin).
func NewMsgConvertERC20(amount sdkmath.Int, receiver string, contract, sender common.Address) *erc20types.MsgConvertERC20 {
	return &erc20types.MsgConvertERC20{
		ContractAddress: contract.Hex(),
		Amount:          amount,
		Receiver:        receiver,
		Sender:          sender.Hex(),
	}
}

// NewMsgRegisterERC20 builds a MsgRegisterERC20 (governance-gated).
func NewMsgRegisterERC20(signer string, contracts []common.Address) *erc20types.MsgRegisterERC20 {
	addrs := make([]string, len(contracts))
	for i, a := range contracts {
		addrs[i] = a.Hex()
	}
	return &erc20types.MsgRegisterERC20{
		Signer:         signer,
		Erc20Addresses: addrs,
	}
}

// NewMsgToggleConversion builds a MsgToggleConversion (governance-gated).
func NewMsgToggleConversion(authority, token string) *erc20types.MsgToggleConversion {
	return &erc20types.MsgToggleConversion{
		Authority: authority,
		Token:     token,
	}
}

// --- transaction helpers ---

// ConvertCoinToERC20 wraps cosmos coins as ERC20 tokens. Receiver is the EVM
// (0x) address that gets the wrapped ERC20. Sender is the client's signer.
func (c *Client) ConvertCoinToERC20(
	ctx context.Context,
	coin sdk.Coin,
	receiver common.Address,
	memo string,
) (*ConversionResult, error) {
	if c == nil || c.Client == nil {
		return nil, fmt.Errorf("client not initialized")
	}
	sender, err := sdkcrypto.AddressFromKey(c.Keyring(), c.KeyName(), c.Cfg().AccountHRP)
	if err != nil {
		return nil, fmt.Errorf("derive sender address: %w", err)
	}

	msg := NewMsgConvertCoin(coin, receiver, sender)
	return c.broadcastConversion(ctx, msg, memo, sender, receiver.Hex(), coin.Amount)
}

// ConvertERC20ToCoin unwraps ERC20 tokens to native cosmos coins. Receiver is
// the cosmos bech32 address; sender is the 0x address derived from the
// client's eth_secp256k1 signing key.
func (c *Client) ConvertERC20ToCoin(
	ctx context.Context,
	contract common.Address,
	amount sdkmath.Int,
	receiver string,
	memo string,
) (*ConversionResult, error) {
	if c == nil || c.Client == nil {
		return nil, fmt.Errorf("client not initialized")
	}
	senderHex, err := sdkcrypto.EVMAddressFromKey(c.Keyring(), c.KeyName())
	if err != nil {
		return nil, fmt.Errorf("derive sender 0x address: %w", err)
	}
	sender := common.HexToAddress(senderHex)

	msg := NewMsgConvertERC20(amount, receiver, contract, sender)
	return c.broadcastConversion(ctx, msg, memo, senderHex, receiver, amount)
}

// RegisterERC20Tx wraps MsgRegisterERC20 and broadcasts it. The chain may
// permit permissionless registration; if it requires governance, the signer
// must be the module authority.
func (c *Client) RegisterERC20Tx(
	ctx context.Context,
	contracts []common.Address,
	memo string,
) (string, error) {
	signer, err := sdkcrypto.AddressFromKey(c.Keyring(), c.KeyName(), c.Cfg().AccountHRP)
	if err != nil {
		return "", fmt.Errorf("derive signer: %w", err)
	}
	msg := NewMsgRegisterERC20(signer, contracts)
	txBytes, err := c.BuildAndSignTx(ctx, msg, memo)
	if err != nil {
		return "", fmt.Errorf("build and sign tx: %w", err)
	}
	txHash, _, err := c.BroadcastAndWait(ctx, txBytes, txtypes.BroadcastMode_BROADCAST_MODE_SYNC)
	return txHash, err
}

// ToggleConversionTx wraps MsgToggleConversion (governance-gated).
func (c *Client) ToggleConversionTx(
	ctx context.Context,
	authority, token, memo string,
) (string, error) {
	msg := NewMsgToggleConversion(authority, token)
	txBytes, err := c.BuildAndSignTx(ctx, msg, memo)
	if err != nil {
		return "", fmt.Errorf("build and sign tx: %w", err)
	}
	txHash, _, err := c.BroadcastAndWait(ctx, txBytes, txtypes.BroadcastMode_BROADCAST_MODE_SYNC)
	return txHash, err
}

func (c *Client) broadcastConversion(
	ctx context.Context,
	msg sdk.Msg,
	memo string,
	from, to string,
	amount sdkmath.Int,
) (*ConversionResult, error) {
	txBytes, err := c.BuildAndSignTx(ctx, msg, memo)
	if err != nil {
		return nil, fmt.Errorf("build and sign tx: %w", err)
	}
	txHash, resp, err := c.BroadcastAndWait(ctx, txBytes, txtypes.BroadcastMode_BROADCAST_MODE_SYNC)
	if err != nil {
		return nil, fmt.Errorf("broadcast and wait: %w", err)
	}
	res := &ConversionResult{
		From:   from,
		To:     to,
		Amount: amount,
		TxHash: txHash,
	}
	if resp != nil && resp.TxResponse != nil {
		res.Height = resp.TxResponse.Height
	}
	return res, nil
}
