package base

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	clientconfig "github.com/LumeraProtocol/sdk-go/client/config"
)

// Client provides common Cosmos SDK gRPC and tx helpers.
type Client struct {
	mu sync.RWMutex

	conn    *grpc.ClientConn
	config  Config
	keyring keyring.Keyring
	keyName string

	newConnFn      func(context.Context) (*grpc.ClientConn, error)
	reconnectHook  func(context.Context) error
	sleepHook      func(ctx context.Context, dMs int64) error
	jitterHook     func(maxExclusive int64) int64
}

// New creates a base blockchain client with a gRPC connection.
func New(ctx context.Context, cfg Config, kr keyring.Keyring, keyName string) (*Client, error) {
	clientconfig.ApplyWaitTxDefaults(&cfg.WaitTx)

	conn, err := newGRPCConn(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC: %w", err)
	}

	c := &Client{
		conn:    conn,
		config:  cfg,
		keyring: kr,
		keyName: keyName,
	}
	c.newConnFn = func(context.Context) (*grpc.ClientConn, error) {
		return newGRPCConn(cfg)
	}
	return c, nil
}

func newGRPCConn(cfg Config) (*grpc.ClientConn, error) {
	// Determine if we should use TLS based on the endpoint.
	// Use TLS if: port is 443, or hostname doesn't start with "localhost"/"127.0.0.1".
	useTLS := shouldUseTLS(cfg.GRPCAddr)
	if cfg.InsecureGRPC {
		useTLS = false
	}

	var creds credentials.TransportCredentials
	if useTLS {
		// Use system TLS credentials for secure connections
		creds = credentials.NewTLS(nil)
	} else {
		// Use insecure credentials for local development
		creds = insecure.NewCredentials()
	}

	// Create gRPC connection
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(cfg.MaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(cfg.MaxSendMsgSize),
		),
	}

	conn, err := grpc.NewClient(cfg.GRPCAddr, dialOpts...)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (c *Client) reconnect(ctx context.Context) error {
	if c.reconnectHook != nil {
		return c.reconnectHook(ctx)
	}
	if c.newConnFn == nil {
		return fmt.Errorf("newConnFn is not configured")
	}

	newConn, err := c.newConnFn(ctx)
	if err != nil {
		return fmt.Errorf("reconnect to gRPC: %w", err)
	}

	c.mu.Lock()
	oldConn := c.conn
	c.conn = newConn
	c.mu.Unlock()

	if oldConn != nil {
		_ = oldConn.Close()
	}
	return nil
}

// Close closes the underlying gRPC connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// GRPCConn exposes the underlying gRPC connection for specialized queries.
func (c *Client) GRPCConn() *grpc.ClientConn {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn
}

// shouldUseTLS determines if TLS should be used based on the gRPC address.
func shouldUseTLS(addr string) bool {
	// Check for explicit port 443 (standard HTTPS/gRPC-TLS port).
	if strings.HasSuffix(addr, ":443") {
		return true
	}

	// Check if it's a local address (localhost, 127.0.0.1, or no hostname).
	if strings.HasPrefix(addr, "localhost:") ||
		strings.HasPrefix(addr, "127.0.0.1:") ||
		strings.HasPrefix(addr, "0.0.0.0:") ||
		strings.HasPrefix(addr, ":") { // Just port, implies localhost.
		return false
	}

	// For any other remote address, prefer TLS by default for security.
	if !strings.Contains(addr, "localhost") &&
		!strings.Contains(addr, "127.0.0.1") &&
		!strings.Contains(addr, "0.0.0.0") {
		return true
	}

	return false
}
