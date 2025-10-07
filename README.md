# Lumera Go SDK

Official Go SDK for the Lumera Protocol - a next-generation blockchain platform for AI and decentralized storage.

## Features

- 🔗 **Unified Client** - Single interface for blockchain and storage operations
- 📦 **Type-Safe** - Full Go type definitions for all Lumera modules
- 🚀 **High-Level API** - Simple methods for complex operations
- 🔐 **Secure** - Built on Cosmos SDK's proven cryptography
- 📝 **Well-Documented** - Comprehensive examples and documentation

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
    
    // Initialize keyring
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
    
    // Query an action
    action, err := client.Blockchain.Action.GetAction(ctx, "action-123")
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Action: %+v", action)
}
```

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

