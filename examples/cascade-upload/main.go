package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/LumeraProtocol/sdk-go/cascade"
	lumerasdk "github.com/LumeraProtocol/sdk-go/client"
	sdkcrypto "github.com/LumeraProtocol/sdk-go/internal/crypto"
)

func main() {
	ctx := context.Background()

	grpcEndpoint := flag.String("grpc-endpoint", "localhost:9090", "Lumera gRPC endpoint")
	chainID := flag.String("chain-id", "lumera-testnet-2", "Chain ID")
	keyringBackend := flag.String("keyring-backend", "os", "Keyring backend: os|file|test")
	keyringDir := flag.String("keyring-dir", "~/.lumera", "Keyring base directory (actual dir appends keyring-<backend> for file/test)")
	keyName := flag.String("key-name", "my-key", "Key name in the keyring")
	address := flag.String("address", "lumera1abc...", "Your Lumera address")

	filePath := flag.String("file-path", "", "Path to file to upload (required)")
	public := flag.Bool("public", true, "Whether upload is public")
	upFileName := flag.String("file-name", "", "Optional filename override")
	flag.Parse()

	if strings.TrimSpace(*filePath) == "" {
		log.Fatal("file-path is required")
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

	client, err := lumerasdk.New(ctx, lumerasdk.Config{
		ChainID:  *chainID,
		GRPCAddr: *grpcEndpoint,
		Address:  *address,
		KeyName:  *keyName,
	}, kr)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close() //nolint:errcheck

	fmt.Println("Uploading file...")
	opts := []cascade.UploadOption{cascade.WithPublic(*public)}
	if fn := strings.TrimSpace(*upFileName); fn != "" {
		opts = append(opts, cascade.WithFileName(fn))
	}
	result, err := client.Cascade.Upload(ctx, *address, client.Blockchain, *filePath, opts...)
	if err != nil {
		log.Fatalf("Upload failed: %v", err)
	}

	fmt.Printf("Upload successful!\n")
	fmt.Printf("Action ID: %s\n", result.ActionID)
	fmt.Printf("Task ID: %s\n", result.TaskID)
}
