package base

import (
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
}
