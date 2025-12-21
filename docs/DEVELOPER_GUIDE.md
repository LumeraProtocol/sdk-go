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

lumera, err := client.New(ctx, cfg, kr, client.WithLogger(log.Default()))
if err != nil {
    log.Fatal(err)
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

`Upload` wraps `CreateRequestActionMessage`, `SendRequestActionMessage`, and `UploadToSupernode`. For manual control, call those methods separately and reuse the returned `MsgRequestAction` or `types.ActionResult`.

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

For offline/ICA-style flows, package-level helpers `cascade.CreateRequestActionMessage` and `cascade.CreateApproveActionMessage` avoid SuperNode dependencies.

### 6) Manage SuperNodes

Registration/updates use `lumera.Blockchain.SuperNode` transaction helpers:

```go
_, err := lumera.Blockchain.RegisterSupernodeTx(ctx, cfg.Address, "lumeravaloper...", "1.2.3.4", "lumera1sn...", "26656", "")
if err != nil { log.Fatal(err) }
```

Query helpers include `GetSuperNode`, `ListSuperNodes`, and `GetTopSuperNodesForBlock`.

## Examples and testing

- Run tests: `make test`
- Build samples: `make examples`
- Execute tutorials end-to-end: `go run ./examples/cascade-upload`, `go run ./examples/cascade-download`, `go run ./examples/query-actions`, `go run ./examples/multi-account`

## Troubleshooting

- **Tx inclusion timing out**: adjust `WaitTx` polling/backoff (see `client/config`). Ensure `RPCEndpoint` allows websocket subscriptions.
- **gRPC TLS errors**: remote hosts/port 443 default to TLS; for local nodes use `localhost:9090` or `127.0.0.1:9090`.
- **Key not found**: confirm the key name exists in the keyring path you passed to `keyring.New`.
- **SuperNode availability**: Cascade operations require reachable SuperNodes; watch `sdk:supernodes_unavailable` events for diagnostics.
