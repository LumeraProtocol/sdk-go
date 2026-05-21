package blockchain

import (
	"context"

	precisebanktypes "github.com/cosmos/evm/x/precisebank/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// PreciseBankClient provides x/precisebank module query operations.
type PreciseBankClient struct {
	query precisebanktypes.QueryClient
}

// Remainder returns the precisebank reserve amount not yet owned by any
// account (sub-ulume accounting bucket).
func (c *PreciseBankClient) Remainder(ctx context.Context) (sdk.Coin, error) {
	resp, err := c.query.Remainder(ctx, &precisebanktypes.QueryRemainderRequest{})
	if err != nil {
		return sdk.Coin{}, err
	}
	if resp == nil {
		return sdk.Coin{}, nil
	}
	return resp.Remainder, nil
}

// FractionalBalance returns the sub-ulume fractional balance for an address.
// Does not include the integer balance stored in x/bank.
func (c *PreciseBankClient) FractionalBalance(ctx context.Context, address string) (sdk.Coin, error) {
	resp, err := c.query.FractionalBalance(ctx, &precisebanktypes.QueryFractionalBalanceRequest{
		Address: address,
	})
	if err != nil {
		return sdk.Coin{}, err
	}
	if resp == nil {
		return sdk.Coin{}, nil
	}
	return resp.FractionalBalance, nil
}
