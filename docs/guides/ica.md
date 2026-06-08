# Interchain Accounts (ICA)

The `ica` package provides an ICS-27 controller plus low-level packet helpers. Use it when a controller-chain account submits Lumera `MsgRequestAction` / `MsgApproveAction` messages on behalf of an ICA address.

## Controller flow

`ica.Controller` connects to both a controller chain and the Lumera host chain over gRPC and handles ICA registration, IBC packet construction, transaction broadcasting, acknowledgement polling, and action ID extraction:

```go
ctrl, _ := ica.NewController(ctx, ica.Config{
    Controller:   controllerBaseConfig,
    Host:         hostBaseConfig,
    Keyring:      kr,
    KeyName:      "controller-key",
    HostKeyName:  "host-key",       // optional: separate key for host chain
    ConnectionID: "connection-0",
})
defer ctrl.Close()

addr, _ := ctrl.EnsureICAAddress(ctx)         // register + poll until ready
result, _ := ctrl.SendRequestAction(ctx, msg) // send, wait for ack, return action ID
txHash, _ := ctrl.SendApproveAction(ctx, approveMsg)
```

For lower-level or offline workflows, the packet-building helpers are exported separately: `PackRequestAny`, `BuildICAPacketData`, `BuildMsgSendTx`.

## Lower-level packet flow

When you want to broadcast the controller-chain `MsgSendTx` with your own tooling, the SDK still helps build the Lumera-side payload:

```go
ctx := context.Background()
// Reuse kr from the client setup.
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

## Tips

- You must provide Lumera chain `grpc` + `chain-id` so metadata (price/expiration) can be computed.
- Set the ICA creator address and app pubkey on the request message.
- The Cascade client uses `ICAOwnerKeyName` + `ICAOwnerHRP` to derive the controller owner address; `appPubkey` should be the controller key's pubkey bytes from the keyring.
- When controller and host chains use different key types, import keys under separate names into the same keyring and set `HostKeyName` on the ICA `Config`. See [crypto.md](crypto.md).

## Strengths

- **Minimal setup** — only gRPC endpoints, a keyring, and an IBC connection ID are required. No Docker, no relayer binary, no chain binaries.
- **End-to-end in one call** — `SendRequestAction` builds the ICA packet, broadcasts on the controller chain, waits for tx inclusion, resolves the counterparty channel, polls for the host-chain acknowledgement, and extracts the action ID.
- **Mixed key type support** — controller and host chains can use different cryptographic key types (`KeyTypeCosmos` / `KeyTypeEVM`) by setting `HostKeyName` to a separate key in the same keyring.
- **Resilient polling** — configurable retry counts and delays for both ICA registration (`PollRetries` / `PollDelay`) and acknowledgement waiting (`AckRetries`).
- **Tight Lumera integration** — purpose-built for `MsgRequestAction` and `MsgApproveAction`, with typed results (`ActionResult`) and Cascade metadata compatibility.

## Limitations

- **Requires running chains** — the controller connects to live gRPC endpoints. It does not spin up chains or relayers; infrastructure must already be deployed.
- **Lumera-specific high-level methods** — `SendRequestAction` and `SendApproveAction` are tailored to Lumera action messages. Generic ICA message execution requires using the lower-level packet helpers directly.
- **No chain lifecycle management** — unlike e2e testing frameworks (e.g., interchaintest), there is no built-in chain provisioning, genesis configuration, or relayer orchestration.
- **Relayer dependency** — IBC packet relay between controller and host chains depends on an external relayer (e.g., Hermes). The controller does not relay packets itself.

## When to use this vs. interchaintest

| Aspect              | `ica.Controller`                | interchaintest                            |
| ------------------- | ------------------------------- | ----------------------------------------- |
| **Purpose**         | Production client / scripting   | E2E integration testing                   |
| **Infrastructure**  | Connects to running chains      | Spins up chains + relayers in Docker      |
| **Setup effort**    | Config struct + keyring         | Docker, chain binaries, genesis config    |
| **Iteration speed** | Fast (gRPC calls)               | Slower (container lifecycle + blocks)     |
| **Scope**           | Lumera ICA operations           | Any IBC flow, any chain                   |

Use `ica.Controller` when you have running chains and need to execute ICA operations in production or automation scripts. Use interchaintest when you need to validate the full ICA flow in CI from scratch without external infrastructure.
