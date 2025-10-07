package blockchain

import (
	"context"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"google.golang.org/grpc"
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
	// Create gRPC connection
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
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

// Close closes the blockchain client connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
