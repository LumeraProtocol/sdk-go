# Lumera Go SDK – API Overview

This is a concise map of the exported Go surface. For full GoDoc see `pkg.go.dev/github.com/LumeraProtocol/sdk-go`.

## Package `client`

- `client.New(ctx, Config, keyring, opts...) (*Client, error)` builds a unified client exposing `Blockchain` and `Cascade`.
- `Config` (alias of `client/config.Config`): chain endpoints, address/key, timeouts, wait-tx config, message sizes, retries, optional logger.
- Options: `WithChainID`, `WithKeyName`, `WithGRPCEndpoint`, `WithRPCEndpoint`, `WithBlockchainTimeout`, `WithStorageTimeout`, `WithMaxRetries`, `WithMaxMessageSize`, `WithWaitTxConfig`, `WithLogLevel`, `WithLogger`, `WithEVMChainID`, `WithEVMNativeDenom`, `WithEVMExtendedDenom`, `WithEVMGasCaps`.
- `Client.Blockchain` is a `*blockchain.Client`; `Client.Cascade` is a `*cascade.Client`. `Close()` tears both down.
- `NewFactory` captures a base config/keyring for multi-signer flows; `Factory.WithSigner` returns a per-signer `Client`.

## Package `cascade`

- `Config`: `ChainID`, `GRPCAddr`, `Address`, `KeyName`, `Timeout`, `LogLevel`.
- Upload helpers:
  - `Upload(ctx, creator, bc, filePath, opts...) (*types.CascadeResult, error)` – one-shot metadata build + request action tx + SuperNode upload.
  - `Client.CreateRequestActionMessage`, `Client.SendRequestActionMessage`, `Client.UploadToSupernode` – stepwise control; optional `UploadOption`s include `WithPublic(bool)` and `WithID(string)`.
- Download helper: `Download(ctx, actionID, outputDir, opts...) (*types.DownloadResult, error)`.
- Approve helpers: client methods `CreateApproveActionMessage`/`SendApproveActionMessage` and package-level `CreateApproveActionMessage`/`SendApproveActionMessage` (use `WithApproveCreator`, `WithApproveBlockchain`, `WithApproveMemo`).
- Event subscriptions: `SubscribeToEvents` and `SubscribeToAllEvents` bridge SuperNode SDK events; event types and metadata keys are defined in `cascade/event`.
- Task utilities: `TaskManager` (in `cascade/task.go`) powers `UploadToSupernode`/`Download`; emits SDK-local events prefixed `sdk-go:`.

## Package `blockchain`

- Config: gRPC/RPC endpoints, chain ID, timeouts, message sizes, wait-tx config.
- Action module:
  - Queries: `GetAction`, `ListActions`, `ListActionsByType`, `ListActionsBySuperNode`, `ListActionsByBlockHeight`, `ListExpiredActions`, `QueryActionByMetadata`, `GetActionFee`, `Params`.
  - Tx helpers: `RequestActionTx`, `ApproveActionTx`, `FinalizeActionTx`, `UpdateActionParamsTx`. Message constructors: `NewMsgRequestAction`, `NewMsgApproveAction`, `NewMsgFinalizeAction`, `NewMsgUpdateParams`.
- SuperNode module:
  - Queries: `GetSuperNode`, `GetSuperNodeBySuperNodeAddress`, `ListSuperNodes`, `GetTopSuperNodesForBlock`, `GetTopSuperNodesForBlockWithOptions`, `Params`.
  - Tx helpers: `RegisterSupernodeTx`, `DeregisterSupernodeTx`, `StartSupernodeTx`, `StopSupernodeTx`, `UpdateSupernodeTx`, `UpdateSuperNodeParamsTx`. Message constructors mirror these names.
- Claim and Audit modules: query clients are wired; add methods as the chain exposes additional endpoints.
- EVMigration module:
  - Queries: `Params`, `MigrationRecord`, `MigrationRecordByNewAddress`, `MigrationEstimate`, `MigrationStats`.
  - Tx helpers: `ClaimLegacyAccountTx`, `MigrateValidatorTx`. Message constructors: `NewMsgClaimLegacyAccount`, `NewMsgMigrateValidator`. Result type: `MigrationResult` (legacy/new address, tx hash, height).
- EVM module (x/vm):
  - Queries: `Code`, `Storage`, `Balance` (alume), `EthAccount`, `CosmosAccount`, `Params`, `BaseFee` (alume), `Config`, `GlobalMinGasPrice`, `EthCall`, `EstimateGas`, `TraceTx`.
  - Tx helpers: `SendEthereumTransaction`, `DeployContract`, `CallContract`, `RawEthereumTx`. Options struct: `EthereumTxOptions` (Nonce/GasLimit/GasTipCap/GasFeeCap/Value/AccessList). Result: `EthereumTransactionResult` (eth tx hash, cosmos hash, height, gas used, vm error, return data, logs).
  - Precompile wrappers (`EVM.Action`, `EVM.Supernode`, `EVM.Wasm`): generic `Call(ctx, method, args...)` and `Send(ctx, method, opts, args...)` route through the precompile address using the embedded ABI; addresses + ABIs live in [`pkg/evm/precompiles`](../pkg/evm/precompiles).
- ERC20 module (x/erc20):
  - Queries: `TokenPairs` (paginated), `TokenPair`, `Params`.
  - Tx helpers: `ConvertCoinToERC20`, `ConvertERC20ToCoin`, `RegisterERC20Tx`, `ToggleConversionTx`. Message constructors: `NewMsgConvertCoin`, `NewMsgConvertERC20`, `NewMsgRegisterERC20`, `NewMsgToggleConversion`. Result: `ConversionResult`.
  - ABI sugar: `Erc20Balance`, `Erc20TotalSupply`, `Erc20Allowance`, `Erc20Metadata` issue ABI-packed read calls through the EVM client.
- FeeMarket module (x/feemarket):
  - Queries: `Params`, `BaseFee` (ulume decimal), `BlockGas`.
- PreciseBank module (x/precisebank):
  - Queries: `Remainder`, `FractionalBalance` (sub-ulume sub-balances backing 18-decimal EVM views).
- Shared tx utilities: `BuildAndSignTx`, `BuildAndSignTxWithGasAdjustment`, `BuildAndSignTxWithOptions`, `PrepareTx` + `SignPreparedTx`, `Simulate`, `Broadcast`, `BroadcastAndWait`, `WaitForTxInclusion`, `GetTx`, `GetTxsByEvents`, `ExtractEventAttribute` (for parsing event attributes like `action_id`). `BuildAndSignTxWithOptions` rejects `MsgEthereumTx` to prevent accidental double-signing — use `EVMClient.SendEthereumTransaction` instead.

## Package `types`

- Chain models: `Action`, `SuperNode` converters from protobuf responses.
- Results: `ActionResult` (tx hash, height, action ID), `CascadeResult` (action result + task ID), `DownloadResult` (action ID, task ID, output path).
- Errors: `ErrInvalidConfig`, `ErrNotFound`, `ErrTimeout`, `ErrInvalidSignature`, `ErrTaskFailed`.

## Package `pkg/crypto`

Crypto helpers for keyring management, key import, address derivation, and transaction signing. A single keyring supports both Cosmos (`secp256k1`) and EVM (`eth_secp256k1`) key types.

- `KeyType` enum: `KeyTypeCosmos` (secp256k1, BIP44 coin type 118) and `KeyTypeEVM` (eth_secp256k1, BIP44 coin type 60). Helper methods: `String()`, `HDPath()`, `SigningAlgo()`.
- `KeyringParams` / `DefaultKeyringParams()`: configuration for keyring initialization (app name, backend, directory).
- `NewKeyring(KeyringParams) (keyring.Keyring, error)`: creates a keyring supporting both Cosmos and EVM key algorithms.
- `LoadKeyring(keyName, mnemonicFile string, keyType KeyType) (keyring.Keyring, []byte, string, error)`: creates a test keyring and imports a mnemonic with the given key type; returns the keyring, pubkey bytes, and Lumera address.
- `ImportKey(kr keyring.Keyring, keyName, mnemonicFile, hrp string, keyType KeyType) ([]byte, string, error)`: imports a mnemonic into an existing keyring under the given key name and key type; returns pubkey bytes and address for the specified HRP.
- `AddressFromKey(kr, keyName, hrp) (string, error)`: derives an HRP-specific bech32 address from a keyring key without mutating global config.
- `EVMAddressFromKey(kr, keyName) (string, error)`: derives the 0x-prefixed EIP-55 hex address for an `eth_secp256k1` key; returns an error for `secp256k1` keys.
- `NewDefaultTxConfig() client.TxConfig`: builds a protobuf tx config with Lumera action and crypto interfaces plus the EVM modules (`evmigration`, `erc20`, `feemarket`, `precisebank`, `vm`) registered.
- `SignTxWithKeyring(kr, keyName, chainID string, txBuilder, txConfig) ([]byte, error)`: signs a transaction using Cosmos SDK builders.
- `SignEthereumTx(kr, keyName, chainID *big.Int, tx *ethtypes.Transaction)`: signs a go-ethereum tx using the keyring's `eth_secp256k1` key with the latest EIP-155 signer. `WrapAsMsgEthereumTx(signed)` packages it as a `MsgEthereumTx`; `RecoverSender(signed)` returns the recovered EVM address.
- `EVMToBech32(addr, hrp)` / `Bech32ToEVM(bech32Addr)`: byte-level conversion between 20-byte EVM addresses and bech32 (round-trips only for `eth_secp256k1`-derived accounts).
- `Wei(ulume)` / `ULume(alume)` / `ULumeDecToWei(dec)`: bridge the 6↔18 decimal boundary defined by `x/precisebank` (10^12 factor).

## Package `pkg/evm/precompiles`

- Embedded Hardhat-format ABIs for Lumera's custom precompiles (`ActionABI`, `SupernodeABI`, `WasmABI`) plus named address constants (`ActionAddress` `0x0901`, `SupernodeAddress` `0x0902`, `WasmAddress` `0x0903`).
- Address constants for the 8 standard cosmos/evm precompiles (`P256Address`, `Bech32Address`, `StakingAddress`, `DistributionAddress`, `ICS20Address`, `BankAddress`, `GovAddress`, `SlashingAddress`).
- Generic helpers: `PackCall(abi, method, args...)`, `UnpackReturn(abi, method, ret)`.

## Package `ica`

ICA (Interchain Accounts / ICS-27) controller for registering interchain accounts and executing messages across chains.

- `Config`: controller/host chain configuration (`Controller`, `Host` as `base.Config`), `Keyring`, `KeyName`, optional `HostKeyName` (separate key for host chain operations), IBC settings (`ConnectionID`, `CounterpartyConnectionID`, `Ordering`, `RelativeTimeout`), and polling parameters (`PollDelay`, `PollRetries`, `AckRetries`).
- `NewController(ctx, Config) (*Controller, error)`: creates a gRPC-based ICA controller. When `HostKeyName` is set, host chain operations use a different key than the controller chain signer.
- `Controller.EnsureICAAddress(ctx)`: resolves or registers an ICA address and polls until available.
- `Controller.SendRequestAction(ctx, *MsgRequestAction) (*ActionResult, error)`: sends a request action over ICA, waits for the ack, and returns the action ID.
- `Controller.SendApproveAction(ctx, *MsgApproveAction) (string, error)`: sends an approve action over ICA.
- Packet helpers: `PackRequestAny`, `PackApproveAny`, `BuildICAPacketData`, `BuildMsgSendTx`.
- Ack extraction: `ExtractRequestActionIDsFromAck`, `ExtractRequestActionIDsFromTxMsgData`.
- CLI helpers: `ParseTxHashJSON`, `ExtractPacketInfoFromTxJSON`, `DecodePacketAcknowledgementJSON`.

## Logging

Logging uses `go.uber.org/zap`. Use `client.WithLogLevel` to set the default level (error by default), or pass a custom `*zap.Logger` via `client.WithLogger`.
