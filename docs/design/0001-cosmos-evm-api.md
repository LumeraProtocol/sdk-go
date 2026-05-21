# 0001 – Cosmos EVM API Surface

**Status:** Draft
**Scope:** `sdk-go` exposure of `github.com/cosmos/evm` modules (`x/vm`, `x/erc20`, `x/feemarket`, `x/precisebank`), Ethereum-format signing, and Lumera's custom precompiles (action `0x0901`, supernode `0x0902`, wasm `0x0903`).
**Non-goals:** JSON-RPC server compatibility, contract source compilation, on-chain governance proposals beyond `MsgUpdateParams`.

## Lumera-specific context

Anchoring decisions in [../../../lumera/docs/evm-integration](../../../lumera/docs/evm-integration):

- **Single shared nonce/sequence.** EVM nonce is backed by the Cosmos `auth` account sequence. A Cosmos tx and an EVM tx from the same key advance the same counter — see [key-type-address.md](../../../lumera/docs/evm-integration/architecture/key-type-address.md). Nonce caching must invalidate on any cosmos-side tx too.
- **Dual encoding for `eth_secp256k1` keys.** The same 20-byte address can be rendered as `0x…` or `lumera1…`. `EVMToBech32` / `Bech32ToEVM` are lossless for EVM keys but **must reject** legacy `secp256k1` cosmos addresses, where derivation is `RIPEMD160(SHA256(pk))` and cannot round-trip.
- **6 ↔ 18 decimal bridging via `x/precisebank`.** Cosmos side uses `ulume` (6-dec); EVM side uses `alume` (18-dec) with conversion `1 ulume = 10^12 alume`. Default `EVMDenom = "alume"` when `EVMChainID` is set. `EthereumTxOptions.Value` is in `alume` (wei-like).
- **Lumera fee-market defaults.** BaseFee `0.0025 ulume/gas`, MinGasPrice `0.0005 ulume/gas`, change-denominator 16 (gentle ~6.25%/block). `GasTipCap` default should pull from `feemarket.Params.MinGasPrice`, not `EthCall`.
- **Custom precompiles** at `0x0901` (action), `0x0902` (supernode), `0x0903` (wasm). The SDK ships Go-side helpers so consumers do not need to compile the Solidity ABI themselves.

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
    // transactions. Distinct from the Cosmos ChainID. Nil means "not an
    // EVM-enabled chain" and disables EVM-tx helpers.
    EVMChainID *big.Int

    // EVMDenom is the 18-decimal "extended" denom used by the EVM
    // (precisebank). Defaults to "alume" when EVMChainID is set; falls back
    // to FeeDenom otherwise. Used by SendEthereumTransaction to construct
    // fees and to interpret EthereumTxOptions.Value.
    EVMDenom string

    // EVMValueUnit defines whether EthereumTxOptions.Value and the
    // EthereumTransactionResult amounts are expressed in the 18-decimal
    // extended denom (alume / "wei-like", default) or the 6-decimal base
    // denom (ulume). Helpers Wei() and ULume() in pkg/crypto handle the
    // 10^12 scaling.
    EVMValueUnit EVMValueUnit // EVMValueUnitWei | EVMValueUnitBase

    // EVMGasTipCap / EVMGasFeeCap are optional defaults (in wei) for EIP-1559
    // gas pricing. Nil means "fetch from chain state".
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

`client/config.Config` mirrors these fields, and `client.New` copies them into
`blockchain.Config` when it constructs the blockchain client.

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

`pkg/crypto/keyring.go` — `MsgEthereumTx` interfaces are already registered. Add `github.com/cosmos/evm/ethereum/eip712.RegisterInterfaces` to `NewDefaultTxConfig` so EIP-712/Web3 extension options round-trip through the cosmos tx config. No new sign-mode handler is needed for v1; EIP-712 signs a typed-data hash and stores the resulting signature in a normal `SIGN_MODE_DIRECT` `SignatureV2`.

## Address translation (`pkg/crypto/ethaddr.go`)

```go
// EVMToBech32 converts a 20-byte EVM address to the bech32 form used by
// Cosmos bank/auth for the same account.
func EVMToBech32(addr common.Address, hrp string) (string, error)

// Bech32ToEVM converts a bech32 account address to a 20-byte EVM address.
// Returns error if the bech32 decodes to a length other than 20 (which
// indicates a legacy Cosmos secp256k1 account whose derivation is not
// reversible to a 0x address — callers must look up the migration record
// via EVMigrationClient.MigrationRecord in that case).
func Bech32ToEVM(bech32Addr string) (common.Address, error)

// EVMAddressFromKey lives in address.go; pairs with these helpers.

// Wei converts a ulume amount to its 18-decimal alume (wei-like) form
// using the 10^12 conversion factor defined by x/precisebank.
func Wei(ulume sdkmath.Int) *big.Int

// ULume converts an alume amount to ulume, truncating any sub-ulume
// fractional component. The dropped fractional remainder is returned for
// callers that want to surface precision loss.
func ULume(alume *big.Int) (ulume sdkmath.Int, fractional *big.Int)
```

These are pure conversions (no keyring access). Used internally by `ERC20Client` to translate `Sender` / `Receiver` fields between the two address spaces. The `Wei` / `ULume` helpers are required wherever `Value` crosses the 6↔18 boundary — `EthereumTxOptions.Value` is alume; `sdk.Coin` amounts are ulume.

## Ethereum signing (`pkg/crypto/ethsign.go`)

```go
// SignEthereumTx signs a go-ethereum *types.Transaction with the named
// keyring entry (must be eth_secp256k1) using the replay-protected Ethereum
// signer for chainID. Returns the signed transaction.
func SignEthereumTx(
    kr keyring.Keyring,
    keyName string,
    chainID *big.Int,
    tx *ethtypes.Transaction,
) (*ethtypes.Transaction, error)

// WrapAsMsgEthereumTx packages a signed go-ethereum tx as a cosmos
// MsgEthereumTx ready for MsgEthereumTx.BuildTx and broadcast.
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

Implementation note: `keyring.Keyring.Sign` produces a 65-byte recoverable signature for `eth_secp256k1` (R, S, recovery ID). Pass that signature to `tx.WithSignature(signer, sig)` using the appropriate go-ethereum EIP-155/EIP-1559 signer; do not synthesize `V` manually.

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
func (c *EVMClient) GlobalMinGasPrice(ctx context.Context) (sdkmath.Int, error)

// EthCall executes a read-only EVM call. data is ABI-encoded calldata.
func (c *EVMClient) EthCall(ctx context.Context, to *common.Address, from common.Address, data []byte, gas uint64) ([]byte, error)

// EstimateGas returns the gas required for a hypothetical tx.
func (c *EVMClient) EstimateGas(ctx context.Context, to *common.Address, from common.Address, data []byte, value *big.Int) (uint64, error)

// TraceTx returns the raw JSON trace for a replayed EVM tx. The cosmos/evm
// gRPC API requires the full message plus block/predecessor context; there is
// no tx-hash-only TraceTx query.
func (c *EVMClient) TraceTx(ctx context.Context, req *evmtypes.QueryTraceTxRequest) ([]byte /*raw JSON*/, error)

// --- opinionated tx helpers ---

// EthereumTxOptions overrides defaults pulled from chain state.
type EthereumTxOptions struct {
    Nonce     *uint64    // default: query Account
    GasLimit  uint64     // default: EstimateGas + 20% buffer
    GasTipCap *big.Int   // default: chain min gas price
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
1. Resolve nonce  -> EthAccount(addr).Nonce (= auth sequence on Lumera).
                     Optional NonceTracker must invalidate after any
                     cosmos-side BuildAndSignTx from the same key.
2. Resolve gas    -> opts.GasLimit || EstimateGas() * 1.2
3. Resolve fees   -> opts.GasTipCap || feemarket.Params.MinGasPrice
                  -> opts.GasFeeCap || feemarket.BaseFee * 2 + tipCap
                     Both expressed in EVMDenom (alume). All wei-like.
4. Build *ethtypes.DynamicFeeTx (EIP-1559); set ChainID = EVMChainID.
5. SignEthereumTx -> signed go-ethereum tx (uses ethtypes.LatestSignerForChainID)
6. WrapAsMsgEthereumTx
7. BuildEthereumTx over the wrapper via MsgEthereumTx.BuildTx(builder,
   evmDenom). The cosmos tx envelope carries
   ExtensionOptionsEthereumTx so Lumera's dual-route ante handler routes
   it down the EVM path. No cosmos signature on the wrapper itself.
8. BroadcastAndWait
9. Decode MsgEthereumTxResponse from TxResponse.Data via
   evmtypes.DecodeTxResponse to populate logs + gas used
```

**Single-counter implication.** Because Lumera shares the EVM nonce with the auth sequence, mixing `SendEthereumTransaction` and `BuildAndSignTx` for the same key requires coordination. The simplest contract: never cache nonces by default; opt-in via `client.WithNonceCache()` only for single-flight EVM-only workloads.

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
func NewMsgRegisterERC20(signer string, addrs []string) *erc20types.MsgRegisterERC20
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

// RegisterERC20Tx wraps MsgRegisterERC20 and broadcasts it. The chain may
// allow permissionless registration; otherwise the signer must be the module
// authority.
func (c *Client) RegisterERC20Tx(
    ctx context.Context,
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

## Lumera precompiles (`pkg/evm/precompiles`)

Lumera ships three custom precompiles at fixed addresses (per [precompiles.md](../../../lumera/docs/evm-integration/precompiles/precompiles.md)). They are normal contracts from a caller's perspective, but the SDK can save consumers from compiling the Solidity ABI by hand.

New subpackage `pkg/evm/precompiles` exports:

```go
// Well-known addresses.
var (
    ActionPrecompile    = common.HexToAddress("0x0000000000000000000000000000000000000901")
    SupernodePrecompile = common.HexToAddress("0x0000000000000000000000000000000000000902")
    WasmPrecompile      = common.HexToAddress("0x0000000000000000000000000000000000000903")
)

// Pre-compiled ABIs, loaded once from go:embed'd JSON.
var (
    ActionABI    abi.ABI
    SupernodeABI abi.ABI
    WasmABI      abi.ABI
)

// ActionPrecompile helpers: typed wrappers that pack calldata and either
// EthCall or SendEthereumTransaction depending on read/write.
type ActionPrecompileClient struct{ evm *EVMClient }

func (a *ActionPrecompileClient) RequestCascade(ctx context.Context, args RequestCascadeArgs, opts *EthereumTxOptions) (*EthereumTransactionResult, string /*actionID*/, error)
func (a *ActionPrecompileClient) FinalizeCascade(ctx context.Context, args FinalizeCascadeArgs, opts *EthereumTxOptions) (*EthereumTransactionResult, error)
func (a *ActionPrecompileClient) RequestSense(ctx context.Context, args RequestSenseArgs, opts *EthereumTxOptions) (*EthereumTransactionResult, string, error)
func (a *ActionPrecompileClient) FinalizeSense(ctx context.Context, args FinalizeSenseArgs, opts *EthereumTxOptions) (*EthereumTransactionResult, error)
func (a *ActionPrecompileClient) ApproveAction(ctx context.Context, actionID string, opts *EthereumTxOptions) (*EthereumTransactionResult, error)
func (a *ActionPrecompileClient) GetAction(ctx context.Context, actionID string) (*types.Action, error)

// SupernodePrecompileClient mirrors the supernode precompile surface.
type SupernodePrecompileClient struct{ evm *EVMClient }

func (s *SupernodePrecompileClient) RegisterSupernode(ctx context.Context, args RegisterSupernodeArgs, opts *EthereumTxOptions) (*EthereumTransactionResult, error)
// ... DeregisterSupernode, StartSupernode, StopSupernode, UpdateSupernode,
//     ReportMetrics, GetSuperNode, GetSuperNodeByAccount, ListSuperNodes,
//     GetTopSuperNodesForBlock, GetMetrics, GetParams

// WasmPrecompileClient.
type WasmPrecompileClient struct{ evm *EVMClient }

func (w *WasmPrecompileClient) Execute(ctx context.Context, contract sdk.AccAddress, msg []byte, funds sdk.Coins, opts *EthereumTxOptions) (*EthereumTransactionResult, error)
func (w *WasmPrecompileClient) Query(ctx context.Context, contract sdk.AccAddress, msg []byte) ([]byte, error)
func (w *WasmPrecompileClient) ContractInfo(ctx context.Context, contract sdk.AccAddress) (*WasmContractInfo, error)
func (w *WasmPrecompileClient) RawQuery(ctx context.Context, contract sdk.AccAddress, key []byte) ([]byte, error)
```

Exposed off the EVM client:

```go
type EVMClient struct {
    // ... query, client backref

    Action    *ActionPrecompileClient
    Supernode *SupernodePrecompileClient
    Wasm      *WasmPrecompileClient
}
```

**Why both `blockchain.Action` (msg-based) and `EVM.Action` (precompile-based) exist.** Same on-chain effect, different signing/fee paths. Consumers using EVM keys / EIP-1559 fees / dApp flows should use the precompile path; cosmos-native consumers continue with `RequestActionTx`. Document this overlap explicitly so callers do not duplicate effort.

**Standard Cosmos EVM precompiles** (bank, staking, distribution, gov, slashing, ICS20, bech32, p256) are out of scope for typed Go wrappers in v1 — they are primarily a Solidity-facing surface. We do export their addresses as named constants for convenience:

```go
var (
    BankPrecompile         = common.HexToAddress("0x0000000000000000000000000000000000000804")
    StakingPrecompile      = common.HexToAddress("0x0000000000000000000000000000000000000800")
    DistributionPrecompile = common.HexToAddress("0x0000000000000000000000000000000000000801")
    ICS20Precompile        = common.HexToAddress("0x0000000000000000000000000000000000000802")
    GovPrecompile          = common.HexToAddress("0x0000000000000000000000000000000000000805")
    SlashingPrecompile     = common.HexToAddress("0x0000000000000000000000000000000000000806")
    Bech32Precompile       = common.HexToAddress("0x0000000000000000000000000000000000000400")
    P256Precompile         = common.HexToAddress("0x0000000000000000000000000000000000000100")
)
```

## `blockchain.FeeMarketClient`

Read-only, no tx helpers needed (params change only via governance):

```go
type FeeMarketClient struct {
    query feemarkettypes.QueryClient
}

func (c *FeeMarketClient) Params(ctx context.Context) (*feemarkettypes.QueryParamsResponse, error)
func (c *FeeMarketClient) BaseFee(ctx context.Context) (sdkmath.LegacyDec, error)
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
func (c *PreciseBankClient) FractionalBalance(ctx context.Context, addr sdk.AccAddress) (sdk.Coin, error)
```

## Tx pipeline interactions

`MsgEthereumTx` does **not** go through the cosmos signature placeholder dance in [blockchain/base/tx.go:149-169](../../blockchain/base/tx.go#L149). The flow is:

1. `BuildAndSignTxWithOptions` is bypassed for EVM txs.
2. A dedicated `BuildEthereumTx` (private, in `evm.go`) builds the cosmos `tx.Tx` envelope and calls `MsgEthereumTx.BuildTx(builder, evmDenom)` to apply gas/fee from the inner Ethereum tx. The tx is encoded without Cosmos signatures; the Ethereum signature lives in `Raw`.
3. Broadcast uses the existing `BroadcastAndWait`.

This is why `EVMClient` keeps a `client *Client` backref — it needs the
base client's gRPC connection for broadcast plus read-only access to the base
config/keyring/key name for `EVMChainID` and Ethereum signing. Add small
accessors on `base.Client` for those fields rather than reaching across package
boundaries.

## Testing strategy

Pattern from [blockchain/evmigration_test.go](../../blockchain/evmigration_test.go) extends cleanly:

- Each `<Module>Client` gets a bufconn-backed `fake<Module>Server` implementing the `UnimplementedQueryServer` (and `MsgServer` where applicable).
- `SendEthereumTransaction` integration test: register fakes for `vm.QueryServer` (Account, BaseFee, EstimateGas) and `tx.ServiceServer` (BroadcastTx, GetTx). Verify that the broadcast `MsgEthereumTx` has nonce/gas pulled from the fakes and that `MsgEthereumTxResponse` is decoded from `TxResponse.Data`.
- Round-trip test for `SignEthereumTx` -> `WrapAsMsgEthereumTx` -> decode -> `MsgEthereumTx.GetSender()` matches `EVMAddressFromKey`.

No tests for read-only ABI helpers beyond a mock `EthCall` returning crafted bytes.

## Open questions

1. **EIP-712 in v1?** The signature stub locks the shape but the implementation is non-trivial (requires `MakeMessages` + typed-data hashing per msg type). Defer to v2 unless a consumer needs it now.
2. **Nonce caching with shared sequence.** Because Lumera shares the EVM nonce with the auth sequence, any `BuildAndSignTx` from the same key invalidates a cached nonce. Default to no cache; offer `client.WithNonceCache()` for opt-in EVM-only workloads.
3. **Multi-msg cosmos tx that contains `MsgEthereumTx` + other cosmos msgs.** Cosmos EVM rejects this on-chain. Should `BuildAndSignTxWithOptions` validate and fail fast? Yes — add a check in `validateTxBuildOptions` (route the message to its dedicated `BuildEthereumTx` path or return an error).
4. **Receipt struct.** Returning `[]*ethtypes.Log` from `EthereumTransactionResult` leaks go-ethereum types into the SDK's public API. Acceptable since we already import `common.Address`. Alternative is a wrapper `EVMLog` type — costs surface, gains independence.
5. **`x/feegrant` / `x/authz` over EVM messages.** Out of scope; cosmos/evm currently disallows.
6. **Precompile vs Msg duplication.** `Action.RequestCascade` exists both as a cosmos msg (`MsgRequestAction`) and as a precompile call at `0x0901`. Document the choice rule (EVM keys / dApp callers → precompile; cosmos keys / existing tooling → msg) and avoid silently routing one to the other.
7. **Lumera-specific defaults.** Should the SDK ship a `client.LumeraDefaults()` option group that sets `EVMChainID`, `EVMDenom = "alume"`, and registers the precompile clients? Cuts boilerplate for the common case but ties the SDK to a single chain; the alternative is per-call configuration.
8. **ABI assets.** Precompile ABIs live in the Lumera repo. Vendor them via `go:embed` of JSON files copied at SDK release time, or fetch from the lumera module dependency at build time? Vendoring is simpler but adds a sync step on each Lumera ABI bump.

## Build sequence

Phased rollout that always leaves `main` green:

1. **Phase 1 — Foundations.** `pkg/crypto/ethsign.go` + `ethaddr.go` with `SignEthereumTx`, `WrapAsMsgEthereumTx`, `EVMToBech32`, `Bech32ToEVM`, `Wei`/`ULume` decimal helpers. Round-trip unit tests. No public client surface yet.
2. **Phase 2 — Read-only clients.** `FeeMarketClient`, `PreciseBankClient`, `EVMClient` queries only (no tx helpers). Wire onto `blockchain.Client`. Bufconn tests.
3. **Phase 3 — EVM tx pipeline.** `SendEthereumTransaction`, `DeployContract`, `CallContract`, `RawEthereumTx`. Add `EVMChainID` / `EVMDenom` / `EVMValueUnit` to `Config` and `client.With...` options. Multi-msg validation in `validateTxBuildOptions`.
4. **Phase 4 — ERC20.** `ERC20Client` queries + `ConvertCoinToERC20` / `ConvertERC20ToCoin` (these reuse the regular cosmos tx pipeline — no Ethereum signing). ABI sugar.
5. **Phase 5 — Lumera precompiles.** `pkg/evm/precompiles` with embedded ABIs, plus `EVMClient.Action` / `Supernode` / `Wasm` typed wrappers. Document the precompile-vs-msg choice rule.
6. **Phase 6 — Docs + examples.** Tutorial in [docs/DEVELOPER_GUIDE.md](../DEVELOPER_GUIDE.md), one example under `examples/evm-transfer` and one under `examples/precompile-action`, API.md updates.
7. **Phase 7 (later).** EIP-712, opt-in nonce tracker, fee-market cache.

Each phase ships independently and is reviewable in ~300-500 LOC chunks.
