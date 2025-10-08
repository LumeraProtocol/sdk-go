package blockchain

import (
	"context"
	"fmt"
	"strconv"

	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
	"github.com/LumeraProtocol/sdk-go/types"
)

// ActionClient provides action module operations
type ActionClient struct {
	query actiontypes.QueryClient
}

// GetAction retrieves an action by ID
func (a *ActionClient) GetAction(ctx context.Context, actionID string) (*types.Action, error) {
	resp, err := a.query.GetAction(ctx, &actiontypes.QueryGetActionRequest{
		ActionID: actionID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get action: %w", err)
	}

	return types.ActionFromProto(resp.Action), nil
}

// ListActions lists actions with optional filters
func (a *ActionClient) ListActions(ctx context.Context, opts ...QueryOption) ([]*types.Action, error) {
	req := &actiontypes.QueryListActionsRequest{}

	// Apply options
	for _, opt := range opts {
		opt.ApplyToActionQuery(req)
	}

	resp, err := a.query.ListActions(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list actions: %w", err)
	}

	// Convert to SDK types
	actions := make([]*types.Action, len(resp.Actions))
	for i, protoAction := range resp.Actions {
		actions[i] = types.ActionFromProto(protoAction)
	}

	return actions, nil
}

// GetActionFee calculates the fee for an action based on data size
func (a *ActionClient) GetActionFee(ctx context.Context, dataSize int64) (string, error) {
	resp, err := a.query.GetActionFee(ctx, &actiontypes.QueryGetActionFeeRequest{
		DataSize: strconv.FormatInt(dataSize, 10),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get action fee: %w", err)
	}

	return resp.Amount, nil
}


// ListActionsByType provides a convenience wrapper accepting actionType as a string with pagination.
func (a *ActionClient) ListActionsByType(ctx context.Context, actionType string, limit, offset uint64) ([]*types.Action, error) {
	return a.ListActions(ctx,
		WithActionTypeStr(actionType),
		WithPagination(limit, offset),
	)
}