package main

import (
	"context"
	"fmt"
	"log"
	"os"

	lumerasdk "github.com/LumeraProtocol/sdk-go/client"
	"github.com/LumeraProtocol/sdk-go/storage"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
)

func main() {
	ctx := context.Background()

	// Initialize keyring
	kr, err := keyring.New("lumera", "test", os.TempDir(), nil)
	if err != nil {
		log.Fatalf("Failed to create keyring: %v", err)
	}

	// Create unified client
	client, err := lumerasdk.New(ctx, lumerasdk.Config{
		ChainID:  "lumera-testnet-2",
		GRPCAddr: "localhost:9090",
		Address:  "lumera1abc...", // Your address
		KeyName:  "my-key",
	}, kr)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Upload file
	fmt.Println("Uploading file...")
	result, err := client.Storage.Upload(ctx, "/path/to/file.txt", "action-123",
		storage.WithPublic(true),
		storage.WithFileName("my-file.txt"),
	)
	if err != nil {
		log.Fatalf("Upload failed: %v", err)
	}

	fmt.Printf("Upload successful!\n")
	fmt.Printf("Action ID: %s\n", result.ActionID)
	fmt.Printf("Task ID: %s\n", result.TaskID)
	fmt.Printf("File Hash: %s\n", result.FileHash)
}
