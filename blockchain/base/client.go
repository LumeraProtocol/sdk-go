package base

import (
	"context"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	clientconfig "github.com/LumeraProtocol/sdk-go/client/config"
)

const defaultMaxMessageSize = 50 * 1024 * 1024

// Client provides common Cosmos SDK gRPC and tx helpers.
type Client struct {
	conn    *grpc.ClientConn
	config  Config
	keyring keyring.Keyring
	keyName string
}

// New creates a base blockchain client with a gRPC connection.
func New(ctx context.Context, cfg Config, kr keyring.Keyring, keyName string) (*Client, error) {
	applyConfigDefaults(&cfg)

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
		return nil, fmt.Errorf("failed to connect to gRPC: %w", err)
	}

	return &Client{
		conn:    conn,
		config:  cfg,
		keyring: kr,
		keyName: keyName,
	}, nil
}

func applyConfigDefaults(cfg *Config) {
	if cfg == nil {
		return
	}
	if cfg.MaxRecvMsgSize <= 0 {
		cfg.MaxRecvMsgSize = defaultMaxMessageSize
	}
	if cfg.MaxSendMsgSize <= 0 {
		cfg.MaxSendMsgSize = defaultMaxMessageSize
	}
	clientconfig.ApplyWaitTxDefaults(&cfg.WaitTx)
}

// Close closes the underlying gRPC connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// GRPCConn exposes the underlying gRPC connection for specialized queries.
func (c *Client) GRPCConn() *grpc.ClientConn {
	return c.conn
}

// Keyring returns the keyring used for signing.
func (c *Client) Keyring() keyring.Keyring {
	return c.keyring
}

// KeyName returns the key uid used for signing.
func (c *Client) KeyName() string {
	return c.keyName
}

// Cfg returns a copy of the base client configuration.
func (c *Client) Cfg() Config {
	return c.config
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
