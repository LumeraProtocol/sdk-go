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
	sdkcrypto "github.com/LumeraProtocol/sdk-go/internal/crypto"
)

func main() {
	ctx := context.Background()

	grpcEndpoint := flag.String("grpc-endpoint", "localhost:9090", "Lumera gRPC endpoint")
	chainID := flag.String("chain-id", "lumera-testnet-2", "Chain ID")
	keyringBackend := flag.String("keyring-backend", "os", "Keyring backend: os|file|test")
	keyringDir := flag.String("keyring-dir", "~/.lumera", "Keyring base directory (actual dir appends keyring-<backend> for file/test)")
	keyName := flag.String("key-name", "my-key", "Key name in the keyring")

	filePath := flag.String("file-path", "", "Path to file to upload (required)")
	public := flag.Bool("public", true, "Whether upload is public")
	upFileName := flag.String("file-name", "", "Optional filename override")
	actionID := flag.String("action-id", "", "Existing action ID to upload bytes for (skips on-chain request)")
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

	address, err := sdkcrypto.AddressFromKey(kr, *keyName, "lumera")
	if err != nil {
		log.Fatalf("derive owner address: %v\n", err)
	}

	client, err := lumerasdk.New(ctx, lumerasdk.Config{
		ChainID:  *chainID,
		GRPCAddr: *grpcEndpoint,
		Address:  address,
		KeyName:  *keyName,
	}, kr)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close() //nolint:errcheck

	aid := strings.TrimSpace(*actionID)
	if aid != "" {
		fmt.Println("Uploading file bytes to SuperNodes for existing action...")
		taskID, err := client.Cascade.UploadToSupernode(ctx, aid, *filePath)
		if err != nil {
			log.Fatalf("UploadToSupernode failed: %v", err)
		}
		fmt.Printf("Upload successful!\n")
		fmt.Printf("Action ID: %s\n", aid)
		fmt.Printf("Task ID: %s\n", taskID)

		// Give the chain a moment, then check status
		time.Sleep(5 * time.Second)
		action, err := client.Blockchain.Action.GetAction(ctx, aid)
		if err != nil {
			log.Fatalf("Failed to get action: %v", err)
		}
		fmt.Printf("Action Status: %s\n", action.State)
		return
	}

	fmt.Println("Uploading file...")
	opts := []cascade.UploadOption{cascade.WithPublic(*public)}
	if fn := strings.TrimSpace(*upFileName); fn != "" {
		opts = append(opts, cascade.WithFileName(fn))
	}
	result, err := client.Cascade.Upload(ctx, address, client.Blockchain, *filePath, opts...)
	if err != nil {
		log.Fatalf("Upload failed: %v", err)
	}

	fmt.Printf("Upload successful!\n")
	fmt.Printf("Action ID: %s\n", result.ActionID)
	fmt.Printf("Task ID: %s\n", result.TaskID)

	//sleep 5 * time.Second
	time.Sleep(5 * time.Second)

	// Check status of the Action
	action, err := client.Blockchain.Action.GetAction(ctx, result.ActionID)
	if err != nil {
		log.Fatalf("Failed to get action: %v", err)
	}
	fmt.Printf("Action Status: %s\n", action.State)
}
