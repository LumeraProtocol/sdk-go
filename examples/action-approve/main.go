package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/LumeraProtocol/sdk-go/cascade"
	lumerasdk "github.com/LumeraProtocol/sdk-go/client"
	sdkcrypto "github.com/LumeraProtocol/sdk-go/pkg/crypto"
	"github.com/LumeraProtocol/sdk-go/types"
)

func main() {
	ctx := context.Background()

	// CLI flags
	actionID := flag.String("action-id", "", "Existing action ID to approve")
	grpcEndpoint := flag.String("grpc-endpoint", "localhost:9090", "Lumera gRPC endpoint")
	rpcEndpoint := flag.String("rpc-endpoint", "http://localhost:26657", "Lumera RPC endpoint")
	chainID := flag.String("chain-id", "lumera-testnet-2", "Chain ID")
	keyringBackend := flag.String("keyring-backend", "os", "Keyring backend: os|file|test")
	keyringDir := flag.String("keyring-dir", "~/.lumera", "Keyring base directory (actual dir appends keyring-<backend> for file/test)")
	keyName := flag.String("key-name", "my-key", "Key name in the keyring")
	flag.Parse()

	if strings.TrimSpace(*actionID) == "" {
		log.Fatalf("action-id is required")
	}

	// Initialize keyring
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

	address, err := sdkcrypto.AddressFromKey(kr, *keyName, "lumera")
	if err != nil {
		log.Fatalf("derive owner address: %v\n", err)
	}

	// Initialize unified Lumera client
	client, err := lumerasdk.New(ctx, lumerasdk.Config{
		ChainID:      *chainID,
		GRPCEndpoint: *grpcEndpoint,
		RPCEndpoint:  *rpcEndpoint,
		Address:      address,
		KeyName:      *keyName,
	}, kr)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close() //nolint:errcheck

	// Preflight: ensure action is already DONE before sending approval
	action, err := client.Blockchain.Action.GetAction(ctx, *actionID)
	if err != nil {
		log.Fatalf("failed to get action %s: %v", *actionID, err)
	}
	if action == nil {
		log.Fatalf("action %s not found", *actionID)
	}
	if action.State != types.ActionStateDone {
		log.Fatalf("action %s state is %s; expected %s. Refusing to send approve.", *actionID, action.State, types.ActionStateDone)
	}

	// Build approve message using package-level helper
	msg, err := cascade.CreateApproveActionMessage(ctx, *actionID,
		cascade.WithApproveCreator(address),
	)
	if err != nil {
		log.Fatalf("CreateApproveActionMessage failed: %v", err)
	}

	// Broadcast approve message
	txHash, err := cascade.SendApproveActionMessage(ctx, msg,
		cascade.WithApproveBlockchain(client.Blockchain),
	)
	if err != nil {
		log.Fatalf("SendApproveActionMessage failed: %v", err)
	}

	fmt.Printf("Approve sent successfully!\n")
	fmt.Printf("Action ID: %s\n", *actionID)
	fmt.Printf("Tx Hash: %s\n", txHash)

	// Small delay then query action status for visibility
	time.Sleep(3 * time.Second)
	a, err := client.Blockchain.Action.GetAction(ctx, *actionID)
	if err != nil {
		log.Fatalf("Failed to get action: %v", err)
	}
	fmt.Printf("Action Status: %s\n", a.State)
}
