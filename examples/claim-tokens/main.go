package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	lumerasdk "github.com/LumeraProtocol/sdk-go/client"
	"github.com/LumeraProtocol/sdk-go/constants"
	sdkcrypto "github.com/LumeraProtocol/sdk-go/pkg/crypto"
)

func main() {
	ctx := context.Background()

	grpcEndpoint := flag.String("grpc-endpoint", "localhost:9090", "Lumera gRPC endpoint")
	rpcEndpoint := flag.String("rpc-endpoint", "http://localhost:26657", "Lumera RPC endpoint")
	chainID := flag.String("chain-id", "lumera-testnet-2", "Chain ID")
	keyringBackend := flag.String("keyring-backend", "os", "Keyring backend: os|file|test")
	keyringDir := flag.String("keyring-dir", "~/.lumera", "Keyring base directory (actual dir appends keyring-<backend> for file/test)")
	keyName := flag.String("key-name", "my-key", "Key name in the keyring")
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

	address, err := sdkcrypto.AddressFromKey(kr, *keyName, constants.LumeraAccountHRP)
	if err != nil {
		log.Fatalf("derive owner address: %v\n", err)
	}

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

	fmt.Println("Claim tokens example - to be implemented")
	fmt.Println("This example will demonstrate claiming tokens from the old chain")
	// TODO: Implement claim token logic once claim module methods are available.
}
