# 0001 – Cosmos EVM API Surface

**Status:** Draft
**Scope:** `sdk-go` exposure of `github.com/cosmos/evm` modules (`x/vm`, `x/erc20`, `x/feemarket`, `x/precisebank`) plus Ethereum-format signing.
**Non-goals:** JSON-RPC server compatibility, contract source compilation, on-chain governance proposals beyond `MsgUpdateParams`.

## Guiding principles

1. **Mirror existing module-client pattern.** Each module gets a `<Module>Client` exposed off `blockchain.Client`, symmetrical with `ActionClient` / `SuperNodeClient` / `EVMigrationClient`.
2. **Lean opinionated.** High-level helpers fill nonce, gas, fee, and signing automatically. Thin wrappers exist underneath for callers that need control.
3. **Single keyring.** EVM keys (`eth_secp256k1`) and Cosmos keys (`secp256k1`) coexist in the same `keyring.Keyring`; selection is per-call via `KeyName`.
4. **Two signing worlds.** Cosmos-sign-mode `MsgEthereumTx` is *not* the same as RLP-signed Ethereum transactions; the SDK helpers always produce the latter (EIP-155), wrapped in `MsgEthereumTx` for broadcast.
5. **No silent breakage.** New surface is additive. Existing `BuildAndSignTx` / module clients stay untouched.

## Configuration additions

`blockchain/base/config.go` — `Config`:

```go
type Config struct {
    // existing fields...

    // EVMChainID is the EIP-155 chain ID used to sign Ethereum-format
    // transactions. Distinct from the Cosmos ChainID. Zero means "not an
    // EVM-enabled chain" and disables EVM-tx helpers.
    EVMChainID *big.Int

    // EVMDenom is the bank denom representing native EVM gas token.
    // Defaults to FeeDenom when unset. Used by SendEthereumTransaction to
    // construct fees.
    EVMDenom string

    // EVMGasTipCap / EVMGasFeeCap are optional defaults (in wei) for EIP-1559
    // gas pricing. Zero means "fetch from feemarket".
    EVMGasTipCap *big.Int
    EVMGasFeeCap *big.Int
}
```

Client options (in `client/`):

```go
client.WithEVMChainID(id *big.Int)
client.WithEVMDenom(denom string)
client.WithEVMGasCaps(tip, fee *big.Int)
```

## Files added

```
blockchain/
  evm.go              # x/vm client + EthereumTransactionResult
  evm_test.go
  erc20.go            # x/erc20 client + tx helpers + result types
  erc20_test.go
  feemarket.go        # x/feemarket client (read-only)
  feemarket_test.go
  precisebank.go      # x/precisebank client (read-only)
  precisebank_test.go
pkg/crypto/
  ethsign.go          # SignEthereumTx + SignEIP712Tx helpers
  ethaddr.go          # Bech32 <-> 0x address translation helpers
```

`blockchain/client.go` grows four fields on `Client`:

```go
type Client struct {
    *base.Client

    Action      *ActionClient
    SuperNode   *SuperNodeClient
    Claim       *ClaimClient
    EVMigration *EVMigrationClient
    Audit       *AuditClient

    EVM         *EVMClient        // x/vm
    ERC20       *ERC20Client      // x/erc20
    FeeMarket   *FeeMarketClient  // x/feemarket
    PreciseBank *PreciseBankClient
}
```

`pkg/crypto/keyring.go` — register `MsgEthereumTx` interfaces (already done) and add EIP-712 sign mode handler registration in `NewDefaultTxConfig` so `MsgEthereumTx` round-trips through the cosmos tx config.

## Address translation (`pkg/crypto/ethaddr.go`)

```go
// EVMToBech32 converts a 20-byte EVM address to the bech32 form used by
// Cosmos bank/auth for the same account.
func EVMToBech32(addr common.Address, hrp string) (string, error)

// Bech32ToEVM converts a bech32 account address to a 20-byte EVM address.
// Returns error if the bech32 decodes to a length other than 20.
func Bech32ToEVM(bech32Addr string) (common.Address, error)

// EVMAddressFromKey lives in address.go; pairs with these helpers.
```

These are pure conversions (no keyring access). Used internally by `ERC20Client` to translate `Sender` / `Receiver` fields between the two address spaces.

## Ethereum signing (`pkg/crypto/ethsign.go`)

```go
// SignEthereumTx signs a go-ethereum *types.Transaction with the named
// keyring entry (must be eth_secp256k1) using EIP-155. Returns the signed
// transaction.
func SignEthereumTx(
    kr keyring.Keyring,
    keyName string,
    chainID *big.Int,
    tx *ethtypes.Transaction,
) (*ethtypes.Transaction, error)

// WrapAsMsgEthereumTx packages a signed go-ethereum tx as a cosmos
// MsgEthereumTx ready for broadcast via the standard tx pipeline.
func WrapAsMsgEthereumTx(signed *ethtypes.Transaction) (*evmtypes.MsgEthereumTx, error)

// SignEIP712Tx signs an arbitrary cosmos tx using EIP-712 typed-data so
// MetaMask-style wallets can authorize cosmos messages. Out of scope for v1;
// stub the signature now to lock the API.
func SignEIP712Tx(
    kr keyring.Keyring,
    keyName string,
    chainID *big.Int,
    txCfg client.TxConfig,
    builder client.TxBuilder,
    signerData authsigning.SignerData,
) error
```

Implementation note: `keyring.Keyring.Sign` produces a 64-byte signature for `eth_secp256k1`; we recover V and ABI-encode into a go-ethereum `Signature` via the existing `cosmos/evm/crypto/ethsecp256k1` package.

## `blockchain.EVMClient` (x/vm)

```go
type EVMClient struct {
    query  evmtypes.QueryClient
    client *Client // backref for tx helpers
}

// --- queries (thin wrappers) ---

func (c *EVMClient) Code(ctx context.Context, addr common.Address) ([]byte, error)
func (c *EVMClient) Storage(ctx context.Context, addr common.Address, key common.Hash) (common.Hash, error)
func (c *EVMClient) Balance(ctx context.Context, addr common.Address) (*big.Int, error)
func (c *EVMClient) EthAccount(ctx context.Context, addr common.Address) (*evmtypes.QueryAccountResponse, error)
func (c *EVMClient) CosmosAccount(ctx context.Context, addr common.Address) (string /*bech32*/, error)
func (c *EVMClient) Params(ctx context.Context) (*evmtypes.QueryParamsResponse, error)
func (c *EVMClient) BaseFee(ctx context.Context) (*big.Int, error)
func (c *EVMClient) Config(ctx context.Context) (*evmtypes.QueryConfigResponse, error)
func (c *EVMClient) GlobalMinGasPrice(ctx context.Context) (sdkmath.LegacyDec, error)

// EthCall executes a read-only EVM call. data is ABI-encoded calldata.
func (c *EVMClient) EthCall(ctx context.Context, to *common.Address, from common.Address, data []byte, gas uint64) ([]byte, error)

// EstimateGas returns the gas required for a hypothetical tx.
func (c *EVMClient) EstimateGas(ctx context.Context, to *common.Address, from common.Address, data []byte, value *big.Int) (uint64, error)

// TraceTx returns the EVM trace for an existing tx hash.
func (c *EVMClient) TraceTx(ctx context.Context, txHash common.Hash) ([]byte /*raw JSON*/, error)

// --- opinionated tx helpers ---

// EthereumTxOptions overrides defaults pulled from chain state.
type EthereumTxOptions struct {
    Nonce     *uint64    // default: query Account
    GasLimit  uint64     // default: EstimateGas + 20% buffer
    GasTipCap *big.Int   // default: feemarket params
    GasFeeCap *big.Int   // default: baseFee*2 + tipCap
    Value     *big.Int   // default: 0
    AccessList ethtypes.AccessList
    Memo      string
}

// EthereumTransactionResult mirrors ActionResult.
type EthereumTransactionResult struct {
    EthTxHash  common.Hash
    CosmosHash string
    Height     int64
    GasUsed    uint64
    Logs       []*ethtypes.Log // decoded from MsgEthereumTxResponse
}

// SendEthereumTransaction signs and broadcasts an Ethereum-format tx using
// the client's KeyName. Pulls nonce, gas, and fees from chain when not set.
// Waits for inclusion. Returns both the eth tx hash and cosmos hash so
// callers can correlate.
func (c *Client) SendEthereumTransaction(
    ctx context.Context,
    to *common.Address, // nil = contract creation
    data []byte,
    opts *EthereumTxOptions,
) (*EthereumTransactionResult, error)

// DeployContract is sugar over SendEthereumTransaction with to=nil; returns
// the deployed contract address parsed from logs.
func (c *Client) DeployContract(
    ctx context.Context,
    bytecode []byte,
    opts *EthereumTxOptions,
) (common.Address, *EthereumTransactionResult, error)

// CallContract is sugar for read-only calls (forwards to EthCall).
func (c *Client) CallContract(
    ctx context.Context,
    to common.Address,
    data []byte,
) ([]byte, error)

// RawEthereumTx broadcasts a pre-signed go-ethereum transaction unchanged.
// For callers bringing their own signer (e.g. hardware wallet).
func (c *Client) RawEthereumTx(
    ctx context.Context,
    signed *ethtypes.Transaction,
) (*EthereumTransactionResult, error)
```

Internal flow for `SendEthereumTransaction`:

```
1. Resolve nonce  -> EthAccount(addr).Nonce  (cached per-client optionally)
2. Resolve gas    -> opts.GasLimit || EstimateGas() * 1.2
3. Resolve fees   -> opts.GasTipCap || feemarket.Params.MinGasPrice
                  -> opts.GasFeeCap || BaseFee * 2 + tipCap
4. Build *ethtypes.DynamicFeeTx (EIP-1559)
5. SignEthereumTx -> signed go-ethereum tx
6. WrapAsMsgEthereumTx
7. BuildAndSignTxWithOptions over the wrapper (no extra cosmos signature on
   MsgEthereumTx itself; SetSignatures with empty SignatureV2)
8. BroadcastAndWait
9. Decode MsgEthereumTxResponse from tx.Events to populate logs + gas used
```

## `blockchain.ERC20Client` (x/erc20)

```go
type ERC20Client struct {
    query  erc20types.QueryClient
    client *Client
}

// --- queries ---

func (c *ERC20Client) TokenPairs(ctx context.Context, pagination *query.PageRequest) ([]erc20types.TokenPair, *query.PageResponse, error)
func (c *ERC20Client) TokenPair(ctx context.Context, token string /*denom or 0x addr*/) (erc20types.TokenPair, error)
func (c *ERC20Client) Params(ctx context.Context) (*erc20types.QueryParamsResponse, error)

// --- message constructors (symmetric with action.go) ---

func NewMsgConvertCoin(coin sdk.Coin, receiver common.Address, sender sdk.AccAddress) *erc20types.MsgConvertCoin
func NewMsgConvertERC20(amount sdkmath.Int, receiver sdk.AccAddress, contract, sender common.Address) *erc20types.MsgConvertERC20
func NewMsgRegisterERC20(authority string, signer string, addrs []string) *erc20types.MsgRegisterERC20
func NewMsgToggleConversion(authority string, token string) *erc20types.MsgToggleConversion

// --- result type ---

type ConversionResult struct {
    From     string  // cosmos bech32 or 0x
    To       string  // opposite address space
    Amount   sdkmath.Int
    TxHash   string
    Height   int64
}

// --- opinionated tx helpers ---

// ConvertCoinToERC20 sends a MsgConvertCoin, signed by the client's KeyName.
// Receiver is the EVM (0x) address that gets the wrapped ERC20.
func (c *Client) ConvertCoinToERC20(
    ctx context.Context,
    coin sdk.Coin,
    receiver common.Address,
    memo string,
) (*ConversionResult, error)

// ConvertERC20ToCoin sends a MsgConvertERC20. Receiver is the bech32 cosmos
// address. Sender is the 0x address that owns the ERC20 (must match the
// client's EVM-keyed signer).
func (c *Client) ConvertERC20ToCoin(
    ctx context.Context,
    contract common.Address,
    amount sdkmath.Int,
    receiver sdk.AccAddress,
    memo string,
) (*ConversionResult, error)

// RegisterERC20Tx is a governance-only path; the helper just wraps the msg
// and broadcasts. Useful for authority key flows in dev networks.
func (c *Client) RegisterERC20Tx(
    ctx context.Context,
    authority string,
    contracts []common.Address,
    memo string,
) (string /*tx hash*/, error)

// ToggleConversionTx (governance).
func (c *Client) ToggleConversionTx(
    ctx context.Context,
    authority string,
    token string,
    memo string,
) (string, error)

// --- ABI sugar (no chain calls) ---

// Erc20Balance returns the balance of holder for the ERC20 contract by
// calling balanceOf(address) via EthCall.
func (c *ERC20Client) Erc20Balance(
    ctx context.Context,
    contract, holder common.Address,
) (*big.Int, error)

// Erc20Allowance / Erc20TotalSupply / Erc20Metadata follow the same shape.
```

ABI calls (`Erc20Balance` etc.) ship a pre-compiled `abi.ABI` for the standard ERC20 interface as a package-level singleton so callers do not pull `go-ethereum/accounts/abi` themselves.

## `blockchain.FeeMarketClient`

Read-only, no tx helpers needed (params change only via governance):

```go
type FeeMarketClient struct {
    query feemarkettypes.QueryClient
}

func (c *FeeMarketClient) Params(ctx context.Context) (*feemarkettypes.QueryParamsResponse, error)
func (c *FeeMarketClient) BaseFee(ctx context.Context) (*big.Int, error)
func (c *FeeMarketClient) BlockGas(ctx context.Context) (*feemarkettypes.QueryBlockGasResponse, error)
```

Cached `BaseFee` lookup (single-flight, ~1 block TTL) is a follow-up optimization for high-throughput callers.

## `blockchain.PreciseBankClient`

Read-only:

```go
type PreciseBankClient struct {
    query precisebanktypes.QueryClient
}

func (c *PreciseBankClient) Remainder(ctx context.Context) (sdk.Coin, error)
func (c *PreciseBankClient) FractionalBalance(ctx context.Context, addr sdk.AccAddress) (sdkmath.Int, error)
```

## Tx pipeline interactions

`MsgEthereumTx` does **not** go through the cosmos signature placeholder dance in [blockchain/base/tx.go:149-169](../../blockchain/base/tx.go#L149). The flow is:

1. `BuildAndSignTxWithOptions` is bypassed for EVM txs.
2. A dedicated `BuildEthereumTx` (private, in `evm.go`) builds the cosmos `tx.Tx` envelope, calls `MsgEthereumTx.BuildTx(builder, evmDenom)` to apply gas/fee from the inner Ethereum tx, and sets an *empty* `SignatureV2` slice (signature lives in `Raw`).
3. Broadcast uses the existing `BroadcastAndWait`.

This is why `EVMClient` keeps a `client *Client` backref — it needs `base.Client.conn` for broadcast and `base.Client.config` for `EVMChainID`.

## Testing strategy

Pattern from [blockchain/evmigration_test.go](../../blockchain/evmigration_test.go) extends cleanly:

- Each `<Module>Client` gets a bufconn-backed `fake<Module>Server` implementing the `UnimplementedQueryServer` (and `MsgServer` where applicable).
- `SendEthereumTransaction` integration test: register fakes for `vm.QueryServer` (Account, BaseFee, EstimateGas), `tx.ServiceServer` (Simulate, BroadcastTx, GetTx), and `auth.QueryServer`. Verify that the broadcast `MsgEthereumTx` has nonce/gas pulled from the fakes.
- Round-trip test for `SignEthereumTx` -> `WrapAsMsgEthereumTx` -> decode -> `MsgEthereumTx.GetSender()` matches `EVMAddressFromKey`.

No tests for read-only ABI helpers beyond a mock `EthCall` returning crafted bytes.

## Open questions

1. **EIP-712 in v1?** The signature stub locks the shape but the implementation is non-trivial (requires `MakeMessages` + typed-data hashing per msg type). Defer to v2 unless a consumer needs it now.
2. **Nonce caching.** A naive per-call `EthAccount` query doubles the chain RTT for every tx. Worth adding `Client.NonceTracker` (single signer, monotonic) but only behind an opt-in option to avoid masking external nonce changes.
3. **Multi-msg cosmos tx that contains `MsgEthereumTx` + other cosmos msgs.** Cosmos EVM rejects this on-chain. Should `BuildAndSignTxWithOptions` validate and fail fast? Probably yes — small lint check in `validateTxBuildOptions`.
4. **Receipt struct.** Returning `[]*ethtypes.Log` from `EthereumTransactionResult` leaks go-ethereum types into the SDK's public API. Acceptable since `pkg/crypto` already imports `common.Address`. Alternative is a wrapper `EVMLog` type — costs surface, gains independence.
5. **`x/feegrant` / `x/authz` over EVM messages.** Out of scope; cosmos/evm currently disallows.

## Build sequence

Phased rollout that always leaves `main` green:

1. **Phase 1 — Foundations.** `pkg/crypto/ethsign.go` + `ethaddr.go` with `SignEthereumTx`, `WrapAsMsgEthereumTx`, `EVMToBech32`, `Bech32ToEVM`. Round-trip unit tests. No public client surface yet.
2. **Phase 2 — Read-only clients.** `FeeMarketClient`, `PreciseBankClient`, `EVMClient` queries only (no tx helpers). Wire onto `blockchain.Client`. Bufconn tests.
3. **Phase 3 — EVM tx pipeline.** `SendEthereumTransaction`, `DeployContract`, `CallContract`, `RawEthereumTx`. Add `EVMChainID` / `EVMDenom` to `Config` and `client.With...` options.
4. **Phase 4 — ERC20.** `ERC20Client` queries + `ConvertCoinToERC20` / `ConvertERC20ToCoin` (these reuse the regular cosmos tx pipeline — no Ethereum signing). ABI sugar.
5. **Phase 5 — Docs + examples.** Tutorial in [docs/DEVELOPER_GUIDE.md](../DEVELOPER_GUIDE.md), one example under `examples/evm-transfer`, API.md updates.
6. **Phase 6 (later).** EIP-712, nonce tracker, fee-market cache.

Each phase ships independently and is reviewable in ~300-500 LOC chunks.
