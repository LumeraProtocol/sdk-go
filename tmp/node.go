//go:build ignore
// +build ignore

package client

import (
	"context"

	nodev1 "cosmossdk.io/api/cosmos/base/node/v1beta1"
)

func (c *clientImpl) GetNodeConfig(ctx context.Context) (*nodev1.ConfigResponse, error) {
	return c.nodeS.Config(ctx, &nodev1.ConfigRequest{})
}
