# Lumera Go SDK – Developer Guide

This guide is for engineers building on the Lumera blockchain and Cascade storage. It shows how to configure the SDK, call the unified client, and run the included examples.

## Prerequisites

- Go 1.25+ with module support.
- Access to Lumera endpoints: `grpc` (chain queries/tx), `rpc` (websocket for tx inclusion), and at least one SuperNode for Cascade uploads/downloads.
- A Cosmos keyring entry that can sign Lumera transactions (`github.com/cosmos/cosmos-sdk/crypto/keyring` is used throughout the SDK).

## Install and Configure

```bash
go get github.com/LumeraProtocol/sdk-go
```

### Configuration reference

`client.Config` (in `client/config`) drives both blockchain and Cascade clients:

- `ChainID`, `GRPCEndpoint`, `RPCEndpoint` – chain connection details. gRPC uses TLS automatically for non-local hosts/port 443.
- `Address`, `KeyName` – Cosmos account info in your keyring.
- `BlockchainTimeout`, `StorageTimeout` – default deadlines for chain and Cascade operations.
- `MaxRecvMsgSize`, `MaxSendMsgSize`, `MaxRetries` – transport tuning.
- `WaitTx` – controls websocket vs polling behaviour when waiting for tx inclusion (see defaults in `client/config`).
- `Logger` – optional; when set, SDK operations emit diagnostics.
- `LogLevel` – default logging threshold when no custom logger is supplied (default: error).

You can override fields with `client.With...` option helpers when calling `client.New`.

### Creating a client

```go
ctx := context.Background()
kr, _ := keyring.New("lumera", "test", "/tmp", nil)

cfg := client.Config{
    ChainID:      "lumera-testnet-2",
    GRPCEndpoint: "localhost:9090",
    RPCEndpoint:  "http://localhost:26657",
    Address:      "lumera1abc...",
    KeyName:      "my-key",
}

logger := zap.NewExample()
lumera, err := client.New(ctx, cfg, kr, client.WithLogger(logger))
if err != nil {
    logger.Error("client init failed", zap.Error(err))
}
defer lumera.Close()
```

`client.Client` exposes `Blockchain` (gRPC chain modules) and `Cascade` (SuperNode SDK + SnApi).

### Using the factory for multiple signers

`client.NewFactory` keeps a shared config/keyring and returns signer-specific clients:

```go
factory, _ := client.NewFactory(cfg, kr)
alice, _ := factory.WithSigner(ctx, "lumera1alice...", "alice")
bob, _ := factory.WithSigner(ctx, "lumera1bob...", "bob")
defer alice.Close()
defer bob.Close()
```

## Tutorials

### 1) Query actions (read-only)

```go
action, err := lumera.Blockchain.Action.GetAction(ctx, "action-id")
if err != nil {
    log.Fatal(err)
}
fmt.Println(action)
```

### 2) Upload a file to Cascade (one-shot)

Steps: build Cascade metadata, register an action on-chain, upload bytes to SuperNodes, wait for completion.

```go
result, err := lumera.Cascade.Upload(ctx, cfg.Address, lumera.Blockchain, "/path/to/file",
    cascade.WithPublic(true), // optional: make file public
)
if err != nil {
    log.Fatal(err)
}
log.Printf("action=%s task=%s", result.ActionID, result.TaskID)
```

`Upload` wraps `Client.CreateRequestActionMessage`, `Client.SendRequestActionMessage`, and `Client.UploadToSupernode`. For manual control, call those methods separately and reuse the returned `MsgRequestAction` or `types.ActionResult`.

### 3) Download from Cascade

```go
dl, err := lumera.Cascade.Download(ctx, "action-id", "/tmp/downloads")
if err != nil {
    log.Fatal(err)
}
log.Printf("downloaded to %s", dl.OutputPath)
```

### 4) Subscribe to Cascade task events

The Cascade client bridges SuperNode SDK events and adds SDK-specific ones (prefixed `sdk-go:`).

```go
lumera.Cascade.SubscribeToAllEvents(ctx, func(_ context.Context, e event.Event) {
    log.Printf("%s task=%s msg=%v", e.Type, e.TaskID, e.Data[event.KeyMessage])
})
```

### 5) Send on-chain actions explicitly

```go
msg, meta, err := lumera.Cascade.CreateRequestActionMessage(ctx, cfg.Address, "/path/file", nil)
_ = meta // metadata bytes used in the action
if err != nil { log.Fatal(err) }

ar, err := lumera.Cascade.SendRequestActionMessage(ctx, lumera.Blockchain, msg, "memo", nil)
if err != nil { log.Fatal(err) }
log.Printf("action registered: %s", ar.ActionID)

// Approve the action (if your flow requires it)
approve := blockchain.NewMsgApproveAction(cfg.Address, ar.ActionID)
_, err = lumera.Cascade.SendApproveActionMessage(ctx, lumera.Blockchain, approve, "")
```

For offline/ICA-style flows, the package-level `cascade.CreateApproveActionMessage` helper builds approvals without SuperNode dependencies.

### 6) Interchain Accounts (ICA) flow

Use ICA when a controller chain account submits Lumera `MsgRequestAction` messages on behalf of an ICA address. The SDK helps build the request message and ICA packet, but you still broadcast the controller-chain `MsgSendTx` with your controller chain tooling.

Key points:

- You must provide Lumera chain `grpc` + `chain-id` so metadata (price/expiration) can be computed.
- For ICA, set the ICA creator address and app pubkey on the request message.
- The Cascade client uses `ICAOwnerKeyName` + `ICAOwnerHRP` to derive the controller owner address.
  `appPubkey` should be the controller key's pubkey bytes from the keyring.

```go
ctx := context.Background()
// Reuse kr from the client setup above.
cascadeClient, err := cascade.New(ctx, cascade.Config{
    ChainID:         "lumera-testnet-2",
    GRPCAddr:        "localhost:9090",
    Address:         "lumera1abc...",
    KeyName:         "my-key",
    ICAOwnerKeyName: "my-key",
    ICAOwnerHRP:     "inj",
}, kr)
if err != nil { log.Fatal(err) }
defer cascadeClient.Close()

uploadOpts := &cascade.UploadOptions{}
cascade.WithICACreatorAddress("lumera1ica...")(uploadOpts)
cascade.WithAppPubkey(appPubkey)(uploadOpts)

msg, _, err := cascadeClient.CreateRequestActionMessage(ctx, "lumera1abc...", "/path/file", uploadOpts)
if err != nil { log.Fatal(err) }

any, err := ica.PackRequestAny(msg)
if err != nil { log.Fatal(err) }

packet, err := ica.BuildICAPacketData([]*codectypes.Any{any})
if err != nil { log.Fatal(err) }

msgSendTx, err := ica.BuildMsgSendTx(ownerAddr, "connection-0", 600_000_000_000, packet)
if err != nil { log.Fatal(err) }

// Broadcast msgSendTx using your controller-chain SDK or CLI.
```

See `examples/ica-request-tx` for a full CLI that builds the ICA packet and prints the JSON.

### 7) Manage SuperNodes

Registration/updates use `lumera.Blockchain.SuperNode` transaction helpers:

```go
_, err := lumera.Blockchain.RegisterSupernodeTx(ctx, cfg.Address, "lumeravaloper...", "1.2.3.4", "lumera1sn...", "26656", "")
if err != nil { log.Fatal(err) }
```

Query helpers include `GetSuperNode`, `ListSuperNodes`, and `GetTopSuperNodesForBlock`.

## Examples and testing

- Run tests: `make test`
- Build samples: `make examples`
- Execute tutorials end-to-end: `go run ./examples/cascade-upload`, `go run ./examples/cascade-download`, `go run ./examples/query-actions`, `go run ./examples/multi-account`, `go run ./examples/ica-request-tx --help`

## Troubleshooting

- **Tx inclusion timing out**: adjust `WaitTx` polling/backoff (see `client/config`). Ensure `RPCEndpoint` allows websocket subscriptions.
- **gRPC TLS errors**: remote hosts/port 443 default to TLS; for local nodes use `localhost:9090` or `127.0.0.1:9090`.
- **Key not found**: confirm the key name exists in the keyring path you passed to `keyring.New`.
- **SuperNode availability**: Cascade operations require reachable SuperNodes; watch `sdk:supernodes_unavailable` events for diagnostics.
