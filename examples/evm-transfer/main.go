// evm-transfer demonstrates sending an EIP-1559 transaction to Lumera using
// an eth_secp256k1 key in the SDK's keyring. It prints the eth tx hash, the
// cosmos tx hash, and the resolved gas/fee values pulled from chain state.
//
// Usage:
//
//	go run ./examples/evm-transfer \
//	    --grpc-endpoint localhost:9090 \
//	    --rpc-endpoint  http://localhost:26657 \
//	    --chain-id      lumera-testnet-2 \
//	    --evm-chain-id  1414 \
//	    --key-name      alice \
//	    --to            0x000000000000000000000000000000000000dEaD \
//	    --amount-ulume  1
package main

import (
	"context"
	"flag"
	"log"
	"math/big"

	sdkmath "cosmossdk.io/math"
	"github.com/LumeraProtocol/sdk-go/blockchain"
	lumerasdk "github.com/LumeraProtocol/sdk-go/client"
	sdkcrypto "github.com/LumeraProtocol/sdk-go/pkg/crypto"
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
	keyName := flag.String("key-name", "alice", "Key name (must be eth_secp256k1)")
	addrBech32 := flag.String("address", "", "Bech32 address paired with the key")
	to := flag.String("to", "0x000000000000000000000000000000000000dEaD", "Recipient 0x address")
	amountULume := flag.Int64("amount-ulume", 1, "Amount to transfer in ulume (will be converted to alume)")
	flag.Parse()

	kr, err := sdkcrypto.NewKeyring(sdkcrypto.KeyringParams{
		AppName: "lumera",
		Backend: *keyringBackend,
		Dir:     *keyringDir,
	})
	if err != nil {
		log.Fatalf("create keyring: %v", err)
	}

	from, err := sdkcrypto.EVMAddressFromKey(kr, *keyName)
	if err != nil {
		log.Fatalf("derive 0x sender (is %q an eth_secp256k1 key?): %v", *keyName, err)
	}
	log.Printf("sender 0x address: %s", from)

	cfg := lumerasdk.Config{
		ChainID:      *chainID,
		GRPCEndpoint: *grpcEndpoint,
		RPCEndpoint:  *rpcEndpoint,
		Address:      *addrBech32,
		KeyName:      *keyName,
	}
	client, err := lumerasdk.New(ctx, cfg, kr,
		lumerasdk.WithEVMChainID(big.NewInt(*evmChainID)),
		lumerasdk.WithEVMNativeDenom("ulume"),
		lumerasdk.WithEVMExtendedDenom("alume"),
	)
	if err != nil {
		log.Fatalf("create client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("close client: %v", err)
		}
	}()

	recipient := common.HexToAddress(*to)
	value := sdkcrypto.Wei(sdkmath.NewInt(*amountULume))
	log.Printf("sending %d ulume (%s alume) to %s", *amountULume, value, recipient.Hex())

	res, err := client.Blockchain.EVM.SendEthereumTransaction(ctx, &recipient, nil, &blockchain.EthereumTxOptions{
		Value: value,
	})
	if err != nil {
		log.Fatalf("send tx: %v", err)
	}

	log.Printf("eth tx hash:    %s", res.EthTxHash.Hex())
	log.Printf("cosmos tx hash: %s", res.CosmosHash)
	log.Printf("height:         %d", res.Height)
	log.Printf("gas used:       %d", res.GasUsed)
	if res.VMError != "" {
		log.Fatalf("vm error: %s", res.VMError)
	}
}
