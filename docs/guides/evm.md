# EVM Integration

Lumera's Cosmos EVM stack exposes four modules — `x/vm`, `x/erc20`, `x/feemarket`, `x/precisebank` — plus three custom precompiles (action `0x0901`, supernode `0x0902`, wasm `0x0903`). The SDK wraps each behind an opinionated client and shares Ethereum-format signing via `pkg/crypto`.

## Configure

EVM helpers require an EIP-155 chain ID and the precisebank denoms. Pass them as options to `client.New`:

```go
lumera, err := client.New(ctx, cfg, kr,
    client.WithKeyName("alice"),                // eth_secp256k1 key
    client.WithEVMChainID(big.NewInt(1414)),    // EIP-155 chain ID
    client.WithEVMNativeDenom("ulume"),         // x/vm evm_denom (6 dec)
    client.WithEVMExtendedDenom("alume"),       // precisebank extended (18 dec)
)
```

When `EVMNativeDenom` / `EVMExtendedDenom` are left empty, the SDK queries `x/vm` params at the first EVM call and caches per-tx.

## Key concepts

- **Single shared nonce.** Lumera's EVM nonce is the cosmos `auth` sequence. Mixing `SendEthereumTransaction` and `BuildAndSignTx` for the same key works without coordination — both advance the same counter. The SDK does not cache nonces by default.
- **Dual address encoding.** `eth_secp256k1` accounts can be rendered as `0x…` *or* `lumera1…`. `pkg/crypto.Bech32ToEVM` and `EVMToBech32` are pure byte conversions; legacy `secp256k1` cosmos addresses are not reversible to a meaningful 0x. See [crypto.md](crypto.md).
- **6 ↔ 18 decimal bridging.** Cosmos side uses `ulume` (6 dec); EVM side uses `alume` (18 dec) with `1 ulume = 10^12 alume`. Use `sdkcrypto.Wei(ulumeAmt)` for tx `Value`, `sdkcrypto.ULume(alumeAmt)` for the inverse, and `sdkcrypto.ULumeDecToWei(dec)` for feemarket params.
- **Cosmos pipeline rejects `MsgEthereumTx`.** `BuildAndSignTxWithOptions` fails fast if you slip a `MsgEthereumTx` into the cosmos signing path. Route EVM txs through `EVMClient.SendEthereumTransaction`.

## Read-only queries

`Blockchain.EVM` (`x/vm`):

```go
code, _ := lumera.Blockchain.EVM.Code(ctx, addr)
slot, _ := lumera.Blockchain.EVM.Storage(ctx, addr, key)
bal, _ := lumera.Blockchain.EVM.Balance(ctx, addr)              // alume
acct, _ := lumera.Blockchain.EVM.EthAccount(ctx, addr)          // nonce, code hash
cosmos, _ := lumera.Blockchain.EVM.CosmosAccount(ctx, addr)     // paired bech32 + sequence
params, _ := lumera.Blockchain.EVM.Params(ctx)
baseFee, _ := lumera.Blockchain.EVM.BaseFee(ctx)                // alume integer
minGas, _ := lumera.Blockchain.EVM.GlobalMinGasPrice(ctx)       // alume integer
ret, _ := lumera.Blockchain.EVM.EthCall(ctx, from, &to, calldata, gasCap)
gas, _ := lumera.Blockchain.EVM.EstimateGas(ctx, from, &to, calldata, value, gasCap)
```

`Blockchain.FeeMarket` (`x/feemarket`) — `Params`, `BaseFee` (ulume decimal), `BlockGas`.
`Blockchain.PreciseBank` (`x/precisebank`) — `Remainder`, `FractionalBalance`.

`examples/evm-balance` walks through reading balance, nonce, base fee, and min gas price for a single address.

## Send an Ethereum-format transaction

`SendEthereumTransaction` resolves nonce, gas, and EIP-1559 caps from chain state, signs with the keyring's `eth_secp256k1` key, wraps in `MsgEthereumTx` with the required extension option, and broadcasts.

```go
to := common.HexToAddress("0x000000000000000000000000000000000000dEaD")
amount := sdkcrypto.Wei(sdkmath.NewInt(1)) // 1 ulume = 10^12 alume

res, err := lumera.Blockchain.EVM.SendEthereumTransaction(ctx, &to, nil, &blockchain.EthereumTxOptions{
    Value: amount,
})
if err != nil { log.Fatal(err) }
log.Printf("eth %s cosmos %s height %d gas %d",
    res.EthTxHash.Hex(), res.CosmosHash, res.Height, res.GasUsed)
```

Overrides on `EthereumTxOptions`: `Nonce`, `GasLimit`, `GasTipCap`, `GasFeeCap`, `Value`, `AccessList`.

Other helpers on `Blockchain.EVM`:

- `CallContract(ctx, to, calldata)` — read-only ABI call (forwards to `EthCall`).
- `DeployContract(ctx, bytecode, opts)` — contract creation; returns the deployed address (resolved before broadcast).
- `RawEthereumTx(ctx, signedTx)` — broadcast a tx signed elsewhere (hardware wallet, MetaMask).

`examples/evm-transfer` is a complete CLI.

## ERC20 conversion

`x/erc20` lets cosmos coins be wrapped as ERC20 and vice versa. Both directions use the regular cosmos signing pipeline (no Ethereum signature on the conversion message itself).

```go
// ulume -> ERC20 for the EVM address.
coin := sdk.NewInt64Coin("ulume", 1_000_000)
recv := common.HexToAddress("0xAbC...")
_, err := lumera.Blockchain.ConvertCoinToERC20(ctx, coin, recv, "memo")

// ERC20 -> ulume for the bech32 receiver. Sender is the 0x address derived
// from the client's eth_secp256k1 key.
contract := common.HexToAddress("0xCafe...")
_, err = lumera.Blockchain.ConvertERC20ToCoin(ctx, contract, sdkmath.NewInt(42), "lumera1recv...", "")

// Governance flows: RegisterERC20Tx, ToggleConversionTx.
```

Read-only ABI sugar (no Solidity compilation needed):

```go
bal, _ := lumera.Blockchain.ERC20.Erc20Balance(ctx, contract, holder)
sup, _ := lumera.Blockchain.ERC20.Erc20TotalSupply(ctx, contract)
allow, _ := lumera.Blockchain.ERC20.Erc20Allowance(ctx, contract, owner, spender)
meta, _ := lumera.Blockchain.ERC20.Erc20Metadata(ctx, contract) // name, symbol, decimals
```

Query helpers: `TokenPairs(ctx, pagination)`, `TokenPair(ctx, denomOr0x)`, `Params(ctx)`.

`examples/erc20-convert` exercises both directions.

## Lumera precompiles

Custom precompiles at fixed addresses (per `pkg/evm/precompiles`):

| Address | Module        | Wrapper                       |
| ------- | ------------- | ----------------------------- |
| `0x0901` | `x/action`    | `Blockchain.EVM.Action`       |
| `0x0902` | `x/supernode` | `Blockchain.EVM.Supernode`    |
| `0x0903` | CosmWasm      | `Blockchain.EVM.Wasm`         |

Each wrapper holds the precompile address plus the embedded ABI (loaded from `pkg/evm/precompiles/abi/`). Two methods:

- `Call(ctx, method, args...)` — read-only, routes through `EthCall`, returns the unpacked outputs.
- `Send(ctx, method, opts, args...)` — state-changing, signs and broadcasts as a `MsgEthereumTx`.

```go
// Read: get action module params from the EVM side.
out, err := lumera.Blockchain.EVM.Action.Call(ctx, "getParams")

// Write: approve an action via the precompile, signed with eth_secp256k1.
_, err = lumera.Blockchain.EVM.Action.Send(ctx, "approveAction", nil, "action-id-here")
```

The eight standard cosmos/evm precompiles (bank, staking, distribution, gov, ICS20, bech32, p256, slashing) are exposed as address constants only — see `pkg/evm/precompiles`.

`examples/precompile-action` issues a read-only `getParams` and a write `approveAction`.

## EVM migration

`x/evmigration` handles the one-time migration of pre-EVM (coin type 118) accounts and validators to the EVM-enabled (coin type 60) chain. Use queries for pre-flight, then submit the migration tx signed by the new address.

The migration tx helpers intentionally build zero-fee transactions for the chain-waived evmigration flow.

```go
// Pre-flight.
est, err := lumera.Blockchain.EVMigration.MigrationEstimate(ctx, legacyAddr)
if err != nil { log.Fatal(err) }
if !est.WouldSucceed {
    log.Fatalf("cannot migrate: %s", est.RejectionReason)
}

// Build proofs externally, then submit.
msg := blockchain.NewMsgClaimLegacyAccount(newAddr, legacyAddr, legacyProof, newProof)
res, err := lumera.Blockchain.ClaimLegacyAccountTx(ctx, msg, "migrate")
if err != nil { log.Fatal(err) }
log.Printf("migrated legacy=%s new=%s tx=%s height=%d",
    res.LegacyAddress, res.NewAddress, res.TxHash, res.Height)

// Validators use MigrateValidatorTx.
vmsg := blockchain.NewMsgMigrateValidator(newAddr, legacyAddr, legacyProof, newProof)
_, err = lumera.Blockchain.MigrateValidatorTx(ctx, vmsg, "")
```

Read-only helpers: `MigrationRecord(legacyAddress)`, `MigrationRecordByNewAddress(newAddress)`, `MigrationStats(ctx)`, `Params(ctx)`. `MigrationProof` construction is chain-specific — see the `x/evmigration` proto definitions.
