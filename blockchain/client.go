package blockchain

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
	//audittypes "github.com/LumeraProtocol/lumera/x/audit/types"
	claimtypes "github.com/LumeraProtocol/lumera/x/claim/types"
	supernodetypes "github.com/LumeraProtocol/lumera/x/supernode/v1/types"
)

// Config for blockchain client
type Config struct {
	ChainID        string
	GRPCAddr       string
	Timeout        time.Duration
	MaxRecvMsgSize int
	MaxSendMsgSize int
}

// Client provides access to blockchain operations
type Client struct {
	// Module-specific clients
	Action    *ActionClient
	SuperNode *SuperNodeClient
	Claim     *ClaimClient
	Audit     *AuditClient

	// Internal
	conn    *grpc.ClientConn
	config  Config
	keyring keyring.Keyring
	keyName string
}

// New creates a new blockchain client
func New(ctx context.Context, cfg Config, kr keyring.Keyring, keyName string) (*Client, error) {
	// Determine if we should use TLS based on the endpoint
	// Use TLS if: port is 443, or hostname doesn't start with "localhost"/"127.0.0.1"
	useTLS := shouldUseTLS(cfg.GRPCAddr)
	
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

	conn, err := grpc.DialContext(ctx, cfg.GRPCAddr, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC: %w", err)
	}

	// Create module clients
	actionClient := &ActionClient{
		query: actiontypes.NewQueryClient(conn),
	}

	supernodeClient := &SuperNodeClient{
		query: supernodetypes.NewQueryClient(conn),
	}

	claimClient := &ClaimClient{
		query: claimtypes.NewQueryClient(conn),
	}

	auditClient := &AuditClient{
		//query: audittypes.NewQueryClient(conn),
	}

	return &Client{
		Action:    actionClient,
		SuperNode: supernodeClient,
		Claim:     claimClient,
		Audit:     auditClient,
		conn:      conn,
		config:    cfg,
		keyring:   kr,
		keyName:   keyName,
	}, nil
}

// shouldUseTLS determines if TLS should be used based on the gRPC address
func shouldUseTLS(addr string) bool {
	// Check for explicit port 443 (standard HTTPS/gRPC-TLS port)
	if strings.HasSuffix(addr, ":443") {
		return true
	}
	
	// Check if it's a local address (localhost, 127.0.0.1, or no hostname)
	if strings.HasPrefix(addr, "localhost:") ||
		strings.HasPrefix(addr, "127.0.0.1:") ||
		strings.HasPrefix(addr, "0.0.0.0:") ||
		strings.HasPrefix(addr, ":") { // Just port, implies localhost
		return false
	}
	
	// For any other remote address, prefer TLS by default for security
	// This covers domain names without explicit port 443
	if !strings.Contains(addr, "localhost") &&
		!strings.Contains(addr, "127.0.0.1") &&
		!strings.Contains(addr, "0.0.0.0") {
		return true
	}
	
	return false
}

// Close closes the blockchain client connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
