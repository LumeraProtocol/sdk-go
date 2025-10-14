package blockchain

import (
	"context"
	"fmt"

	txtypes "cosmossdk.io/api/cosmos/tx/v1beta1"
	supernodetypes "github.com/LumeraProtocol/lumera/x/supernode/v1/types"
	"github.com/LumeraProtocol/sdk-go/types"
	"github.com/cosmos/cosmos-sdk/types/query"
)

// -------- Message Constructors --------

// NewSuperNodeMsgUpdateParams constructs a SuperNode MsgUpdateParams with the provided authority and params.
func NewSuperNodeMsgUpdateParams(
	authority string,
	params supernodetypes.Params,
) *supernodetypes.MsgUpdateParams {
	return &supernodetypes.MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}
}

// NewMsgRegisterSupernode constructs a MsgRegisterSupernode.
func NewMsgRegisterSupernode(
	creator, validatorAddress, ipAddress, supernodeAccount, p2pPort string,
) *supernodetypes.MsgRegisterSupernode {
	return &supernodetypes.MsgRegisterSupernode{
		Creator:          creator,
		ValidatorAddress: validatorAddress,
		IpAddress:        ipAddress,
		SupernodeAccount: supernodeAccount,
		P2PPort:          p2pPort,
	}
}

// NewMsgDeregisterSupernode constructs a MsgDeregisterSupernode.
func NewMsgDeregisterSupernode(
	creator, validatorAddress string,
) *supernodetypes.MsgDeregisterSupernode {
	return &supernodetypes.MsgDeregisterSupernode{
		Creator:          creator,
		ValidatorAddress: validatorAddress,
	}
}

// NewMsgStartSupernode constructs a MsgStartSupernode.
func NewMsgStartSupernode(
	creator, validatorAddress string,
) *supernodetypes.MsgStartSupernode {
	return &supernodetypes.MsgStartSupernode{
		Creator:          creator,
		ValidatorAddress: validatorAddress,
	}
}

// NewMsgStopSupernode constructs a MsgStopSupernode.
func NewMsgStopSupernode(
	creator, validatorAddress, reason string,
) *supernodetypes.MsgStopSupernode {
	return &supernodetypes.MsgStopSupernode{
		Creator:          creator,
		ValidatorAddress: validatorAddress,
		Reason:           reason,
	}
}

// NewMsgUpdateSupernode constructs a MsgUpdateSupernode.
func NewMsgUpdateSupernode(
	creator, validatorAddress, ipAddress, note, supernodeAccount, p2pPort string,
) *supernodetypes.MsgUpdateSupernode {
	return &supernodetypes.MsgUpdateSupernode{
		Creator:          creator,
		ValidatorAddress: validatorAddress,
		IpAddress:        ipAddress,
		Note:             note,
		SupernodeAccount: supernodeAccount,
		P2PPort:          p2pPort,
	}
}

// SuperNodeClient provides supernode module operations
type SuperNodeClient struct {
	query supernodetypes.QueryClient
}

// GetSuperNode retrieves a supernode by validator address
func (s *SuperNodeClient) GetSuperNode(ctx context.Context, validatorAddr string) (*types.SuperNode, error) {
	resp, err := s.query.GetSuperNode(ctx, &supernodetypes.QueryGetSuperNodeRequest{
		ValidatorAddress: validatorAddr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get supernode: %w", err)
	}

	return types.SuperNodeFromProto(resp.Supernode), nil
}

 // GetTopSuperNodesForBlock retrieves top supernodes for a specific block
func (s *SuperNodeClient) GetTopSuperNodesForBlock(ctx context.Context, blockHeight int32) ([]*supernodetypes.SuperNode, error) {
	resp, err := s.query.GetTopSuperNodesForBlock(ctx, &supernodetypes.QueryGetTopSuperNodesForBlockRequest{
		BlockHeight: blockHeight,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get top supernodes: %w", err)
	}

	return resp.Supernodes, nil
}

// Params retrieves the SuperNode module parameters.
func (s *SuperNodeClient) Params(ctx context.Context) (*supernodetypes.Params, error) {
	resp, err := s.query.Params(ctx, &supernodetypes.QueryParamsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get supernode params: %w", err)
	}
	if resp == nil {
		return nil, fmt.Errorf("empty params response")
	}
	return &resp.Params, nil
}

// GetSuperNodeBySuperNodeAddress retrieves a supernode by its supernode account address.
func (s *SuperNodeClient) GetSuperNodeBySuperNodeAddress(ctx context.Context, supernodeAddress string) (*types.SuperNode, error) {
	resp, err := s.query.GetSuperNodeBySuperNodeAddress(ctx, &supernodetypes.QueryGetSuperNodeBySuperNodeAddressRequest{
		SupernodeAddress: supernodeAddress,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get supernode by address: %w", err)
	}

	return types.SuperNodeFromProto(resp.Supernode), nil
}

// ListSuperNodes returns a paginated list of supernodes (converted to SDK types).
func (s *SuperNodeClient) ListSuperNodes(ctx context.Context, limit, offset uint64) ([]*types.SuperNode, error) {
	resp, err := s.query.ListSuperNodes(ctx, &supernodetypes.QueryListSuperNodesRequest{
		Pagination: &query.PageRequest{
			Limit:  limit,
			Offset: offset,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list supernodes: %w", err)
	}

	sns := make([]*types.SuperNode, len(resp.Supernodes))
	for i, pb := range resp.Supernodes {
		sns[i] = types.SuperNodeFromProto(pb)
	}
	return sns, nil
}

// GetTopSuperNodesForBlockWithOptions retrieves top supernodes for a block with optional limit and state filter.
func (s *SuperNodeClient) GetTopSuperNodesForBlockWithOptions(ctx context.Context, blockHeight int32, limit int32, state string) ([]*types.SuperNode, error) {
	resp, err := s.query.GetTopSuperNodesForBlock(ctx, &supernodetypes.QueryGetTopSuperNodesForBlockRequest{
		BlockHeight: blockHeight,
		Limit:       limit,
		State:       state,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get top supernodes: %w", err)
	}

	sns := make([]*types.SuperNode, len(resp.Supernodes))
	for i, pb := range resp.Supernodes {
		sns[i] = types.SuperNodeFromProto(pb)
	}
	return sns, nil
}

// -------- Transaction Helpers --------

// UpdateSuperNodeParamsTx builds, signs, broadcasts and confirms a SuperNode MsgUpdateParams.
func (c *Client) UpdateSuperNodeParamsTx(ctx context.Context, authority string, params supernodetypes.Params, memo string) (*types.ActionResult, error) {
	msg := NewSuperNodeMsgUpdateParams(authority, params)

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
		TxHash: txHash,
		Height: resp.TxResponse.Height,
	}, nil
}

// RegisterSupernodeTx registers a new supernode.
func (c *Client) RegisterSupernodeTx(ctx context.Context, creator, validatorAddress, ipAddress, supernodeAccount, p2pPort, memo string) (*types.ActionResult, error) {
	msg := NewMsgRegisterSupernode(creator, validatorAddress, ipAddress, supernodeAccount, p2pPort)

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
		TxHash: txHash,
		Height: resp.TxResponse.Height,
	}, nil
}

// DeregisterSupernodeTx de-registers an existing supernode.
func (c *Client) DeregisterSupernodeTx(ctx context.Context, creator, validatorAddress, memo string) (*types.ActionResult, error) {
	msg := NewMsgDeregisterSupernode(creator, validatorAddress)

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
		TxHash: txHash,
		Height: resp.TxResponse.Height,
	}, nil
}

// StartSupernodeTx starts a supernode.
func (c *Client) StartSupernodeTx(ctx context.Context, creator, validatorAddress, memo string) (*types.ActionResult, error) {
	msg := NewMsgStartSupernode(creator, validatorAddress)

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
		TxHash: txHash,
		Height: resp.TxResponse.Height,
	}, nil
}

// StopSupernodeTx stops a supernode with a reason.
func (c *Client) StopSupernodeTx(ctx context.Context, creator, validatorAddress, reason, memo string) (*types.ActionResult, error) {
	msg := NewMsgStopSupernode(creator, validatorAddress, reason)

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
		TxHash: txHash,
		Height: resp.TxResponse.Height,
	}, nil
}

// UpdateSupernodeTx updates a supernode's info.
func (c *Client) UpdateSupernodeTx(ctx context.Context, creator, validatorAddress, ipAddress, note, supernodeAccount, p2pPort, memo string) (*types.ActionResult, error) {
	msg := NewMsgUpdateSupernode(creator, validatorAddress, ipAddress, note, supernodeAccount, p2pPort)

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
		TxHash: txHash,
		Height: resp.TxResponse.Height,
	}, nil
}
