package blockchain

import (
	"context"
	"fmt"

	sdkmath "cosmossdk.io/math"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
)

// FeeMarketClient provides x/feemarket module query operations.
type FeeMarketClient struct {
	query feemarkettypes.QueryClient
}

// Params returns the current feemarket module parameters.
func (c *FeeMarketClient) Params(ctx context.Context) (*feemarkettypes.QueryParamsResponse, error) {
	return c.query.Params(ctx, &feemarkettypes.QueryParamsRequest{})
}

// BaseFee returns the EIP-1559 base fee of the parent block, expressed in the
// native denom (ulume on Lumera). Callers that need an integer wei-like value
// for EIP-1559 fee fields should pass the result through
// pkg/crypto.ULumeDecToWei.
func (c *FeeMarketClient) BaseFee(ctx context.Context) (sdkmath.LegacyDec, error) {
	resp, err := c.query.BaseFee(ctx, &feemarkettypes.QueryBaseFeeRequest{})
	if err != nil {
		return sdkmath.LegacyDec{}, err
	}
	if resp == nil || resp.BaseFee == nil {
		return sdkmath.LegacyZeroDec(), nil
	}
	return *resp.BaseFee, nil
}

// BlockGas returns the gas used at the most recent block.
func (c *FeeMarketClient) BlockGas(ctx context.Context) (int64, error) {
	resp, err := c.query.BlockGas(ctx, &feemarkettypes.QueryBlockGasRequest{})
	if err != nil {
		return 0, err
	}
	if resp == nil {
		return 0, fmt.Errorf("empty block gas response")
	}
	return resp.Gas, nil
}
