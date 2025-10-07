package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	lumerasdk "github.com/LumeraProtocol/sdk-go/client"
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

	// Download file
	fmt.Println("Downloading file...")
	result, err := client.Storage.Download(ctx, "action-123", "./output")
	if err != nil {
		log.Fatalf("Download failed: %v", err)
	}

	fmt.Printf("Download successful!\n")
	fmt.Printf("Action ID: %s\n", result.ActionID)
	fmt.Printf("Task ID: %s\n", result.TaskID)
	fmt.Printf("Output Path: %s\n", result.OutputPath)
}

