package base

import (
	"math/big"
	"time"

	sdkmath "cosmossdk.io/math"

	clientconfig "github.com/LumeraProtocol/sdk-go/client/config"
)

// Config captures shared Cosmos SDK chain settings for gRPC + tx workflows.
type Config struct {
	ChainID        string
	GRPCAddr       string
	RPCEndpoint    string
	AccountHRP     string
	FeeDenom       string
	GasPrice       sdkmath.LegacyDec
	Timeout        time.Duration
	MaxRecvMsgSize int
	MaxSendMsgSize int
	InsecureGRPC   bool
	WaitTx         clientconfig.WaitTxConfig

	// EVMChainID is the EIP-155 chain ID for Ethereum-format transactions.
	// Distinct from the Cosmos ChainID. Nil disables EVM-tx helpers.
	EVMChainID *big.Int

	// EVMNativeDenom is the cosmos/evm `evm_denom` parameter — the bank denom
	// fees are deducted in (Lumera: "ulume"). Used by MsgEthereumTx.BuildTx
	// when constructing the cosmos fee coin from the inner Ethereum tx.
	EVMNativeDenom string

	// EVMExtendedDenom is the 18-decimal precisebank denom (Lumera: "alume").
	// Ethereum tx value, fee caps, balances, and receipts use this denom's
	// integer "wei-like" representation.
	EVMExtendedDenom string

	// EVMGasTipCap and EVMGasFeeCap are optional defaults (in the extended
	// denom's integer unit) for EIP-1559 gas pricing. Nil means "fetch from
	// chain state".
	EVMGasTipCap *big.Int
	EVMGasFeeCap *big.Int
}
