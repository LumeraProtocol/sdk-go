package blockchain

import (
	"context"
	"fmt"

	supernodetypes "github.com/LumeraProtocol/lumera/x/supernode/v1/types"
	"github.com/LumeraProtocol/sdk-go/types"
)

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
