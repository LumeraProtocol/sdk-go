package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	lumerasdk "github.com/LumeraProtocol/sdk-go/client"
	sdkcrypto "github.com/LumeraProtocol/sdk-go/internal/crypto"
)

func main() {
	ctx := context.Background()

	grpcEndpoint := flag.String("grpc-endpoint", "localhost:9090", "Lumera gRPC endpoint")
	rpcEndpoint := flag.String("rpc-endpoint", "http://localhost:26657", "Tendermint RPC endpoint")
	chainID := flag.String("chain-id", "lumera-testnet-2", "Chain ID")
	keyringBackend := flag.String("keyring-backend", "os", "Keyring backend: os|file|test")
	keyringDir := flag.String("keyring-dir", "~/.lumera", "Keyring base directory (actual dir appends keyring-<backend> for file/test)")
	keyName := flag.String("key-name", "my-key", "Key name in the keyring")
	address := flag.String("address", "lumera1abc...", "Your Lumera address")

	actionID := flag.String("action-id", "", "Action ID to download (required)")
	outputDir := flag.String("output-dir", "./output", "Output directory for download")
	flag.Parse()

	if strings.TrimSpace(*actionID) == "" {
		log.Fatal("action-id is required")
	}

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

	key, err := sdkcrypto.GetKey(kr, *keyName)
	if err != nil {
		log.Fatalf("Failed to get key: %v", err)
	}
	if key == nil {
		log.Fatalf("Key %s not found in keyring", *keyName)
	}

	client, err := lumerasdk.New(ctx, lumerasdk.Config{
		ChainID:      *chainID,
		GRPCEndpoint: *grpcEndpoint,
		RPCEndpoint:  *rpcEndpoint,
		Address:      *address,
		KeyName:      *keyName,
	}, kr)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close() //nolint:errcheck

	fmt.Println("Downloading file...")
	result, err := client.Cascade.Download(ctx, *actionID, *outputDir)
	if err != nil {
		log.Fatalf("Download failed: %v", err)
	}

	fmt.Printf("Download successful!\n")
	fmt.Printf("Action ID: %s\n", result.ActionID)
	fmt.Printf("Task ID: %s\n", result.TaskID)
	fmt.Printf("Output Path: %s\n", result.OutputPath)
}
