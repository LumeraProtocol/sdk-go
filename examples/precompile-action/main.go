// precompile-action invokes Lumera's action precompile at 0x0901 through the
// SDK's generic PrecompileClient. Two modes:
//
//   get-params   reads x/action params via Call (read-only, no signature)
//   approve      sends approveAction(actionID) via Send (signed, broadcast)
//
// The published ABI lives at pkg/evm/precompiles/abi/action.json — the
// embedded copy here is the same artifact, parsed at SDK init.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"

	lumerasdk "github.com/LumeraProtocol/sdk-go/client"
	sdkcrypto "github.com/LumeraProtocol/sdk-go/pkg/crypto"
)

func main() {
	ctx := context.Background()

	grpcEndpoint := flag.String("grpc-endpoint", "localhost:9090", "Lumera gRPC endpoint")
	rpcEndpoint := flag.String("rpc-endpoint", "http://localhost:26657", "Lumera RPC endpoint")
	chainID := flag.String("chain-id", "lumera-testnet-2", "Cosmos chain ID")
	evmChainID := flag.Int64("evm-chain-id", 1414, "EIP-155 chain ID")
	keyringBackend := flag.String("keyring-backend", "os", "Keyring backend: os|file|test")
	keyringDir := flag.String("keyring-dir", "~/.lumera", "Keyring base directory")
	keyName := flag.String("key-name", "alice", "Key name (eth_secp256k1) for signed methods")
	addrBech32 := flag.String("address", "", "Bech32 address paired with the key (required for signed methods)")
	mode := flag.String("mode", "get-params", "get-params | approve")
	actionID := flag.String("action-id", "", "Action ID to approve (required for --mode=approve)")
	flag.Parse()

	kr, err := sdkcrypto.NewKeyring(sdkcrypto.KeyringParams{
		AppName: "lumera",
		Backend: *keyringBackend,
		Dir:     *keyringDir,
	})
	if err != nil {
		log.Fatalf("create keyring: %v", err)
	}

	client, err := lumerasdk.New(ctx, lumerasdk.Config{
		ChainID:      *chainID,
		GRPCEndpoint: *grpcEndpoint,
		RPCEndpoint:  *rpcEndpoint,
		Address:      *addrBech32,
		KeyName:      *keyName,
	}, kr,
		lumerasdk.WithEVMChainID(big.NewInt(*evmChainID)),
		lumerasdk.WithEVMNativeDenom("ulume"),
		lumerasdk.WithEVMExtendedDenom("alume"),
	)
	if err != nil {
		log.Fatalf("create client: %v", err)
	}
	defer client.Close()

	action := client.Blockchain.EVM.Action
	log.Printf("precompile address: %s", action.Address().Hex())

	switch *mode {
	case "get-params":
		out, err := action.Call(ctx, "getParams")
		if err != nil {
			log.Fatalf("Call getParams: %v", err)
		}
		log.Printf("getParams returned %d values:", len(out))
		for i, v := range out {
			log.Printf("  [%d] %s", i, fmt.Sprintf("%+v", v))
		}

	case "approve":
		if *actionID == "" {
			log.Fatalf("--action-id is required for --mode=approve")
		}
		res, err := action.Send(ctx, "approveAction", nil, *actionID)
		if err != nil {
			log.Fatalf("Send approveAction: %v", err)
		}
		log.Printf("eth tx:        %s", res.EthTxHash.Hex())
		log.Printf("cosmos tx:     %s", res.CosmosHash)
		log.Printf("height:        %d", res.Height)
		log.Printf("gas used:      %d", res.GasUsed)
		if res.VMError != "" {
			log.Fatalf("vm error: %s", res.VMError)
		}

	default:
		log.Fatalf("unknown --mode %q (expected get-params or approve)", *mode)
	}
}
