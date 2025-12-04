package cascade

import (
	"context"
	"fmt"
	"sync"
	"time"
	"strings"

	"github.com/LumeraProtocol/lumera/x/lumeraid/securekeyx"
	snsdk "github.com/LumeraProtocol/supernode/v2/sdk/action"
	snconfig "github.com/LumeraProtocol/supernode/v2/sdk/config"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"

	sdkEvent "github.com/LumeraProtocol/sdk-go/cascade/event"
	sdklog "github.com/LumeraProtocol/sdk-go/pkg/log"
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
	logger   sdklog.Logger

	subMu        sync.RWMutex
	localSubs    map[sdkEvent.EventType][]sdkEvent.Handler
	localSubsAll []sdkEvent.Handler
}

// New creates a new cascade client
func New(ctx context.Context, cfg Config, kr keyring.Keyring) (*Client, error) {
	// Create SuperNode SDK config
	accountCfg := snconfig.AccountConfig{
		KeyName:  cfg.KeyName,
		Keyring:  kr,
		PeerType: securekeyx.Simplenode,
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
		snClient:  snClient, // store single-level pointer
		tasks:     taskMgr,
		config:    cfg,
		keyring:   kr,
		localSubs: make(map[sdkEvent.EventType][]sdkEvent.Handler),
	}, nil
}

// SetLogger configures optional diagnostics logging.
func (c *Client) SetLogger(logger sdklog.Logger) {
	c.logger = logger
}

// Close closes the cascade client
func (c *Client) Close() error {
	// SuperNode SDK client doesn't have a Close method yet
	// Add if/when it's implemented
	return nil
}

func (c *Client) isLocalEventType(t sdkEvent.EventType) bool {
	// EventType format: type:subtype
	return strings.HasPrefix(string(t), "sdk-go:")
}

func (c *Client) addLocalSubscriber(t sdkEvent.EventType, handler sdkEvent.Handler) {
	c.subMu.Lock()
	defer c.subMu.Unlock()
	c.localSubs[t] = append(c.localSubs[t], handler)
}

func (c *Client) addLocalAll(handler sdkEvent.Handler) {
	c.subMu.Lock()
	defer c.subMu.Unlock()
	c.localSubsAll = append(c.localSubsAll, handler)
}

func (c *Client) emitLocalEvent(ctx context.Context, evt sdkEvent.Event) {
	c.subMu.RLock()
	handlers := append([]sdkEvent.Handler{}, c.localSubs[evt.Type]...)
	all := append([]sdkEvent.Handler{}, c.localSubsAll...)
	c.subMu.RUnlock()

	for _, h := range handlers {
		h(ctx, evt)
	}
	for _, h := range all {
		h(ctx, evt)
	}
}
