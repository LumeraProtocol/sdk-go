package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	lumerasdk "github.com/LumeraProtocol/sdk-go/client"
	"github.com/LumeraProtocol/sdk-go/blockchain"
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

	// Query a specific action
	fmt.Println("Querying action...")
	action, err := client.Blockchain.Action.GetAction(ctx, "action-123")
	if err != nil {
		log.Fatalf("Failed to get action: %v", err)
	}

	fmt.Printf("Action Details:\n")
	fmt.Printf("  ID: %s\n", action.ID)
	fmt.Printf("  Creator: %s\n", action.Creator)
	fmt.Printf("  Type: %s\n", action.Type)
	fmt.Printf("  State: %s\n", action.State)
	fmt.Printf("  Price: %s\n", action.Price)

	// List cascade actions
	fmt.Println("\nListing cascade actions...")
	actions, err := client.Blockchain.Action.ListActions(ctx,
		blockchain.WithActionType("CASCADE"),
		blockchain.WithPagination(10, 0),
	)
	if err != nil {
		log.Fatalf("Failed to list actions: %v", err)
	}

	fmt.Printf("Found %d cascade actions:\n", len(actions))
	for i, a := range actions {
		fmt.Printf("  %d. %s (%s)\n", i+1, a.ID, a.State)
	}
}

