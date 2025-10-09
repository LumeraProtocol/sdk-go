package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/LumeraProtocol/sdk-go/blockchain"
	lumerasdk "github.com/LumeraProtocol/sdk-go/client"
	sdkcrypto "github.com/LumeraProtocol/sdk-go/internal/crypto"
	"github.com/LumeraProtocol/sdk-go/types"
)

func main() {
	ctx := context.Background()

	grpcEndpoint := flag.String("grpc-endpoint", "localhost:9090", "Lumera gRPC endpoint")
	chainID := flag.String("chain-id", "lumera-testnet-2", "Chain ID")
	keyringBackend := flag.String("keyring-backend", "os", "Keyring backend: os|file|test")
	keyringDir := flag.String("keyring-dir", "~/.lumera", "Keyring base directory (actual dir appends keyring-<backend> for file/test)")
	keyName := flag.String("key-name", "my-key", "Key name in the keyring")
	address := flag.String("address", "lumera1abc...", "Your Lumera address")

	actionID := flag.String("action-id", "", "Action ID to query (optional; if empty, list actions)")
	actionType := flag.String("action-type", "CASCADE", "Action type filter when listing")
	limit := flag.Uint("limit", 10, "Pagination limit when listing")
	offset := flag.Uint("offset", 0, "Pagination offset when listing")
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

	client, err := lumerasdk.New(ctx, lumerasdk.Config{
		ChainID:  *chainID,
		GRPCAddr: *grpcEndpoint,
		Address:  *address,
		KeyName:  *keyName,
	}, kr)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	if id := strings.TrimSpace(*actionID); id != "" {
		// Query a specific action
		fmt.Println("Querying action...")
		action, err := client.Blockchain.Action.GetAction(ctx, id)
		if err != nil {
			log.Fatalf("Failed to get action: %v", err)
		}
		fmt.Printf("Action Details:\n")
		fmt.Printf("  ID: %s\n", action.ID)
		fmt.Printf("  Creator: %s\n", action.Creator)
		fmt.Printf("  Type: %s\n", action.Type)
		fmt.Printf("  State: %s\n", action.State)
		fmt.Printf("  Price: %s\n", action.Price)

		if action.Metadata != nil {
			switch action.Metadata.Type() {
			case types.ActionTypeCascade:
				cascadeMeta := action.Metadata.(*types.CascadeMetadata)
				fmt.Println("FileName:", cascadeMeta.FileName)
				fmt.Println("Public:", cascadeMeta.Public)
			case types.ActionTypeSense:
				senseMeta := action.Metadata.(*types.SenseMetadata)
				fmt.Println("DataHash:", senseMeta.DataHash)
			}
		}
		return
	}

	// List actions with filters
	fmt.Println("Listing actions...")
	actions, err := client.Blockchain.Action.ListActions(ctx,
		blockchain.WithActionTypeStr(*actionType),
		blockchain.WithPagination(uint64(*limit), uint64(*offset)),
	)
	if err != nil {
		log.Fatalf("Failed to list actions: %v", err)
	}
	fmt.Printf("Found %d actions (type=%s):\n", len(actions), *actionType)
	for i, a := range actions {
		fmt.Printf("  %d. %s (%s)\n", i+1, a.ID, a.State)
	}
}
