package client

import (
	"time"

	clientconfig "github.com/LumeraProtocol/sdk-go/client/config"
	sdklog "github.com/LumeraProtocol/sdk-go/pkg/log"
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

// WithLogger enables diagnostic logging using the provided logger.
func WithLogger(logger sdklog.Logger) Option {
	return func(c *Config) {
		c.Logger = logger
	}
}
