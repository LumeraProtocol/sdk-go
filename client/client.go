package client

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"

	"github.com/LumeraProtocol/sdk-go/blockchain"
	"github.com/LumeraProtocol/sdk-go/cascade"
	sdklog "github.com/LumeraProtocol/sdk-go/pkg/log"
)

// Client provides unified access to Lumera blockchain and storage
type Client struct {
	// High-level modules
	Blockchain *blockchain.Client
	Cascade    *cascade.Client

	// Configuration
	config  *Config
	keyring keyring.Keyring
	logger  sdklog.Logger
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
		GRPCAddr:       cfg.GRPCEndpoint,
		RPCEndpoint:    cfg.RPCEndpoint,
		Timeout:        cfg.BlockchainTimeout,
		MaxRecvMsgSize: cfg.MaxRecvMsgSize,
		MaxSendMsgSize: cfg.MaxSendMsgSize,
		WaitTx:         cfg.WaitTx,
	}, kr, cfg.KeyName)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize blockchain client: %w", err)
	}

	// Initialize cascade client (wraps SuperNode SDK)
	cascadeClient, cascadeErr := cascade.New(ctx, cascade.Config{
		ChainID:  cfg.ChainID,
		GRPCAddr: cfg.GRPCEndpoint,
		Address:  cfg.Address,
		KeyName:  cfg.KeyName,
		Timeout:  cfg.StorageTimeout,
	}, kr)
	if cascadeErr != nil {
		if closeErr := blockchainClient.Close(); closeErr != nil {
			return nil, fmt.Errorf("failed to initialize cascade client: %w; also failed to close blockchain client: %v", cascadeErr, closeErr)
		}
		return nil, fmt.Errorf("failed to initialize cascade client: %w", cascadeErr)
	}
	cascadeClient.SetLogger(cfg.Logger)

	return &Client{
		Blockchain: blockchainClient,
		Cascade:    cascadeClient,
		config:     &cfg,
		keyring:    kr,
		logger:     cfg.Logger,
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
