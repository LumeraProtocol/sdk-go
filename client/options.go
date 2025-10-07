package client

import "time"

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
		c.GRPCAddr = addr
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

