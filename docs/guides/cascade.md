# Cascade Storage

The `Cascade` client bundles SuperNode uploads/downloads with the on-chain action lifecycle.

## Upload a file (one-shot)

`Upload` builds Cascade metadata, registers an action on-chain, uploads bytes to SuperNodes, and waits for completion.

```go
result, err := lumera.Cascade.Upload(ctx, cfg.Address, lumera.Blockchain, "/path/to/file",
    cascade.WithPublic(true), // optional: make file public
)
if err != nil {
    log.Fatal(err)
}
log.Printf("action=%s task=%s", result.ActionID, result.TaskID)
```

`Upload` wraps `Client.CreateRequestActionMessage`, `Client.SendRequestActionMessage`, and `Client.UploadToSupernode`. Call those methods separately for manual control and reuse the returned `MsgRequestAction` or `types.ActionResult`.

## Download

```go
dl, err := lumera.Cascade.Download(ctx, "action-id", "/tmp/downloads")
if err != nil {
    log.Fatal(err)
}
log.Printf("downloaded to %s", dl.OutputPath)
```

## Subscribe to task events

The Cascade client bridges SuperNode SDK events and adds SDK-specific diagnostics such as `sdk:supernodes_unavailable`.

```go
lumera.Cascade.SubscribeToAllEvents(ctx, func(_ context.Context, e event.Event) {
    log.Printf("%s task=%s msg=%v", e.Type, e.TaskID, e.Data[event.KeyMessage])
})
```

## Stepwise control

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

For offline / ICA-style flows, the package-level `cascade.CreateApproveActionMessage` helper builds approvals without SuperNode dependencies.
