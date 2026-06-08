package blockchain

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// Minimal ERC20 ABI covering the methods the SDK exposes via Erc20* sugar.
// We hand-roll the JSON rather than embed a vendored file so we do not have
// to ship contract artifacts in the SDK.
const erc20MinimalABI = `[
{"constant":true,"inputs":[{"name":"owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"type":"function"},
{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"type":"function"},
{"constant":true,"inputs":[{"name":"owner","type":"address"},{"name":"spender","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"type":"function"},
{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"type":"function"},
{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"type":"function"},
{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"type":"function"}
]`

var erc20ABI abi.ABI

func init() {
	parsed, err := abi.JSON(strings.NewReader(erc20MinimalABI))
	if err != nil {
		panic(fmt.Sprintf("parse erc20 abi: %v", err))
	}
	erc20ABI = parsed
}

// Erc20Metadata bundles the static ERC20 metadata fields.
type Erc20Metadata struct {
	Name     string
	Symbol   string
	Decimals uint8
}

// Erc20Balance calls balanceOf(holder) on the ERC20 contract.
func (c *ERC20Client) Erc20Balance(ctx context.Context, contract, holder common.Address) (*big.Int, error) {
	data, err := erc20ABI.Pack("balanceOf", holder)
	if err != nil {
		return nil, fmt.Errorf("pack balanceOf: %w", err)
	}
	return c.callUint256(ctx, contract, "balanceOf", data)
}

// Erc20TotalSupply calls totalSupply() on the ERC20 contract.
func (c *ERC20Client) Erc20TotalSupply(ctx context.Context, contract common.Address) (*big.Int, error) {
	data, err := erc20ABI.Pack("totalSupply")
	if err != nil {
		return nil, fmt.Errorf("pack totalSupply: %w", err)
	}
	return c.callUint256(ctx, contract, "totalSupply", data)
}

// Erc20Allowance calls allowance(owner, spender) on the ERC20 contract.
func (c *ERC20Client) Erc20Allowance(ctx context.Context, contract, owner, spender common.Address) (*big.Int, error) {
	data, err := erc20ABI.Pack("allowance", owner, spender)
	if err != nil {
		return nil, fmt.Errorf("pack allowance: %w", err)
	}
	return c.callUint256(ctx, contract, "allowance", data)
}

// Erc20Metadata returns name, symbol, decimals for the ERC20 contract.
func (c *ERC20Client) Erc20Metadata(ctx context.Context, contract common.Address) (Erc20Metadata, error) {
	name, err := c.callString(ctx, contract, "name")
	if err != nil {
		return Erc20Metadata{}, fmt.Errorf("name: %w", err)
	}
	symbol, err := c.callString(ctx, contract, "symbol")
	if err != nil {
		return Erc20Metadata{}, fmt.Errorf("symbol: %w", err)
	}
	decimals, err := c.callUint8(ctx, contract, "decimals")
	if err != nil {
		return Erc20Metadata{}, fmt.Errorf("decimals: %w", err)
	}
	return Erc20Metadata{Name: name, Symbol: symbol, Decimals: decimals}, nil
}

func (c *ERC20Client) callContract(ctx context.Context, contract common.Address, data []byte) ([]byte, error) {
	if c.client == nil || c.client.EVM == nil {
		return nil, fmt.Errorf("ERC20Client not wired to an EVM client")
	}
	return c.client.EVM.CallContract(ctx, contract, data)
}

func (c *ERC20Client) callUint256(ctx context.Context, contract common.Address, method string, data []byte) (*big.Int, error) {
	ret, err := c.callContract(ctx, contract, data)
	if err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		return nil, fmt.Errorf("%s: contract %s returned no data (address has no code or call reverted)", method, contract)
	}
	out, err := erc20ABI.Unpack(method, ret)
	if err != nil {
		return nil, fmt.Errorf("unpack %s: %w", method, err)
	}
	bn, ok := out[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("unexpected return type for %s: %T", method, out[0])
	}
	return bn, nil
}

func (c *ERC20Client) callString(ctx context.Context, contract common.Address, method string) (string, error) {
	data, err := erc20ABI.Pack(method)
	if err != nil {
		return "", fmt.Errorf("pack %s: %w", method, err)
	}
	ret, err := c.callContract(ctx, contract, data)
	if err != nil {
		return "", err
	}
	if len(ret) == 0 {
		return "", fmt.Errorf("%s: contract %s returned no data (address has no code or call reverted)", method, contract)
	}
	out, err := erc20ABI.Unpack(method, ret)
	if err != nil {
		return "", fmt.Errorf("unpack %s: %w", method, err)
	}
	s, ok := out[0].(string)
	if !ok {
		return "", fmt.Errorf("unexpected return type for %s: %T", method, out[0])
	}
	return s, nil
}

func (c *ERC20Client) callUint8(ctx context.Context, contract common.Address, method string) (uint8, error) {
	data, err := erc20ABI.Pack(method)
	if err != nil {
		return 0, fmt.Errorf("pack %s: %w", method, err)
	}
	ret, err := c.callContract(ctx, contract, data)
	if err != nil {
		return 0, err
	}
	if len(ret) == 0 {
		return 0, fmt.Errorf("%s: contract %s returned no data (address has no code or call reverted)", method, contract)
	}
	out, err := erc20ABI.Unpack(method, ret)
	if err != nil {
		return 0, fmt.Errorf("unpack %s: %w", method, err)
	}
	v, ok := out[0].(uint8)
	if !ok {
		return 0, fmt.Errorf("unexpected return type for %s: %T", method, out[0])
	}
	return v, nil
}
