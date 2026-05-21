package blockchain

import (
	"context"
	"fmt"

	txtypes "cosmossdk.io/api/cosmos/tx/v1beta1"
	evmigrationtypes "github.com/LumeraProtocol/lumera/x/evmigration/types"
)

// EVMigrationClient provides evmigration module query operations.
type EVMigrationClient struct {
	query evmigrationtypes.QueryClient
}

// MigrationResult contains the result of an evmigration tx.
type MigrationResult struct {
	LegacyAddress string
	NewAddress    string
	TxHash        string
	Height        int64
}

// Params returns the current evmigration module parameters.
func (c *EVMigrationClient) Params(ctx context.Context) (*evmigrationtypes.QueryParamsResponse, error) {
	return c.query.Params(ctx, &evmigrationtypes.QueryParamsRequest{})
}

// MigrationRecord returns the migration record for a legacy address, or nil if
// the account has not been migrated.
func (c *EVMigrationClient) MigrationRecord(ctx context.Context, legacyAddress string) (*evmigrationtypes.QueryMigrationRecordResponse, error) {
	return c.query.MigrationRecord(ctx, &evmigrationtypes.QueryMigrationRecordRequest{
		LegacyAddress: legacyAddress,
	})
}

// MigrationRecordByNewAddress returns the migration record for a migrated
// destination address, or nil if the address was not used as a migration target.
func (c *EVMigrationClient) MigrationRecordByNewAddress(ctx context.Context, newAddress string) (*evmigrationtypes.QueryMigrationRecordByNewAddressResponse, error) {
	return c.query.MigrationRecordByNewAddress(ctx, &evmigrationtypes.QueryMigrationRecordByNewAddressRequest{
		NewAddress: newAddress,
	})
}

// MigrationEstimate returns a pre-flight estimate of whether the migration
// would succeed, including asset counts and rejection reason.
func (c *EVMigrationClient) MigrationEstimate(ctx context.Context, legacyAddress string) (*evmigrationtypes.QueryMigrationEstimateResponse, error) {
	return c.query.MigrationEstimate(ctx, &evmigrationtypes.QueryMigrationEstimateRequest{
		LegacyAddress: legacyAddress,
	})
}

// MigrationStats returns aggregate migration statistics (total migrated,
// total legacy, etc.).
func (c *EVMigrationClient) MigrationStats(ctx context.Context) (*evmigrationtypes.QueryMigrationStatsResponse, error) {
	return c.query.MigrationStats(ctx, &evmigrationtypes.QueryMigrationStatsRequest{})
}

// -------- Message Constructors --------

// NewMsgClaimLegacyAccount constructs a MsgClaimLegacyAccount.
func NewMsgClaimLegacyAccount(newAddress, legacyAddress string, legacyProof, newProof evmigrationtypes.MigrationProof) *evmigrationtypes.MsgClaimLegacyAccount {
	return &evmigrationtypes.MsgClaimLegacyAccount{
		NewAddress:    newAddress,
		LegacyAddress: legacyAddress,
		LegacyProof:   legacyProof,
		NewProof:      newProof,
	}
}

// NewMsgMigrateValidator constructs a MsgMigrateValidator.
func NewMsgMigrateValidator(newAddress, legacyAddress string, legacyProof, newProof evmigrationtypes.MigrationProof) *evmigrationtypes.MsgMigrateValidator {
	return &evmigrationtypes.MsgMigrateValidator{
		NewAddress:    newAddress,
		LegacyAddress: legacyAddress,
		LegacyProof:   legacyProof,
		NewProof:      newProof,
	}
}

// -------- Transaction Helpers --------

// ClaimLegacyAccountTx builds, signs, broadcasts and confirms a MsgClaimLegacyAccount.
func (c *Client) ClaimLegacyAccountTx(ctx context.Context, msg *evmigrationtypes.MsgClaimLegacyAccount, memo string) (*MigrationResult, error) {
	if msg == nil {
		return nil, fmt.Errorf("msg is required")
	}

	txBytes, err := c.BuildAndSignTx(ctx, msg, memo)
	if err != nil {
		return nil, fmt.Errorf("build and sign tx: %w", err)
	}

	txHash, resp, err := c.BroadcastAndWait(ctx, txBytes, txtypes.BroadcastMode_BROADCAST_MODE_SYNC)
	if err != nil {
		return nil, fmt.Errorf("broadcast and wait: %w", err)
	}

	return &MigrationResult{
		LegacyAddress: msg.LegacyAddress,
		NewAddress:    msg.NewAddress,
		TxHash:        txHash,
		Height:        resp.TxResponse.Height,
	}, nil
}

// MigrateValidatorTx builds, signs, broadcasts and confirms a MsgMigrateValidator.
func (c *Client) MigrateValidatorTx(ctx context.Context, msg *evmigrationtypes.MsgMigrateValidator, memo string) (*MigrationResult, error) {
	if msg == nil {
		return nil, fmt.Errorf("msg is required")
	}

	txBytes, err := c.BuildAndSignTx(ctx, msg, memo)
	if err != nil {
		return nil, fmt.Errorf("build and sign tx: %w", err)
	}

	txHash, resp, err := c.BroadcastAndWait(ctx, txBytes, txtypes.BroadcastMode_BROADCAST_MODE_SYNC)
	if err != nil {
		return nil, fmt.Errorf("broadcast and wait: %w", err)
	}

	return &MigrationResult{
		LegacyAddress: msg.LegacyAddress,
		NewAddress:    msg.NewAddress,
		TxHash:        txHash,
		Height:        resp.TxResponse.Height,
	}, nil
}
