package cascade

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/LumeraProtocol/lumera/x/lumeraid/securekeyx"
	snsdk "github.com/LumeraProtocol/supernode/v2/sdk/action"
	snconfig "github.com/LumeraProtocol/supernode/v2/sdk/config"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	sdkEvent "github.com/LumeraProtocol/sdk-go/cascade/event"
)

// Config for a cascade client
type Config struct {
	ChainID  string
	GRPCAddr string
	Address  string
	KeyName  string
	// ICAOwnerKeyName is the local key name for the ICA controller owner.
	ICAOwnerKeyName string
	// ICAOwnerHRP is the bech32 prefix for the ICA controller owner chain.
	ICAOwnerHRP string
	Timeout     time.Duration
	// LogLevel controls SDK logging (debug, info, warn, error). Default is error.
	LogLevel string
}

// Client provides access to cascade operations (wraps SuperNode SDK)
type Client struct {
	snClient snsdk.Client
	tasks    *TaskManager
	config   Config
	keyring  keyring.Keyring
	logger   *zap.Logger
	snLogger *supernodeLogger

	subMu        sync.RWMutex
	localSubs    map[sdkEvent.EventType][]sdkEvent.Handler
	localSubsAll []sdkEvent.Handler
}


// New creates a new cascade client
func New(ctx context.Context, cfg Config, kr keyring.Keyring) (*Client, error) {

	// Create SuperNode SDK config
	accountCfg := snconfig.AccountConfig{
		KeyName:         cfg.KeyName,
		Keyring:         kr,
		PeerType:        securekeyx.Simplenode,
		ICAOwnerKeyName: cfg.ICAOwnerKeyName,
		ICAOwnerHRP:     cfg.ICAOwnerHRP,
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

	logger, err := newLogger(cfg.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	// Create SuperNode SDK client with adapted logger
	snLogger := newSupernodeLogger(logger)
	snClient, err := snsdk.NewClient(ctx, sdkConfig, snLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create supernode client (grpc=%s, chain_id=%s): %w", cfg.GRPCAddr, cfg.ChainID, err)
	}

	// Create a task manager
	taskMgr := NewTaskManager(snClient)

	return &Client{
		snClient:  snClient, // store single-level pointer
		tasks:     taskMgr,
		config:    cfg,
		keyring:   kr,
		logger:    logger,
		snLogger:  snLogger,
		localSubs: make(map[sdkEvent.EventType][]sdkEvent.Handler),
	}, nil
}

// SetLogger configures optional diagnostics logging.
func (c *Client) SetLogger(logger *zap.Logger) {
	c.logger = logger
	if c.snLogger != nil {
		c.snLogger.SetLogger(logger)
	}
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

func newLogger(level string) (*zap.Logger, error) {
	normalized := strings.ToLower(strings.TrimSpace(level))
	if normalized == "" {
		normalized = "error"
	}
	var parsed zapcore.Level
	if err := parsed.Set(normalized); err != nil {
		return nil, fmt.Errorf("log level must be one of: debug, info, warn, error")
	}
	if parsed > zapcore.ErrorLevel {
		return nil, fmt.Errorf("log level must be one of: debug, info, warn, error")
	}
	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		MessageKey:     "msg",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeTime:     encodeSlashMillisTime,
		EncodeLevel:    encodeBracketLevel,
		EncodeDuration: zapcore.StringDurationEncoder,
	}
	encoderCfg.ConsoleSeparator = " "
	encoder := zapcore.NewConsoleEncoder(encoderCfg)
	core := zapcore.NewCore(encoder, zapcore.Lock(os.Stderr), parsed)
	return zap.New(core), nil
}

func encodeBracketLevel(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + l.CapitalString() + "]")
}

func encodeSlashMillisTime(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006/01/02 15:04:05.000"))
}
