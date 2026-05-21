package client

import (
	"math/big"
	"time"

	clientconfig "github.com/LumeraProtocol/sdk-go/client/config"
	"go.uber.org/zap"
)

// Option is a function that modifies Config
type Option func(*Config)

// WithChainID sets the chain ID
func WithChainID(chainID string) Option {
	return func(c *Config) {
		c.ChainID = chainID
	}
}

// WithKeyName sets the key name in the keyring.
func WithKeyName(name string) Option {
	return func(c *Config) {
		c.KeyName = name
	}
}

// WithGRPCEndpoint sets the gRPC address.
func WithGRPCEndpoint(endpoint string) Option {
	return func(c *Config) {
		c.GRPCEndpoint = endpoint
	}
}

// WithRPCEndpoint sets the CometBFT RPC endpoint.
func WithRPCEndpoint(endpoint string) Option {
	return func(c *Config) {
		c.RPCEndpoint = endpoint
	}
}

// WithBlockchainTimeout sets the blockchain timeout
func WithBlockchainTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.BlockchainTimeout = timeout
	}
}

// WithStorageTimeout sets the storage timeout
func WithStorageTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.StorageTimeout = timeout
	}
}

// WithMaxRetries sets the maximum number of retries
func WithMaxRetries(retries int) Option {
	return func(c *Config) {
		c.MaxRetries = retries
	}
}

// WithMaxMessageSize sets both send and receive message sizes
func WithMaxMessageSize(size int) Option {
	return func(c *Config) {
		c.MaxRecvMsgSize = size
		c.MaxSendMsgSize = size
	}
}

// WithWaitTxConfig overrides the wait-for-tx behavior.
func WithWaitTxConfig(waitCfg clientconfig.WaitTxConfig) Option {
	return func(c *Config) {
		c.WaitTx = waitCfg
	}
}

// WithLogLevel sets the SDK log level.
func WithLogLevel(level string) Option {
	return func(c *Config) {
		c.LogLevel = level
	}
}

// WithLogger enables diagnostic logging using the provided logger.
func WithLogger(logger *zap.Logger) Option {
	return func(c *Config) {
		c.Logger = logger
	}
}

// WithEVMChainID sets the EIP-155 chain ID used for Ethereum-format
// transactions. Distinct from the Cosmos ChainID. Required for EVM tx helpers.
func WithEVMChainID(id *big.Int) Option {
	return func(c *Config) {
		c.EVMChainID = id
	}
}

// WithEVMNativeDenom sets the cosmos/evm `evm_denom` (e.g. "ulume").
func WithEVMNativeDenom(denom string) Option {
	return func(c *Config) {
		c.EVMNativeDenom = denom
	}
}

// WithEVMExtendedDenom sets the 18-decimal precisebank denom (e.g. "alume").
func WithEVMExtendedDenom(denom string) Option {
	return func(c *Config) {
		c.EVMExtendedDenom = denom
	}
}

// WithEVMGasCaps sets optional defaults for EIP-1559 gas pricing in the
// extended denom's integer unit (alume/gas). Nil means "fetch from chain
// state at tx time".
func WithEVMGasCaps(tip, fee *big.Int) Option {
	return func(c *Config) {
		c.EVMGasTipCap = tip
		c.EVMGasFeeCap = fee
	}
}
