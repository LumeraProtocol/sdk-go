package blockchain

import (
	"context"

	evmigrationtypes "github.com/LumeraProtocol/lumera/x/evmigration/types"
)

// EVMigrationClient provides evmigration module query operations.
type EVMigrationClient struct {
	query evmigrationtypes.QueryClient
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
