package client

import (
	"time"

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

// WithGRPCAddr sets the gRPC address
func WithGRPCAddr(addr string) Option {
	return func(c *Config) {
		c.GRPCEndpoint = addr
	}
}

// WithRPCAddr sets the Tendermint RPC endpoint.
func WithRPCAddr(addr string) Option {
	return func(c *Config) {
		c.RPCEndpoint = addr
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

// WithLogger enables diagnostic logging using the provided logger.
func WithLogger(logger sdklog.Logger) Option {
	return func(c *Config) {
		c.Logger = logger
	}
}
