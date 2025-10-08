package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	lumerasdk "github.com/LumeraProtocol/sdk-go/client"
	sdkcrypto "github.com/LumeraProtocol/sdk-go/internal/crypto"
)

func expandPath(p string) string {
	if p == "" {
		return p
	}
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			if p == "~" {
				return home
			}
			if strings.HasPrefix(p, "~/") {
				return filepath.Join(home, p[2:])
			}
		}
	}
	return p
}

func adjustKeyringDir(base, backend string) string {
	if backend == "file" || backend == "test" {
		return filepath.Join(base, "keyring-"+backend)
	}
	return base
}

func main() {
	ctx := context.Background()

	grpcEndpoint := flag.String("grpc-endpoint", "localhost:9090", "Lumera gRPC endpoint")
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

	baseDir := expandPath(*keyringDir)
	actualDir := adjustKeyringDir(baseDir, *keyringBackend)
	params := sdkcrypto.KeyringParams{
		AppName: "lumera",
		Backend: *keyringBackend,
		Dir:     actualDir,
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