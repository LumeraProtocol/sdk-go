package main

import (
	"context"
	"fmt"
	"log"
	"os"

	lumerasdk "github.com/LumeraProtocol/sdk-go/client"
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

	fmt.Println("Claim tokens example - to be implemented")
	fmt.Println("This example will demonstrate claiming tokens from the old chain")

	// TODO: Implement claim token logic once claim module methods are available
	// Example:
	// result, err := client.Blockchain.Claim.ClaimTokens(ctx, ...)
}
