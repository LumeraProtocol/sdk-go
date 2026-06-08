# Actions and SuperNodes

Lumera's `x/action` and `x/supernode` modules are exposed off `Client.Blockchain`. Both ship with read-only queries plus opinionated transaction helpers; message constructors are exported for callers that need finer control.

## Query actions

```go
action, err := lumera.Blockchain.Action.GetAction(ctx, "action-id")
if err != nil {
    log.Fatal(err)
}
fmt.Println(action)
```

Other queries: `ListActions`, `ListActionsByType`, `ListActionsBySuperNode`, `ListActionsByBlockHeight`, `ListExpiredActions`, `QueryActionByMetadata`, `GetActionFee`, `Params`.

## Send action transactions

```go
res, err := lumera.Blockchain.RequestActionTx(ctx, creator, actionType, metadata, price, expiration, fileSizeKbs, "memo")
if err != nil {
    log.Fatal(err)
}
log.Printf("action %s tx %s height %d", res.ActionID, res.TxHash, res.Height)
```

Companion helpers: `ApproveActionTx`, `FinalizeActionTx`, `UpdateActionParamsTx`. Message constructors: `NewMsgRequestAction`, `NewMsgApproveAction`, `NewMsgFinalizeAction`, `NewMsgUpdateParams`.

## Manage SuperNodes

Registration and lifecycle changes use the `lumera.Blockchain.SuperNode` transaction helpers:

```go
_, err := lumera.Blockchain.RegisterSupernodeTx(ctx, cfg.Address, "lumeravaloper...", "1.2.3.4", "lumera1sn...", "26656", "")
if err != nil {
    log.Fatal(err)
}
```

Other tx helpers: `DeregisterSupernodeTx`, `StartSupernodeTx`, `StopSupernodeTx`, `UpdateSupernodeTx`, `UpdateSuperNodeParamsTx`. Query helpers: `GetSuperNode`, `GetSuperNodeBySuperNodeAddress`, `ListSuperNodes`, `GetTopSuperNodesForBlock`, `GetTopSuperNodesForBlockWithOptions`, `Params`.

## EVM-side equivalents

Both `x/action` and `x/supernode` are also reachable through the Lumera precompiles at `0x0901` and `0x0902` for Solidity / EVM callers. See [evm.md](evm.md#lumera-precompiles) for the precompile wrapper.
