package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	lumerasdk "github.com/LumeraProtocol/sdk-go/client"
	sdkcrypto "github.com/LumeraProtocol/sdk-go/internal/crypto"
)

func main() {
	ctx := context.Background()

	grpcEndpoint := flag.String("grpc-endpoint", "localhost:9090", "Lumera gRPC endpoint")
	rpcEndpoint := flag.String("rpc-endpoint", "http://localhost:26657", "Tendermint RPC endpoint")
	chainID := flag.String("chain-id", "lumera-testnet-2", "Chain ID")
	keyringBackend := flag.String("keyring-backend", "os", "Keyring backend: os|file|test")
	keyringDir := flag.String("keyring-dir", "~/.lumera", "Keyring base directory")
	flag.Parse()

	params := sdkcrypto.KeyringParams{
		AppName: "lumera",
		Backend: *keyringBackend,
		Dir:     *keyringDir,
		Input:   nil,
	}
	kr, err := sdkcrypto.NewKeyring(params)
	if err != nil {
		log.Fatalf("Failed to create keyring: %v", err)
	}

	factory, err := lumerasdk.NewFactory(lumerasdk.Config{
		ChainID:      *chainID,
		GRPCEndpoint: *grpcEndpoint,
		RPCEndpoint:  *rpcEndpoint,
	}, kr)
	if err != nil {
		log.Fatalf("Failed to create client factory: %v", err)
	}

	alice, err := factory.WithSigner(ctx, "lumera1alice...", "alice")
	if err != nil {
		log.Fatalf("Failed to create Alice client: %v", err)
	}
	defer alice.Close() //nolint:errcheck

	bob, err := factory.WithSigner(ctx, "lumera1bob...", "bob")
	if err != nil {
		log.Fatalf("Failed to create Bob client: %v", err)
	}
	defer bob.Close() //nolint:errcheck

	fmt.Println("Querying sample actions for Alice and Bob...")
	if _, err := alice.Blockchain.Action.GetAction(ctx, "action-alice"); err != nil {
		fmt.Printf("Alice query error: %v\n", err)
	}
	if _, err := bob.Blockchain.Action.GetAction(ctx, "action-bob"); err != nil {
		fmt.Printf("Bob query error: %v\n", err)
	}
}
