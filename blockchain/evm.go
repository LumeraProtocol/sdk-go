package blockchain

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	txtypes "cosmossdk.io/api/cosmos/tx/v1beta1"
	sdkcrypto "github.com/LumeraProtocol/sdk-go/pkg/crypto"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// EVMClient provides x/vm module query operations plus high-level Ethereum
// transaction helpers (SendEthereumTransaction, DeployContract, CallContract,
// RawEthereumTx).
type EVMClient struct {
	query  evmtypes.QueryClient
	client *Client // backref for tx helpers (nil for query-only tests)

	// Lumera precompile wrappers. Each routes through this EVMClient.
	Action    *PrecompileClient
	Supernode *PrecompileClient
	Wasm      *PrecompileClient
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
	if resp == nil || resp.BaseFee == nil || resp.BaseFee.IsNil() {
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
	if resp == nil || resp.MinGasPrice.IsNil() {
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

// -------- Transaction Helpers --------

// EthereumTxOptions overrides defaults pulled from chain state.
type EthereumTxOptions struct {
	Nonce      *uint64  // default: query EthAccount
	GasLimit   uint64   // default: EstimateGas + 20% buffer
	GasTipCap  *big.Int // default: feemarket MinGasPrice scaled to alume/gas
	GasFeeCap  *big.Int // default: BaseFee * 2 + tipCap (alume/gas)
	Value      *big.Int // default: 0 (alume)
	AccessList ethtypes.AccessList
	Memo       string // unused for EVM-path txs; reserved for future
}

// EthereumTransactionResult captures the outcome of an Ethereum-format tx.
type EthereumTransactionResult struct {
	EthTxHash  common.Hash
	CosmosHash string
	Height     int64
	GasUsed    uint64
	VMError    string
	ReturnData []byte
	Logs       []*evmtypes.Log
}

// SendEthereumTransaction signs and broadcasts an Ethereum-format tx using
// the client's key. Pulls nonce, gas, and fees from chain when not set.
// Waits for inclusion. Returns both the eth tx hash and cosmos hash so
// callers can correlate.
//
// to=nil deploys a contract; pass DeployContract for that case.
func (c *EVMClient) SendEthereumTransaction(
	ctx context.Context,
	to *common.Address,
	data []byte,
	opts *EthereumTxOptions,
) (*EthereumTransactionResult, error) {
	if c.client == nil {
		return nil, fmt.Errorf("EVMClient not wired to a base.Client (constructed standalone)")
	}
	cfg := c.client.Cfg()
	if cfg.EVMChainID == nil || cfg.EVMChainID.Sign() <= 0 {
		return nil, fmt.Errorf("EVMChainID is required to send Ethereum transactions")
	}
	keyName := c.client.KeyName()
	if keyName == "" {
		return nil, fmt.Errorf("base client has no KeyName")
	}
	if opts == nil {
		opts = &EthereumTxOptions{}
	}

	from, err := sdkcrypto.EVMAddressFromKey(c.client.Keyring(), keyName)
	if err != nil {
		return nil, fmt.Errorf("derive sender address: %w", err)
	}
	sender := common.HexToAddress(from)

	value := opts.Value
	if value == nil {
		value = new(big.Int)
	}
	if value.Sign() < 0 {
		return nil, fmt.Errorf("value must be non-negative")
	}

	nonce, err := c.resolveNonce(ctx, sender, opts.Nonce)
	if err != nil {
		return nil, fmt.Errorf("resolve nonce: %w", err)
	}

	gasLimit, err := c.resolveGasLimit(ctx, sender, to, data, value, opts.GasLimit)
	if err != nil {
		return nil, fmt.Errorf("resolve gas limit: %w", err)
	}

	tipCap, feeCap, err := c.resolveGasCaps(ctx, opts.GasTipCap, opts.GasFeeCap)
	if err != nil {
		return nil, fmt.Errorf("resolve gas caps: %w", err)
	}

	tx := ethtypes.NewTx(&ethtypes.DynamicFeeTx{
		ChainID:    cfg.EVMChainID,
		Nonce:      nonce,
		GasTipCap:  tipCap,
		GasFeeCap:  feeCap,
		Gas:        gasLimit,
		To:         to,
		Value:      value,
		Data:       data,
		AccessList: opts.AccessList,
	})

	signed, err := sdkcrypto.SignEthereumTx(c.client.Keyring(), keyName, cfg.EVMChainID, tx)
	if err != nil {
		return nil, fmt.Errorf("sign tx: %w", err)
	}

	return c.RawEthereumTx(ctx, signed)
}

// DeployContract is sugar over SendEthereumTransaction for contract creation.
// Returns the deployed contract address derived from sender+nonce per EIP-161.
func (c *EVMClient) DeployContract(
	ctx context.Context,
	bytecode []byte,
	opts *EthereumTxOptions,
) (common.Address, *EthereumTransactionResult, error) {
	if c.client == nil {
		return common.Address{}, nil, fmt.Errorf("EVMClient not wired to a base.Client (constructed standalone)")
	}
	if opts == nil {
		opts = &EthereumTxOptions{}
	} else {
		copied := *opts
		opts = &copied
	}

	from, err := sdkcrypto.EVMAddressFromKey(c.client.Keyring(), c.client.KeyName())
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("derive sender address: %w", err)
	}
	sender := common.HexToAddress(from)
	nonce, err := c.resolveNonce(ctx, sender, opts.Nonce)
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("resolve nonce: %w", err)
	}

	opts.Nonce = &nonce
	addr := ethCreateAddress(sender, nonce)
	res, err := c.SendEthereumTransaction(ctx, nil, bytecode, opts)
	if err != nil {
		return common.Address{}, nil, err
	}
	return addr, res, nil
}

// CallContract is a read-only call (forwards to EthCall).
func (c *EVMClient) CallContract(
	ctx context.Context,
	to common.Address,
	data []byte,
) ([]byte, error) {
	var from common.Address
	if c.client != nil && c.client.KeyName() != "" {
		hex, err := sdkcrypto.EVMAddressFromKey(c.client.Keyring(), c.client.KeyName())
		if err == nil {
			from = common.HexToAddress(hex)
		}
	}
	resp, err := c.EthCall(ctx, from, &to, data, 0)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	if resp.VmError != "" {
		return resp.Ret, fmt.Errorf("vm error: %s", resp.VmError)
	}
	return resp.Ret, nil
}

// RawEthereumTx broadcasts a pre-signed go-ethereum transaction unchanged.
// For callers bringing their own signer (e.g. hardware wallet).
func (c *EVMClient) RawEthereumTx(
	ctx context.Context,
	signed *ethtypes.Transaction,
) (*EthereumTransactionResult, error) {
	if c.client == nil {
		return nil, fmt.Errorf("EVMClient not wired to a base.Client")
	}
	cfg := c.client.Cfg()
	if cfg.EVMChainID == nil {
		return nil, fmt.Errorf("EVMChainID required to encode MsgEthereumTx")
	}

	nativeDenom, extendedDenom, err := c.resolveEVMDenoms(ctx, cfg)
	if err != nil {
		return nil, err
	}

	txBytes, err := buildEthereumTxBytes(signed, nativeDenom, extendedDenom, cfg.FeeDenom)
	if err != nil {
		return nil, err
	}

	cosmosHash, getResp, err := c.client.BroadcastAndWait(ctx, txBytes, txtypes.BroadcastMode_BROADCAST_MODE_SYNC)
	if err != nil {
		return nil, fmt.Errorf("broadcast tx: %w", err)
	}

	res := &EthereumTransactionResult{
		EthTxHash:  signed.Hash(),
		CosmosHash: cosmosHash,
	}
	if getResp != nil && getResp.TxResponse != nil {
		res.Height = getResp.TxResponse.Height
		res.GasUsed = uint64(getResp.TxResponse.GasUsed)
		if data := getResp.TxResponse.Data; len(data) > 0 {
			decoded, derr := decodeMsgEthereumTxResponses([]byte(data))
			if derr != nil {
				return nil, fmt.Errorf("decode EVM tx response: %w", derr)
			}
			if len(decoded) > 0 {
				res.ReturnData = decoded[0].Ret
				res.VMError = decoded[0].VmError
				res.Logs = decoded[0].Logs
			}
		}
	}
	return res, nil
}

func (c *EVMClient) resolveNonce(ctx context.Context, sender common.Address, override *uint64) (uint64, error) {
	if override != nil {
		return *override, nil
	}
	acct, err := c.EthAccount(ctx, sender)
	if err != nil {
		return 0, err
	}
	if acct == nil {
		return 0, nil
	}
	return acct.Nonce, nil
}

func (c *EVMClient) resolveGasLimit(ctx context.Context, from common.Address, to *common.Address, data []byte, value *big.Int, override uint64) (uint64, error) {
	if override > 0 {
		return override, nil
	}
	estimate, err := c.EstimateGas(ctx, from, to, data, value, 0)
	if err != nil {
		return 0, err
	}
	if estimate == 0 {
		return 0, fmt.Errorf("gas estimator returned 0")
	}
	// 20% buffer.
	buffer := estimate / 5
	return estimate + buffer, nil
}

func (c *EVMClient) resolveGasCaps(ctx context.Context, tipOverride, feeOverride *big.Int) (*big.Int, *big.Int, error) {
	cfg := c.client.Cfg()
	tip := tipOverride
	if tip == nil {
		tip = cfg.EVMGasTipCap
	}
	if tip == nil {
		mg, err := c.resolveMinGasTipCap(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("min gas price: %w", err)
		}
		tip = mg
	}
	fee := feeOverride
	if fee == nil {
		fee = cfg.EVMGasFeeCap
	}
	if fee == nil {
		base, err := c.BaseFee(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("base fee: %w", err)
		}
		// feeCap = base*2 + tip
		fee = new(big.Int).Add(new(big.Int).Mul(base, big.NewInt(2)), tip)
	}
	if tip.Sign() < 0 || fee.Sign() < 0 {
		return nil, nil, fmt.Errorf("gas caps must be non-negative")
	}
	if tip.Cmp(fee) > 0 {
		return nil, nil, fmt.Errorf("tip cap %s exceeds fee cap %s", tip, fee)
	}
	return tip, fee, nil
}

func (c *EVMClient) resolveMinGasTipCap(ctx context.Context) (*big.Int, error) {
	if c.client == nil || c.client.FeeMarket == nil {
		return nil, fmt.Errorf("feemarket client is required")
	}
	params, err := c.client.FeeMarket.Params(ctx)
	if err != nil {
		return nil, err
	}
	if params == nil {
		return nil, fmt.Errorf("empty feemarket params response")
	}
	return sdkcrypto.ULumeDecToWei(params.Params.MinGasPrice)
}

func (c *EVMClient) resolveEVMDenoms(ctx context.Context, cfg Config) (string, string, error) {
	nativeDenom := strings.TrimSpace(cfg.EVMNativeDenom)
	extendedDenom := strings.TrimSpace(cfg.EVMExtendedDenom)
	if nativeDenom != "" && extendedDenom != "" {
		return nativeDenom, extendedDenom, nil
	}

	if c.query == nil {
		return "", "", fmt.Errorf("EVMNativeDenom and EVMExtendedDenom are required when EVM params query client is unavailable")
	}
	resp, err := c.Params(ctx)
	if err != nil {
		return "", "", fmt.Errorf("query EVM params: %w", err)
	}
	if resp != nil {
		if nativeDenom == "" {
			nativeDenom = strings.TrimSpace(resp.Params.EvmDenom)
		}
		if extendedDenom == "" && resp.Params.ExtendedDenomOptions != nil {
			extendedDenom = strings.TrimSpace(resp.Params.ExtendedDenomOptions.ExtendedDenom)
		}
	}

	if nativeDenom == "" {
		return "", "", fmt.Errorf("EVM native denom is required")
	}
	if extendedDenom == "" {
		return "", "", fmt.Errorf("EVM extended denom is required")
	}
	return nativeDenom, extendedDenom, nil
}

// buildEthereumTxBytes wraps a signed Ethereum tx as a MsgEthereumTx, builds
// the cosmos envelope with the EVM extension option and the correct fee/gas,
// and returns the encoded bytes ready for broadcast.
func buildEthereumTxBytes(signed *ethtypes.Transaction, nativeDenom, extendedDenom, feeDenom string) ([]byte, error) {
	msg, err := sdkcrypto.WrapAsMsgEthereumTx(signed)
	if err != nil {
		return nil, fmt.Errorf("wrap MsgEthereumTx: %w", err)
	}

	if nativeDenom == "" {
		nativeDenom = feeDenom
	}
	if extendedDenom == "" {
		return nil, fmt.Errorf("EVM extended denom is required")
	}

	txCfg := sdkcrypto.NewDefaultTxConfig()
	builder := txCfg.NewTxBuilder()
	if _, err := msg.BuildTxWithEvmParams(builder, evmtypes.Params{
		EvmDenom: nativeDenom,
		ExtendedDenomOptions: &evmtypes.ExtendedDenomOptions{
			ExtendedDenom: extendedDenom,
		},
	}); err != nil {
		return nil, fmt.Errorf("build cosmos envelope: %w", err)
	}

	txBytes, err := txCfg.TxEncoder()(builder.GetTx())
	if err != nil {
		return nil, fmt.Errorf("encode tx: %w", err)
	}
	return txBytes, nil
}

// decodeMsgEthereumTxResponses extracts MsgEthereumTxResponse entries from a
// cosmos TxResponse.Data field. The data is a hex-encoded TxMsgData proto.
func decodeMsgEthereumTxResponses(data []byte) ([]*evmtypes.MsgEthereumTxResponse, error) {
	encoded := strings.TrimPrefix(string(data), "0x")
	raw, err := hex.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	return evmtypes.DecodeTxResponses(raw)
}

// ethCreateAddress computes the deployed contract address for sender+nonce
// using the standard CREATE rule: keccak256(rlp([sender, nonce]))[12:].
func ethCreateAddress(sender common.Address, nonce uint64) common.Address {
	return ethcrypto.CreateAddress(sender, nonce)
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
		if value.Sign() < 0 {
			return nil, fmt.Errorf("value must be non-negative")
		}
		args.Value = (*hexutil.Big)(value)
	}
	return json.Marshal(args)
}
