package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// EVMClient provides x/vm module query operations. Tx helpers
// (SendEthereumTransaction, DeployContract, CallContract) land in Phase 3.
type EVMClient struct {
	query evmtypes.QueryClient
}

// Code returns the EVM bytecode deployed at addr.
func (c *EVMClient) Code(ctx context.Context, addr common.Address) ([]byte, error) {
	resp, err := c.query.Code(ctx, &evmtypes.QueryCodeRequest{Address: addr.Hex()})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	return resp.Code, nil
}

// Storage returns the storage slot value at key for the contract at addr.
func (c *EVMClient) Storage(ctx context.Context, addr common.Address, key common.Hash) (common.Hash, error) {
	resp, err := c.query.Storage(ctx, &evmtypes.QueryStorageRequest{
		Address: addr.Hex(),
		Key:     key.Hex(),
	})
	if err != nil {
		return common.Hash{}, err
	}
	if resp == nil {
		return common.Hash{}, nil
	}
	return common.HexToHash(resp.Value), nil
}

// Balance returns the account balance in the EVM extended denom (alume on
// Lumera) as an integer wei-like *big.Int.
func (c *EVMClient) Balance(ctx context.Context, addr common.Address) (*big.Int, error) {
	resp, err := c.query.Balance(ctx, &evmtypes.QueryBalanceRequest{Address: addr.Hex()})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Balance == "" {
		return new(big.Int), nil
	}
	bal, ok := new(big.Int).SetString(resp.Balance, 10)
	if !ok {
		return nil, fmt.Errorf("decode balance %q", resp.Balance)
	}
	return bal, nil
}

// EthAccount returns the raw EVM account record: balance, code hash, nonce.
// On Lumera the nonce is shared with the Cosmos auth sequence.
func (c *EVMClient) EthAccount(ctx context.Context, addr common.Address) (*evmtypes.QueryAccountResponse, error) {
	return c.query.Account(ctx, &evmtypes.QueryAccountRequest{Address: addr.Hex()})
}

// CosmosAccount returns the cosmos bech32 address paired with the given EVM
// address, plus its current sequence and account number.
func (c *EVMClient) CosmosAccount(ctx context.Context, addr common.Address) (*evmtypes.QueryCosmosAccountResponse, error) {
	return c.query.CosmosAccount(ctx, &evmtypes.QueryCosmosAccountRequest{Address: addr.Hex()})
}

// Params returns the current x/vm module parameters.
func (c *EVMClient) Params(ctx context.Context) (*evmtypes.QueryParamsResponse, error) {
	return c.query.Params(ctx, &evmtypes.QueryParamsRequest{})
}

// BaseFee returns the EIP-1559 base fee in the EVM extended denom (alume) as
// a *big.Int. Differs from FeeMarketClient.BaseFee, which returns the native
// denom (ulume) decimal value.
func (c *EVMClient) BaseFee(ctx context.Context) (*big.Int, error) {
	resp, err := c.query.BaseFee(ctx, &evmtypes.QueryBaseFeeRequest{})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.BaseFee == nil {
		return new(big.Int), nil
	}
	bf := *resp.BaseFee
	return bf.BigInt(), nil
}

// Config returns the EVM configuration (chain config, denoms, active forks).
func (c *EVMClient) Config(ctx context.Context) (*evmtypes.QueryConfigResponse, error) {
	return c.query.Config(ctx, &evmtypes.QueryConfigRequest{})
}

// GlobalMinGasPrice returns the chain-wide minimum gas price in alume/gas.
func (c *EVMClient) GlobalMinGasPrice(ctx context.Context) (*big.Int, error) {
	resp, err := c.query.GlobalMinGasPrice(ctx, &evmtypes.QueryGlobalMinGasPriceRequest{})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return new(big.Int), nil
	}
	return resp.MinGasPrice.BigInt(), nil
}

// EthCall executes a read-only EVM call. from may be the zero address.
func (c *EVMClient) EthCall(ctx context.Context, from common.Address, to *common.Address, data []byte, gasCap uint64) (*evmtypes.MsgEthereumTxResponse, error) {
	args, err := encodeEthCallArgs(from, to, data, nil)
	if err != nil {
		return nil, err
	}
	if gasCap == 0 {
		gasCap = 25_000_000
	}
	return c.query.EthCall(ctx, &evmtypes.EthCallRequest{
		Args:   args,
		GasCap: gasCap,
	})
}

// EstimateGas estimates gas for a hypothetical tx. value may be nil.
func (c *EVMClient) EstimateGas(ctx context.Context, from common.Address, to *common.Address, data []byte, value *big.Int, gasCap uint64) (uint64, error) {
	args, err := encodeEthCallArgs(from, to, data, value)
	if err != nil {
		return 0, err
	}
	if gasCap == 0 {
		gasCap = 25_000_000
	}
	resp, err := c.query.EstimateGas(ctx, &evmtypes.EthCallRequest{
		Args:   args,
		GasCap: gasCap,
	})
	if err != nil {
		return 0, err
	}
	if resp == nil {
		return 0, nil
	}
	return resp.Gas, nil
}

// TraceTx returns the raw JSON trace for a replayed EVM tx. Building the
// request is non-trivial (block context, predecessors) so callers construct
// the request themselves; this helper just forwards it.
func (c *EVMClient) TraceTx(ctx context.Context, req *evmtypes.QueryTraceTxRequest) ([]byte, error) {
	resp, err := c.query.TraceTx(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	return resp.Data, nil
}

func encodeEthCallArgs(from common.Address, to *common.Address, data []byte, value *big.Int) ([]byte, error) {
	args := evmtypes.TransactionArgs{
		From: &from,
		To:   to,
	}
	if len(data) > 0 {
		input := hexutil.Bytes(data)
		args.Input = &input
	}
	if value != nil {
		args.Value = (*hexutil.Big)(value)
	}
	return json.Marshal(args)
}
