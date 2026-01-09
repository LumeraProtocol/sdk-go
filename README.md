# Lumera Go SDK

Official Go SDK for the Lumera Protocol - a next-generation blockchain platform for AI and decentralized storage.

## Features

- 🔗 Unified APIs — Single interface unifying Lumera gRPC, SuperNode SDK, and SnApi
- 📦 Type-Safe — Full Go type definitions for all Lumera modules
- 🚀 High-Level API — Simple methods for complex operations
- 🔐 Secure — Built on Cosmos SDK's proven cryptography
- 📝 Well-Documented — Comprehensive examples and documentation

## Unified APIs

This SDK unifies three distinct Lumera interfaces behind one easy client:

- Lumera API (via gRPC): Standard blockchain queries and transactions.
  - Accessed via [client.Client.Blockchain](client/client.go:16), which exposes module clients like [blockchain.Client.Action](blockchain/client.go:50) and [blockchain.Client.SuperNode](blockchain/client.go:51).
- SuperNode SDK: Direct interaction with Supernodes for data operations, task lifecycle, and event subscriptions.
  - Accessed via [client.Client.Cascade](client/client.go:17), which wraps the SuperNode SDK. Key methods: [cascade.Client.Upload](cascade/cascade.go:38), [cascade.Client.Download](cascade/cascade.go:127), [cascade.Client.SubscribeToEvents](cascade/events.go:9).
- SnApi (via gRPC): Supernode gRPC interface used under the hood by the SuperNode SDK to communicate with Supernodes.
  - This SDK integrates SnApi through the SuperNode SDK; direct, first-class SnApi wrappers may be added in future releases.

## Installation

```bash
go get github.com/LumeraProtocol/sdk-go
```

## Quick Start

```go
package main

import (
    "context"
    "log"
    
    "github.com/cosmos/cosmos-sdk/crypto/keyring"
    lumerasdk "github.com/LumeraProtocol/sdk-go/client"
)

func main() {
    ctx := context.Background()
    
    // Initialize keyring (for queries-only flows, any key name/address placeholders are fine)
    kr, err := keyring.New("lumera", "test", "/tmp", nil)
    if err != nil {
        log.Fatal(err)
    }
    
    // Create client
    client, err := lumerasdk.New(ctx, lumerasdk.Config{
        ChainID:      "lumera-testnet-2",
        GRPCEndpoint: "localhost:9090",
        RPCEndpoint:  "http://localhost:26657",
        Address:      "lumera1abc...",
        KeyName:      "my-key",
    }, kr, lumerasdk.WithLogger(log.Default()))
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Query an action via Lumera gRPC API
    action, err := client.Blockchain.Action.GetAction(ctx, "action-123")
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Action: %+v", action)
}
```

Note: For Cascade file operations (SuperNode SDK + SnApi), see:

- [examples/cascade-upload](examples/cascade-upload)
- [examples/cascade-download](examples/cascade-download)

### ICA Cascade Flow (Upload/Download)

If you need to register Cascade actions via an interchain account (ICS-27) while still using the SDK for metadata, uploads, and downloads, the high-level flow is:

1. Resolve the controller-chain owner address and ensure the ICA host address is registered.
2. Fund the ICA host address on Lumera so it can pay fees.
3. Initialize `cascade.Client` with `ICAOwnerKeyName` and `ICAOwnerHRP` to allow controller-chain download signatures.
4. Build and submit `MsgRequestAction` over ICA, then use the returned action ID for the supernode upload.
5. Download with a controller-chain signer (or let the client derive it).
6. After the action is `DONE`, send `MsgApproveAction` over ICA and wait for `APPROVED`.

Minimal wiring (controller-chain submit helpers are application-specific):

```go
// setup cascade client with ICA owner config
cascadeClient, err := cascade.New(ctx, cascade.Config{
    ChainID:         lumeraChainID,
    GRPCAddr:        lumeraGRPC,
    Address:         lumeraAddr,  // host chain address
    KeyName:         lumeraKeyName,
    ICAOwnerKeyName: simdKeyName, // controller chain key name
    ICAOwnerHRP:     "cosmos",    // controller chain address HRP
    Timeout:         30 * time.Second,
}, kr)

// send function submits ICA MsgRequestAction and returns action id
sendFunc := func(ctx context.Context, msg *actiontypes.MsgRequestAction, _ []byte, _ string, _ *cascade.UploadOptions) (*types.ActionResult, error) {
    actionIDs, err := sendICARequestTx(ctx, []*actiontypes.MsgRequestAction{msg})
    if err != nil {
        return nil, err
    }
    return &types.ActionResult{ActionID: actionIDs[0]}, nil
}

res, err := cascadeClient.Upload(ctx, icaAddr, nil, filePath,
    cascade.WithICACreatorAddress(icaAddr),
    cascade.WithAppPubkey(simdPubkey),
    cascade.WithICASendFunc(sendFunc),
)

// download using controller address for signature (optional override)
_, err = cascadeClient.Download(ctx, res.ActionID, downloadDir, cascade.WithDownloadSignerAddress(ownerAddr))

// approve via ICA when action is DONE
approveMsg, _ := cascade.CreateApproveActionMessage(ctx, res.ActionID, cascade.WithApproveCreator(icaAddr))
_ = sendICAApproveTx(ctx, []*actiontypes.MsgApproveAction{approveMsg})
```

Notes:
- `WithICASendFunc` is required when using `WithICACreatorAddress` and/or `WithAppPubkey`.
- Some chains enforce `app_pubkey` for ICA creators; set it to the controller-chain key pubkey.
- The full end-to-end test lives in `../lumera/devnet/tests/hermes/ibc_hermes_ica_test.go` (`TestICACascadeFlow`).
- Devnet test link: [lumera/devnet/tests/hermes/ibc_hermes_ica_test.go](https://github.com/LumeraProtocol/lumera/blob/main/devnet/tests/hermes/ibc_hermes_ica_test.go)

#### Sending ICA RequestAction (helpers + CLI)

The SDK includes small helpers to assemble ICS-27 packets and decode acknowledgements:

- `PackRequestForICA`: packs `MsgRequestAction` into `google.protobuf.Any` bytes.
- `BuildICAPacketData`: wraps one or more Any messages into `InterchainAccountPacketData` for `EXECUTE_TX`.
- `BuildMsgSendTx`: builds controller-side `MsgSendTx` if you submit the tx programmatically.
- `ExtractRequestActionIDsFromAck` / `ExtractRequestActionIDsFromTxMsgData`: pull action IDs out of acknowledgements.
- `ParseTxHashJSON`, `ExtractPacketInfoFromTxJSON`, `DecodePacketAcknowledgementJSON`: CLI-friendly helpers for tx hash, packet info, and ack decoding.

Build a packet JSON file (used by the CLI) from a `MsgRequestAction`:

```go
msg, _, _ := cascadeClient.CreateRequestActionMessage(ctx, icaAddr, filePath, &cascade.UploadOptions{
    ICACreatorAddress: icaAddr,
    AppPubkey:         simdPubkey,
})

anyBz, _ := cascade.PackRequestForICA(msg)
var any codectypes.Any
_ = gogoproto.Unmarshal(anyBz, &any)

packet, _ := cascade.BuildICAPacketData([]*codectypes.Any{&any})
packetJSON, _ := codec.NewProtoCodec(codectypes.NewInterfaceRegistry()).MarshalJSON(&packet)
_ = os.WriteFile("ica-packet.json", packetJSON, 0o600)
```

Send the packet on the controller chain (example `simd` CLI):

```bash
simd tx interchain-accounts controller send-tx <connection-id> ica-packet.json \
  --from <controller-key> \
  --chain-id <controller-chain-id> \
  --gas auto \
  --gas-adjustment 1.3 \
  --broadcast-mode sync \
  --output json \
  --yes
```

Once you have the IBC acknowledgement bytes (from relayer output or packet-ack query), decode action IDs:

```go
ids, err := cascade.ExtractRequestActionIDsFromAck(ackBytes)
```

If you already have `sdk.TxMsgData` (for example, from an ack you decoded yourself), use:

```go
ids := cascade.ExtractRequestActionIDsFromTxMsgData(msgData)
```

Packet/ack CLI helpers (controller chain):

```bash
# tx response -> packet identifiers
simd q tx <tx-hash> --output json
# packet ack query uses port/channel/sequence from send_packet event
simd q ibc channel packet-ack <port> <channel> <sequence> --output json
```

```go
txHash, _ := cascade.ParseTxHashJSON(txJSON)
packetInfo, _ := cascade.ExtractPacketInfoFromTxJSON(txQueryJSON)
ackBytes, _ := cascade.DecodePacketAcknowledgementJSON(ackQueryJSON)
ids, _ := cascade.ExtractRequestActionIDsFromAck(ackBytes)
_ = txHash
_ = packetInfo
```

### Crypto Helpers (pkg/crypto)

Common helpers:

- `DefaultKeyringParams` / `NewKeyring` for consistent keyring setup.
- `LoadKeyringFromMnemonic` / `ImportKeyFromMnemonic` for mnemonic-based flows.
- `AddressFromKey` to derive HRP-specific addresses without mutating global config.
- `NewDefaultTxConfig` and `SignTxWithKeyring` for signing with Cosmos SDK builders.

```go
import sdkcrypto "github.com/LumeraProtocol/sdk-go/pkg/crypto"

kr, _ := sdkcrypto.NewKeyring(sdkcrypto.DefaultKeyringParams())
addr, _ := sdkcrypto.AddressFromKey(kr, "alice", "lumera")
_ = addr
```

### Multi-Account Usage

Reuse the same configuration and transports for multiple local accounts via the client factory:

```go
import (
    "context"

    "github.com/cosmos/cosmos-sdk/crypto/keyring"
    lumerasdk "github.com/LumeraProtocol/sdk-go/client"
    sdkcrypto "github.com/LumeraProtocol/sdk-go/pkg/crypto"
)

kr, _ := keyring.New("lumera", "os", "~/.lumera", nil)
factory, err := lumerasdk.NewFactory(lumerasdk.Config{
    ChainID:      "lumera-testnet-2",
    GRPCEndpoint: "localhost:9090",
    RPCEndpoint:  "http://localhost:26657",
}, kr)

aliceAddr, _ := sdkcrypto.AddressFromKey(kr, "alice", "lumera")
bobAddr, _ := sdkcrypto.AddressFromKey(kr, "bob", "lumera")

alice, _ := factory.WithSigner(ctx, aliceAddr, "alice")
bob, _ := factory.WithSigner(ctx, bobAddr, "bob")
defer alice.Close()
defer bob.Close()

// Upload or query with different signers using the same underlying connections
_, _ = alice.Blockchain.Action.GetAction(ctx, "some-action-id")
_, _ = bob.Blockchain.Action.GetAction(ctx, "another-action-id")
```

See [examples/multi-account](examples/multi-account) for a runnable sample.

## Examples

See the [examples](./examples) directory for complete working examples:

- [Cascade Upload](./examples/cascade-upload) - Upload files to decentralized storage
- [Cascade Download](./examples/cascade-download) - Download files from storage
- [Query Actions](./examples/query-actions) - Query blockchain actions
- [Claim Tokens](./examples/claim-tokens) - Claim tokens from old chain
- [Multi-Account Factory](./examples/multi-account) - Reuse a config while swapping local signers

## Documentation

- [Developer Guide & Tutorials](docs/DEVELOPER_GUIDE.md)
- [API Overview](docs/API.md)
- [API Documentation](https://pkg.go.dev/github.com/LumeraProtocol/sdk-go)
- [Lumera Documentation](https://docs.lumera.io)

## Development

```bash
# Clone the repository
git clone https://github.com/LumeraProtocol/sdk-go.git
cd sdk-go

# Install dependencies
go mod download

# Run tests
make test

# Run linters
make lint

# Build examples
make examples
```

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details.

## License

Apache 2.0 - see [LICENSE](LICENSE) file for details.

## Links

- [Lumera Protocol](https://lumera.io)
- [Documentation](https://docs.lumera.io)
- [Discord](https://discord.gg/lumera)
- [Twitter](https://twitter.com/LumeraProtocol)
