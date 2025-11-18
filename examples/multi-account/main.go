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
	keyName1 := flag.String("key-name-1", "bob", "Key name for account1")
	keyName2 := flag.String("key-name-2", "alice", "Key name for account2")
	actionID1 := flag.String("action-id-1", "", "Optional action ID to fetch with account1")
	actionID2 := flag.String("action-id-2", "", "Optional action ID to fetch with account2")
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

	account1Addr, err := sdkcrypto.AddressFromKey(kr, *keyName1, "lumera")
	if err != nil {
		log.Fatalf("Failed to derive account1 address (%s): %v", *keyName1, err)
	}
	account2Addr, err := sdkcrypto.AddressFromKey(kr, *keyName2, "lumera")
	if err != nil {
		log.Fatalf("Failed to derive account2 address (%s): %v", *keyName2, err)
	}

	account1, err := factory.WithSigner(ctx, account1Addr, *keyName1)
	if err != nil {
		log.Fatalf("Failed to create account1 client: %v", err)
	}
	defer account1.Close() //nolint:errcheck

	account2, err := factory.WithSigner(ctx, account2Addr, *keyName2)
	if err != nil {
		log.Fatalf("Failed to create account2 client: %v", err)
	}
	defer account2.Close() //nolint:errcheck

	fmt.Println("Querying sample actions for account1 and account2...")
	if id := *actionID1; id != "" {
		if _, err := account1.Blockchain.Action.GetAction(ctx, id); err != nil {
			fmt.Printf("account1 failed to fetch action %q (expected if it doesn't exist yet): %v\n", id, err)
		} else {
			fmt.Printf("account1 successfully fetched action %q\n", id)
		}
	} else {
		fmt.Println("account1 action ID not provided; skipping query.")
	}

	if id := *actionID2; id != "" {
		if _, err := account2.Blockchain.Action.GetAction(ctx, id); err != nil {
			fmt.Printf("account2 failed to fetch action %q (expected if it doesn't exist yet): %v\n", id, err)
		} else {
			fmt.Printf("account2 successfully fetched action %q\n", id)
		}
	} else {
		fmt.Println("account2 action ID not provided; skipping query.")
	}
}
