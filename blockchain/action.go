package blockchain

import (
	"context"
	"fmt"
	"strconv"

	txtypes "cosmossdk.io/api/cosmos/tx/v1beta1"
	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
	"github.com/LumeraProtocol/sdk-go/types"
	"github.com/cosmos/cosmos-sdk/types/query"
)

// -------- Message Constructors --------

// NewMsgRequestAction constructs a MsgRequestAction with the provided parameters.
// Converts typed inputs to the string format required by the protobuf message.
func NewMsgRequestAction(
	creator string,
	actionType actiontypes.ActionType,
	metadata string,
	price string,
	expiration string,
	fileSizeKbs int64,
) *actiontypes.MsgRequestAction {
	// Convert ActionType enum to string
	actionTypeStr := actionType.String()
	fileSizeKbsStr := ""
	if fileSizeKbs != 0 {
		fileSizeKbsStr = strconv.FormatInt(fileSizeKbs, 10)
	}

	return &actiontypes.MsgRequestAction{
		Creator:        creator,
		ActionType:     actionTypeStr,
		Metadata:       metadata,
		Price:          price,
		ExpirationTime: expiration,
		FileSizeKbs:    fileSizeKbsStr,
	}
}

// NewMsgApproveAction constructs a MsgApproveAction with the provided creator and actionID.
func NewMsgApproveAction(
	creator string,
	actionID string,
) *actiontypes.MsgApproveAction {
	return &actiontypes.MsgApproveAction{
		Creator:  creator,
		ActionId: actionID,
	}
}

// NewMsgFinalizeAction constructs a MsgFinalizeAction with the provided parameters.
func NewMsgFinalizeAction(
	creator string,
	actionID string,
	actionType actiontypes.ActionType,
	metadata string,
) *actiontypes.MsgFinalizeAction {
	actionTypeStr := actionType.String()
	return &actiontypes.MsgFinalizeAction{
		Creator:    creator,
		ActionId:   actionID,
		ActionType: actionTypeStr,
		Metadata:   metadata,
	}
}

// NewMsgUpdateParams constructs a MsgUpdateParams with the provided authority and params.
func NewMsgUpdateParams(
	authority string,
	params actiontypes.Params,
) *actiontypes.MsgUpdateParams {
	return &actiontypes.MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}
}

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

// Params retrieves the Action module parameters.
func (a *ActionClient) Params(ctx context.Context) (*actiontypes.Params, error) {
	resp, err := a.query.Params(ctx, &actiontypes.QueryParamsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get action params: %w", err)
	}
	if resp == nil {
		return nil, fmt.Errorf("empty params response")
	}
	return &resp.Params, nil
}

// ListActionsBySuperNode lists actions for a specific supernode address with pagination.
func (a *ActionClient) ListActionsBySuperNode(ctx context.Context, superNodeAddress string, limit, offset uint64) ([]*types.Action, error) {
	req := &actiontypes.QueryListActionsBySuperNodeRequest{
		SuperNodeAddress: superNodeAddress,
		Pagination: &query.PageRequest{
			Limit:  limit,
			Offset: offset,
		},
	}
	resp, err := a.query.ListActionsBySuperNode(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list actions by supernode: %w", err)
	}

	actions := make([]*types.Action, len(resp.Actions))
	for i, protoAction := range resp.Actions {
		actions[i] = types.ActionFromProto(protoAction)
	}

	return actions, nil
}

// ListActionsByBlockHeight lists actions created at a specific block height with pagination.
func (a *ActionClient) ListActionsByBlockHeight(ctx context.Context, blockHeight int64, limit, offset uint64) ([]*types.Action, error) {
	req := &actiontypes.QueryListActionsByBlockHeightRequest{
		BlockHeight: blockHeight,
		Pagination: &query.PageRequest{
			Limit:  limit,
			Offset: offset,
		},
	}
	resp, err := a.query.ListActionsByBlockHeight(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list actions by block height: %w", err)
	}

	actions := make([]*types.Action, len(resp.Actions))
	for i, protoAction := range resp.Actions {
		actions[i] = types.ActionFromProto(protoAction)
	}

	return actions, nil
}

// ListExpiredActions lists expired actions with pagination.
func (a *ActionClient) ListExpiredActions(ctx context.Context, limit, offset uint64) ([]*types.Action, error) {
	req := &actiontypes.QueryListExpiredActionsRequest{
		Pagination: &query.PageRequest{
			Limit:  limit,
			Offset: offset,
		},
	}
	resp, err := a.query.ListExpiredActions(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list expired actions: %w", err)
	}

	actions := make([]*types.Action, len(resp.Actions))
	for i, protoAction := range resp.Actions {
		actions[i] = types.ActionFromProto(protoAction)
	}

	return actions, nil
}

// QueryActionByMetadataEnum queries actions by metadata and typed ActionType with pagination.
func (a *ActionClient) QueryActionByMetadataEnum(ctx context.Context, actionType actiontypes.ActionType, metadataQuery string, limit, offset uint64) ([]*types.Action, error) {
	req := &actiontypes.QueryActionByMetadataRequest{
		ActionType:    actionType,
		MetadataQuery: metadataQuery,
		Pagination: &query.PageRequest{
			Limit:  limit,
			Offset: offset,
		},
	}
	resp, err := a.query.QueryActionByMetadata(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to query actions by metadata: %w", err)
	}

	actions := make([]*types.Action, len(resp.Actions))
	for i, protoAction := range resp.Actions {
		actions[i] = types.ActionFromProto(protoAction)
	}

	return actions, nil
}

// QueryActionByMetadata queries actions by metadata and string ActionType with pagination.
func (a *ActionClient) QueryActionByMetadata(ctx context.Context, actionTypeStr, metadataQuery string, limit, offset uint64) ([]*types.Action, error) {
	at, ok := parseActionType(actionTypeStr)
	if !ok {
		at = 0 // unspecified
	}
	return a.QueryActionByMetadataEnum(ctx, at, metadataQuery, limit, offset)
}

// -------- Transaction Helpers --------

// RequestActionTx builds, signs, broadcasts and confirms a MsgRequestAction.
func (c *Client) RequestActionTx(ctx context.Context, creator string, actionType actiontypes.ActionType, metadata, price, expiration string, fileSizeKbs int64, memo string) (*types.ActionResult, error) {
	msg := NewMsgRequestAction(creator, actionType, metadata, price, expiration, fileSizeKbs)

	txBytes, err := c.BuildAndSignTx(ctx, msg, memo)
	if err != nil {
		return nil, fmt.Errorf("build and sign tx: %w", err)
	}

	txHash, err := c.Broadcast(ctx, txBytes, txtypes.BroadcastMode_BROADCAST_MODE_SYNC)
	if err != nil {
		return nil, fmt.Errorf("broadcast tx: %w", err)
	}

	resp, err := c.WaitForTxInclusion(ctx, txHash)
	if err != nil {
		return nil, fmt.Errorf("wait for tx inclusion: %w", err)
	}

	actionID, err := c.ExtractEventAttribute(resp, "action_registered", "action_id")
	if err != nil {
		return nil, fmt.Errorf("extract action_id: %w", err)
	}

	return &types.ActionResult{
		ActionID: actionID,
		TxHash:   txHash,
		Height:   resp.TxResponse.Height,
	}, nil
}

// FinalizeActionTx builds, signs, broadcasts and confirms a MsgFinalizeAction.
func (c *Client) FinalizeActionTx(ctx context.Context, creator, actionID string, actionType actiontypes.ActionType, metadata, memo string) (*types.ActionResult, error) {
	msg := NewMsgFinalizeAction(creator, actionID, actionType, metadata)

	txBytes, err := c.BuildAndSignTx(ctx, msg, memo)
	if err != nil {
		return nil, fmt.Errorf("build and sign tx: %w", err)
	}

	txHash, err := c.Broadcast(ctx, txBytes, txtypes.BroadcastMode_BROADCAST_MODE_SYNC)
	if err != nil {
		return nil, fmt.Errorf("broadcast tx: %w", err)
	}

	resp, err := c.WaitForTxInclusion(ctx, txHash)
	if err != nil {
		return nil, fmt.Errorf("wait for tx inclusion: %w", err)
	}

	return &types.ActionResult{
		ActionID: actionID, // echo input; chain may emit events, but we don't rely on them here
		TxHash:   txHash,
		Height:   resp.TxResponse.Height,
	}, nil
}

// ApproveActionTx builds, signs, broadcasts and confirms a MsgApproveAction.
func (c *Client) ApproveActionTx(ctx context.Context, creator, actionID, memo string) (*types.ActionResult, error) {
	msg := NewMsgApproveAction(creator, actionID)

	txBytes, err := c.BuildAndSignTx(ctx, msg, memo)
	if err != nil {
		return nil, fmt.Errorf("build and sign tx: %w", err)
	}

	txHash, err := c.Broadcast(ctx, txBytes, txtypes.BroadcastMode_BROADCAST_MODE_SYNC)
	if err != nil {
		return nil, fmt.Errorf("broadcast tx: %w", err)
	}

	resp, err := c.WaitForTxInclusion(ctx, txHash)
	if err != nil {
		return nil, fmt.Errorf("wait for tx inclusion: %w", err)
	}

	return &types.ActionResult{
		ActionID: actionID, // echo input
		TxHash:   txHash,
		Height:   resp.TxResponse.Height,
	}, nil
}

// UpdateActionParamsTx builds, signs, broadcasts and confirms a MsgUpdateParams for the Action module.
func (c *Client) UpdateActionParamsTx(ctx context.Context, authority string, params actiontypes.Params, memo string) (*types.ActionResult, error) {
	msg := NewMsgUpdateParams(authority, params)

	txBytes, err := c.BuildAndSignTx(ctx, msg, memo)
	if err != nil {
		return nil, fmt.Errorf("build and sign tx: %w", err)
	}

	txHash, err := c.Broadcast(ctx, txBytes, txtypes.BroadcastMode_BROADCAST_MODE_SYNC)
	if err != nil {
		return nil, fmt.Errorf("broadcast tx: %w", err)
	}

	resp, err := c.WaitForTxInclusion(ctx, txHash)
	if err != nil {
		return nil, fmt.Errorf("wait for tx inclusion: %w", err)
	}

	return &types.ActionResult{
		// ActionID intentionally empty for params update
		TxHash: txHash,
		Height: resp.TxResponse.Height,
	}, nil
}
