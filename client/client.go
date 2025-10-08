package client

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"

	"github.com/LumeraProtocol/sdk-go/blockchain"
	"github.com/LumeraProtocol/sdk-go/cascade"
)

// Client provides unified access to Lumera blockchain and storage
type Client struct {
	// High-level modules
	Blockchain *blockchain.Client
	Cascade    *cascade.Client

	// Configuration
	config  *Config
	keyring keyring.Keyring
}

// New creates a new unified Lumera client
func New(ctx context.Context, cfg Config, kr keyring.Keyring, opts ...Option) (*Client, error) {
	// Apply options
	for _, opt := range opts {
		opt(&cfg)
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Initialize blockchain client
	blockchainClient, err := blockchain.New(ctx, blockchain.Config{
		ChainID:        cfg.ChainID,
		GRPCAddr:       cfg.GRPCAddr,
		Timeout:        cfg.BlockchainTimeout,
		MaxRecvMsgSize: cfg.MaxRecvMsgSize,
		MaxSendMsgSize: cfg.MaxSendMsgSize,
	}, kr, cfg.KeyName)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize blockchain client: %w", err)
	}

	// Initialize cascade client (wraps SuperNode SDK)
	cascadeClient, err := cascade.New(ctx, cascade.Config{
		ChainID:  cfg.ChainID,
		GRPCAddr: cfg.GRPCAddr,
		Address:  cfg.Address,
		KeyName:  cfg.KeyName,
		Timeout:  cfg.StorageTimeout,
	}, kr)
	if err != nil {
		blockchainClient.Close()
		return nil, fmt.Errorf("failed to initialize cascade client: %w", err)
	}

	return &Client{
		Blockchain: blockchainClient,
		Cascade:    cascadeClient,
		config:     &cfg,
		keyring:    kr,
	}, nil
}

// Close releases all resources
func (c *Client) Close() error {
	var errs []error

	if c.Blockchain != nil {
		if err := c.Blockchain.Close(); err != nil {
			errs = append(errs, fmt.Errorf("blockchain close: %w", err))
		}
	}

	if c.Cascade != nil {
		if err := c.Cascade.Close(); err != nil {
			errs = append(errs, fmt.Errorf("cascade close: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}

	return nil
}

// Config returns the client configuration
func (c *Client) Config() Config {
	return *c.config
}
