package cascade

import (
	"context"
	"fmt"
	"time"

	"github.com/LumeraProtocol/lumera/x/lumeraid/securekeyx"
	snsdk "github.com/LumeraProtocol/supernode/v2/sdk/action"
	snconfig "github.com/LumeraProtocol/supernode/v2/sdk/config"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
)

// Config for a cascade client
type Config struct {
	ChainID  string
	GRPCAddr string
	Address  string
	KeyName  string
	Timeout  time.Duration
}

// Client provides access to cascade operations (wraps SuperNode SDK)
type Client struct {
	snClient snsdk.Client
	tasks    *TaskManager
	config   Config
	keyring  keyring.Keyring
}

// New creates a new cascade client
func New(ctx context.Context, cfg Config, kr keyring.Keyring) (*Client, error) {
	// Create SuperNode SDK config
	accountCfg := snconfig.AccountConfig{
		LocalCosmosAddress: cfg.Address,
		KeyName:            cfg.KeyName,
		Keyring:            kr,
		PeerType:           securekeyx.Simplenode,
	}

	lumeraCfg := snconfig.LumeraConfig{
		GRPCAddr: cfg.GRPCAddr,
		ChainID:  cfg.ChainID,
	}

	sdkConfig := snconfig.NewConfig(accountCfg, lumeraCfg)

	// Validate config
	if err := sdkConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid supernode config: %w", err)
	}

	// Create SuperNode client (pass nil for logger to use default)
	snClient, err := snsdk.NewClient(ctx, sdkConfig, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create supernode client: %w", err)
	}

	// Create a task manager
	taskMgr := NewTaskManager(snClient)

	return &Client{
		snClient: snClient, // store single-level pointer
		tasks:    taskMgr,
		config:   cfg,
		keyring:  kr,
	}, nil
}

// Close closes the cascade client
func (c *Client) Close() error {
	// SuperNode SDK client doesn't have a Close method yet
	// Add if/when it's implemented
	return nil
}
