# Getting Started

## Prerequisites

- Go 1.26+ with module support.
- Access to Lumera endpoints: `grpc` (chain queries/tx), `rpc` (websocket for tx inclusion), and at least one SuperNode for Cascade uploads/downloads.
- A Cosmos keyring entry that can sign Lumera transactions (`github.com/cosmos/cosmos-sdk/crypto/keyring` is used throughout the SDK).

## Install

```bash
go get github.com/LumeraProtocol/sdk-go
```

## Configuration reference

`client.Config` (in `client/config`) drives both blockchain and Cascade clients:

- `ChainID`, `GRPCEndpoint`, `RPCEndpoint` – chain connection details. gRPC uses TLS automatically for non-local hosts/port 443.
- `Address`, `KeyName` – Cosmos account info in your keyring.
- `BlockchainTimeout`, `StorageTimeout` – default deadlines for chain and Cascade operations.
- `MaxRecvMsgSize`, `MaxSendMsgSize`, `MaxRetries` – transport tuning.
- `WaitTx` – controls websocket vs polling behaviour when waiting for tx inclusion (see defaults in `client/config`).
- `Logger` – optional; when set, SDK operations emit diagnostics.
- `LogLevel` – default logging threshold when no custom logger is supplied (default: error).
- EVM settings: `EVMChainID`, `EVMNativeDenom`, `EVMExtendedDenom`, `EVMGasTipCap`, `EVMGasFeeCap`. See [evm.md](evm.md).

Override fields with `client.With...` option helpers when calling `client.New`.

## Creating a client

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

## Multiple signers via the factory

`client.NewFactory` keeps a shared config/keyring and returns signer-specific clients:

```go
factory, _ := client.NewFactory(cfg, kr)
alice, _ := factory.WithSigner(ctx, "lumera1alice...", "alice")
bob, _ := factory.WithSigner(ctx, "lumera1bob...", "bob")
defer alice.Close()
defer bob.Close()
```

## Running the examples

- Run tests: `make test`
- Build samples: `make examples`
- Try a flow: `go run ./examples/cascade-upload`, `./examples/query-actions`, `./examples/multi-account`, `./examples/evm-transfer`, `./examples/erc20-convert`, `./examples/precompile-action --help`

## Troubleshooting

- **Tx inclusion timing out**: adjust `WaitTx` polling/backoff (see `client/config`). Ensure `RPCEndpoint` allows websocket subscriptions.
- **gRPC TLS errors**: remote hosts/port 443 default to TLS; for local nodes use `localhost:9090` or `127.0.0.1:9090`.
- **Key not found**: confirm the key name exists in the keyring path you passed to `keyring.New`.
- **SuperNode availability**: Cascade operations require reachable SuperNodes; watch `sdk:supernodes_unavailable` events for diagnostics.
- **EVM tx fails to encode**: confirm `EVMChainID` is set and the signing key is `eth_secp256k1`. The SDK rejects `MsgEthereumTx` in the cosmos signing pipeline — use `EVM.SendEthereumTransaction`.
