// erc20-convert exercises Lumera's ERC20 conversion flow in both directions.
// Direction is controlled by --mode:
//
//   coin-to-erc20: wraps ulume coins as ERC20 tokens at the configured 0x receiver
//   erc20-to-coin: unwraps ERC20 tokens back to ulume at the configured bech32 receiver
//
// The signing key must be eth_secp256k1 so the 0x sender derivation matches
// Lumera's account semantics.
package main

import (
	"context"
	"flag"
	"log"
	"math/big"

	sdkmath "cosmossdk.io/math"
	lumerasdk "github.com/LumeraProtocol/sdk-go/client"
	sdkcrypto "github.com/LumeraProtocol/sdk-go/pkg/crypto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

func main() {
	ctx := context.Background()

	grpcEndpoint := flag.String("grpc-endpoint", "localhost:9090", "Lumera gRPC endpoint")
	rpcEndpoint := flag.String("rpc-endpoint", "http://localhost:26657", "Lumera RPC endpoint")
	chainID := flag.String("chain-id", "lumera-testnet-2", "Cosmos chain ID")
	evmChainID := flag.Int64("evm-chain-id", 1414, "EIP-155 chain ID")
	keyringBackend := flag.String("keyring-backend", "os", "Keyring backend: os|file|test")
	keyringDir := flag.String("keyring-dir", "~/.lumera", "Keyring base directory")
	keyName := flag.String("key-name", "alice", "Key name (eth_secp256k1)")
	addrBech32 := flag.String("address", "", "Bech32 address paired with the key")
	mode := flag.String("mode", "coin-to-erc20", "coin-to-erc20 | erc20-to-coin")
	amount := flag.Int64("amount", 1, "amount (ulume for coin-to-erc20, raw integer for erc20-to-coin)")
	contract := flag.String("contract", "", "ERC20 contract address (required for erc20-to-coin)")
	to := flag.String("to", "", "0x receiver (coin-to-erc20) or bech32 receiver (erc20-to-coin)")
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

	switch *mode {
	case "coin-to-erc20":
		if *to == "" {
			log.Fatalf("--to (0x receiver) is required")
		}
		coin := sdk.NewCoin("ulume", sdkmath.NewInt(*amount))
		recv := common.HexToAddress(*to)
		res, err := client.Blockchain.ConvertCoinToERC20(ctx, coin, recv, "")
		if err != nil {
			log.Fatalf("ConvertCoinToERC20: %v", err)
		}
		log.Printf("wrapped %s ulume -> %s", res.Amount, res.To)
		log.Printf("tx %s height %d", res.TxHash, res.Height)

	case "erc20-to-coin":
		if *contract == "" || *to == "" {
			log.Fatalf("--contract and --to (bech32 receiver) are required")
		}
		res, err := client.Blockchain.ConvertERC20ToCoin(ctx,
			common.HexToAddress(*contract),
			sdkmath.NewInt(*amount),
			*to,
			"",
		)
		if err != nil {
			log.Fatalf("ConvertERC20ToCoin: %v", err)
		}
		log.Printf("unwrapped %s from %s -> %s", res.Amount, res.From, res.To)
		log.Printf("tx %s height %d", res.TxHash, res.Height)

	default:
		log.Fatalf("unknown --mode %q (expected coin-to-erc20 or erc20-to-coin)", *mode)
	}

	// Optional: print the receiver's ERC20 balance after coin-to-erc20.
	if *mode == "coin-to-erc20" && *contract != "" {
		holder := common.HexToAddress(*to)
		bal, err := client.Blockchain.ERC20.Erc20Balance(ctx, common.HexToAddress(*contract), holder)
		if err == nil {
			log.Printf("post-tx ERC20 balance for %s: %s", holder.Hex(), bal)
		}
	}
}
