package config

import (
	"fmt"
	"time"

	sdklog "github.com/LumeraProtocol/sdk-go/pkg/log"
)

// Config holds all configuration for the Lumera client.
type Config struct {
	// Blockchain connection
	ChainID      string
	GRPCEndpoint string // Lumera blockchain gRPC endpoint
	RPCEndpoint  string // Tendermint RPC endpoint for websocket subscriptions

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

	// WaitTx controls transaction confirmation behaviour.
	WaitTx WaitTxConfig

	// Logger is optional; when set, SDK operations emit diagnostics.
	Logger sdklog.Logger
}

// WaitTxConfig configures how the SDK waits for transaction inclusion.
type WaitTxConfig struct {
	// SubscriberSetupTimeout defines how long we wait for the websocket subscription to become ready.
	SubscriberSetupTimeout time.Duration

	// Polling is a fallback mechanism when a websocket subscription is not available.
	// PollInterval controls how frequently the fallback poller queries gRPC for the tx.
	PollInterval time.Duration
	// PollMaxRetries limits the number of poll attempts before failing (0 => unlimited until ctx deadline).
	PollMaxRetries int
	// PollBackoffMultiplier > 1 enables exponential growth for poll intervals.
	PollBackoffMultiplier float64
	// PollBackoffMaxInterval caps the exponential backoff delay (0 => unlimited).
	PollBackoffMaxInterval time.Duration
	// PollBackoffJitter randomizes delays (0..1) to avoid synced retries.
	PollBackoffJitter float64
}

// Validate checks if the configuration is valid and populates defaults.
func (c *Config) Validate() error {
	if c.ChainID == "" {
		return fmt.Errorf("chain_id is required")
	}
	if c.GRPCEndpoint == "" {
		return fmt.Errorf("grpc_addr is required")
	}
	if c.RPCEndpoint == "" {
		return fmt.Errorf("rpc_addr is required")
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
	ApplyWaitTxDefaults(&c.WaitTx)

	return nil
}

// Default returns a configuration with sensible defaults for testnet.
func Default() Config {
	return Config{
		ChainID:           "lumera-testnet-2",
		GRPCEndpoint:      "localhost:9090",
		RPCEndpoint:       "http://localhost:26657",
		BlockchainTimeout: 10 * time.Second,
		StorageTimeout:    5 * time.Minute,
		MaxRetries:        3,
		MaxRecvMsgSize:    1024 * 1024 * 50,
		MaxSendMsgSize:    1024 * 1024 * 50,
		WaitTx:            DefaultWaitTxConfig(),
	}
}

// DefaultWaitTxConfig returns recommended defaults for wait-tx behaviour.
func DefaultWaitTxConfig() WaitTxConfig {
	return WaitTxConfig{
		SubscriberSetupTimeout: 5 * time.Second,
		PollInterval:           500 * time.Millisecond,
		PollMaxRetries:         40,
		PollBackoffMultiplier:  1.5,
		PollBackoffMaxInterval: 20 * time.Second,
		PollBackoffJitter:      0,
	}
}

// ApplyWaitTxDefaults normalizes zero or negative values using defaults.
func ApplyWaitTxDefaults(cfg *WaitTxConfig) {
	if cfg == nil {
		return
	}
	def := DefaultWaitTxConfig()
	
	if cfg.SubscriberSetupTimeout <= 0 {
		cfg.SubscriberSetupTimeout = def.SubscriberSetupTimeout
	}

	if cfg.PollInterval <= 0 {
		cfg.PollInterval = def.PollInterval
	}
	if cfg.PollBackoffMultiplier <= 0 {
		cfg.PollBackoffMultiplier = def.PollBackoffMultiplier
	}
	if cfg.PollBackoffMaxInterval <= 0 {
		cfg.PollBackoffMaxInterval = def.PollBackoffMaxInterval
	}
	if cfg.PollBackoffJitter < 0 {
		cfg.PollBackoffJitter = 0
	}
}
