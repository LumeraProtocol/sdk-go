package blockchain

import (
	"context"
	"strings"

	sdkmath "cosmossdk.io/math"
	"github.com/LumeraProtocol/sdk-go/blockchain/base"
	"github.com/LumeraProtocol/sdk-go/constants"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"

	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
	//audittypes "github.com/LumeraProtocol/lumera/x/audit/types"
	claimtypes "github.com/LumeraProtocol/lumera/x/claim/types"
	evmigrationtypes "github.com/LumeraProtocol/lumera/x/evmigration/types"
	supernodetypes "github.com/LumeraProtocol/lumera/x/supernode/v1/types"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	precisebanktypes "github.com/cosmos/evm/x/precisebank/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

// Config mirrors the base blockchain config for Lumera-specific usage.
type Config = base.Config

// Client provides access to Lumera-specific blockchain operations.
type Client struct {
	*base.Client

	// Module-specific clients
	Action      *ActionClient
	SuperNode   *SuperNodeClient
	Claim       *ClaimClient
	EVMigration *EVMigrationClient
	Audit       *AuditClient

	// EVM module clients (cosmos/evm)
	EVM         *EVMClient
	FeeMarket   *FeeMarketClient
	PreciseBank *PreciseBankClient
}

// New creates a new Lumera blockchain client.
func New(ctx context.Context, cfg Config, kr keyring.Keyring, keyName string) (*Client, error) {
	if strings.TrimSpace(cfg.AccountHRP) == "" {
		cfg.AccountHRP = constants.LumeraAccountHRP
	}
	if strings.TrimSpace(cfg.FeeDenom) == "" {
		cfg.FeeDenom = "ulume"
	}
	if cfg.GasPrice.IsNil() || cfg.GasPrice.IsZero() {
		cfg.GasPrice = sdkmath.LegacyNewDecWithPrec(25, 3) // 0.025
	}

	baseClient, err := base.New(ctx, cfg, kr, keyName)
	if err != nil {
		return nil, err
	}

	conn := baseClient.GRPCConn()
	return &Client{
		Client: baseClient,
		Action: &ActionClient{
			query: actiontypes.NewQueryClient(conn),
		},
		SuperNode: &SuperNodeClient{
			query: supernodetypes.NewQueryClient(conn),
		},
		Claim: &ClaimClient{
			query: claimtypes.NewQueryClient(conn),
		},
		EVMigration: &EVMigrationClient{
			query: evmigrationtypes.NewQueryClient(conn),
		},
		Audit: &AuditClient{
			//query: audittypes.NewQueryClient(conn),
		},
		EVM: &EVMClient{
			query: evmtypes.NewQueryClient(conn),
		},
		FeeMarket: &FeeMarketClient{
			query: feemarkettypes.NewQueryClient(conn),
		},
		PreciseBank: &PreciseBankClient{
			query: precisebanktypes.NewQueryClient(conn),
		},
	}, nil
}
