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
        ChainID:  "lumera-testnet-2",
        GRPCAddr: "localhost:9090",
        Address:  "lumera1abc...",
        KeyName:  "my-key",
    }, kr)
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

## Examples

See the [examples](./examples) directory for complete working examples:

- [Cascade Upload](./examples/cascade-upload) - Upload files to decentralized storage
- [Cascade Download](./examples/cascade-download) - Download files from storage
- [Query Actions](./examples/query-actions) - Query blockchain actions
- [Claim Tokens](./examples/claim-tokens) - Claim tokens from old chain

## Documentation

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