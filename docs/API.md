# Lumera Go SDK – API Overview

This is a concise map of the exported Go surface. For full GoDoc see `pkg.go.dev/github.com/LumeraProtocol/sdk-go`.

## Package `client`

- `client.New(ctx, Config, keyring, opts...) (*Client, error)` builds a unified client exposing `Blockchain` and `Cascade`.
- `Config` (alias of `client/config.Config`): chain endpoints, address/key, timeouts, wait-tx config, message sizes, retries, optional logger.
- Options: `WithChainID`, `WithKeyName`, `WithGRPCEndpoint`, `WithRPCEndpoint`, `WithBlockchainTimeout`, `WithStorageTimeout`, `WithMaxRetries`, `WithMaxMessageSize`, `WithWaitTxConfig`, `WithLogLevel`, `WithLogger`.
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
- Shared tx utilities: `BuildAndSignTx`, `Simulate`, `Broadcast`, `WaitForTxInclusion`, `GetTx`, `ExtractEventAttribute` (for parsing event attributes like `action_id`).

## Package `types`

- Chain models: `Action`, `SuperNode` converters from protobuf responses.
- Results: `ActionResult` (tx hash, height, action ID), `CascadeResult` (action result + task ID), `DownloadResult` (action ID, task ID, output path).
- Errors: `ErrInvalidConfig`, `ErrNotFound`, `ErrTimeout`, `ErrInvalidSignature`, `ErrTaskFailed`.

## Logging

Logging uses `go.uber.org/zap`. Use `client.WithLogLevel` to set the default level (error by default), or pass a custom `*zap.Logger` via `client.WithLogger`.
