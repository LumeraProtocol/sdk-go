// evm-balance prints EVM-side state for an address: balance (alume), nonce,
// the current feemarket BaseFee and MinGasPrice, plus the precisebank
// fractional balance. Read-only — no signing or broadcasting.
//
// Usage:
//
//	go run ./examples/evm-balance \
//	    --grpc-endpoint localhost:9090 \
//	    --address 0xAbC...
//
// If --address is omitted the example reads the keyring entry and derives
// the 0x address from it.
package main

import (
	"context"
	"flag"
	"log"

	lumerasdk "github.com/LumeraProtocol/sdk-go/client"
	sdkcrypto "github.com/LumeraProtocol/sdk-go/pkg/crypto"
	"github.com/ethereum/go-ethereum/common"
)

func main() {
	ctx := context.Background()

	grpcEndpoint := flag.String("grpc-endpoint", "localhost:9090", "Lumera gRPC endpoint")
	rpcEndpoint := flag.String("rpc-endpoint", "http://localhost:26657", "Lumera RPC endpoint")
	chainID := flag.String("chain-id", "lumera-testnet-2", "Cosmos chain ID")
	keyringBackend := flag.String("keyring-backend", "os", "Keyring backend: os|file|test")
	keyringDir := flag.String("keyring-dir", "~/.lumera", "Keyring base directory")
	keyName := flag.String("key-name", "alice", "Key name (eth_secp256k1) — used only if --address is empty")
	addrFlag := flag.String("address", "", "0x address to inspect; defaults to the keyring entry")
	flag.Parse()

	kr, err := sdkcrypto.NewKeyring(sdkcrypto.KeyringParams{
		AppName: "lumera",
		Backend: *keyringBackend,
		Dir:     *keyringDir,
	})
	if err != nil {
		log.Fatalf("create keyring: %v", err)
	}

	addr := *addrFlag
	if addr == "" {
		addr, err = sdkcrypto.EVMAddressFromKey(kr, *keyName)
		if err != nil {
			log.Fatalf("derive address from key %q: %v", *keyName, err)
		}
	}
	evmAddr := common.HexToAddress(addr)
	log.Printf("inspecting %s", evmAddr.Hex())

	client, err := lumerasdk.New(ctx, lumerasdk.Config{
		ChainID:      *chainID,
		GRPCEndpoint: *grpcEndpoint,
		RPCEndpoint:  *rpcEndpoint,
		Address:      "lumera1placeholder", // not used for read-only flow
		KeyName:      *keyName,
	}, kr)
	if err != nil {
		log.Fatalf("create client: %v", err)
	}
	defer client.Close()

	bc := client.Blockchain

	balance, err := bc.EVM.Balance(ctx, evmAddr)
	if err != nil {
		log.Fatalf("Balance: %v", err)
	}
	log.Printf("balance (alume):       %s", balance)

	if ulume, frac := sdkcrypto.ULume(balance); frac.Sign() == 0 {
		log.Printf("balance (ulume):       %s", ulume)
	} else {
		log.Printf("balance (ulume):       %s + %s alume fractional", ulume, frac)
	}

	acct, err := bc.EVM.EthAccount(ctx, evmAddr)
	if err != nil {
		log.Fatalf("EthAccount: %v", err)
	}
	log.Printf("nonce / sequence:      %d", acct.Nonce)
	log.Printf("code hash:             %s", acct.CodeHash)

	baseFee, err := bc.EVM.BaseFee(ctx)
	if err != nil {
		log.Fatalf("EVM BaseFee: %v", err)
	}
	log.Printf("EVM base fee (alume):  %s", baseFee)

	feeMarketBase, err := bc.FeeMarket.BaseFee(ctx)
	if err != nil {
		log.Fatalf("FeeMarket BaseFee: %v", err)
	}
	log.Printf("base fee (ulume):      %s", feeMarketBase)

	params, err := bc.FeeMarket.Params(ctx)
	if err != nil {
		log.Fatalf("FeeMarket Params: %v", err)
	}
	log.Printf("min gas price (ulume): %s", params.Params.MinGasPrice)

	// Optional precisebank fractional balance (lookup by bech32 address).
	bech32, err := sdkcrypto.EVMToBech32(evmAddr, "lumera")
	if err == nil {
		frac, err := bc.PreciseBank.FractionalBalance(ctx, bech32)
		if err != nil {
			log.Printf("FractionalBalance: %v", err)
		} else {
			log.Printf("fractional balance:    %s", frac)
		}
	}
}
