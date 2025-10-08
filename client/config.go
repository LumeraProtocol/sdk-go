package client

import (
	"fmt"
	"time"
)

// Config holds all configuration for the Lumera client
type Config struct {
	// Blockchain connection
	ChainID  string
	GRPCAddr string // Lumera blockchain gRPC endpoint

	// Account settings
	Address string // Your cosmos address (lumera1...)
	KeyName string // Key name in keyring

	// Timeouts
	BlockchainTimeout time.Duration
	StorageTimeout    time.Duration

	// Optional overrides
	MaxRetries     int
	MaxRecvMsgSize int // Max message size for gRPC (default: 50MB)
	MaxSendMsgSize int
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.ChainID == "" {
		return fmt.Errorf("chain_id is required")
	}
	if c.GRPCAddr == "" {
		return fmt.Errorf("grpc_addr is required")
	}
	if c.Address == "" {
		return fmt.Errorf("address is required")
	}
	if c.KeyName == "" {
		return fmt.Errorf("key_name is required")
	}

	// Set defaults
	if c.BlockchainTimeout == 0 {
		c.BlockchainTimeout = 10 * time.Second
	}
	if c.StorageTimeout == 0 {
		c.StorageTimeout = 5 * time.Minute
	}
	if c.MaxRecvMsgSize == 0 {
		c.MaxRecvMsgSize = 1024 * 1024 * 50 // 50MB
	}
	if c.MaxSendMsgSize == 0 {
		c.MaxSendMsgSize = 1024 * 1024 * 50 // 50MB
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}

	return nil
}

// DefaultConfig returns a configuration with sensible defaults for testnet
func DefaultConfig() Config {
	return Config{
		ChainID:           "lumera-testnet-2",
		GRPCAddr:          "localhost:9090",
		BlockchainTimeout: 10 * time.Second,
		StorageTimeout:    5 * time.Minute,
		MaxRetries:        3,
		MaxRecvMsgSize:    1024 * 1024 * 50,
		MaxSendMsgSize:    1024 * 1024 * 50,
	}
}
