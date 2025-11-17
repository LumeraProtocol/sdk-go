//go:build ignore
// +build ignore

package client

import (
	"context"
	"time"

	nodev1 "cosmossdk.io/api/cosmos/base/node/v1beta1"
	actionv1 "github.com/LumeraProtocol/lumera/x/action/v1/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
)

type LumeraClient interface {
	Connect(ctx context.Context) error

	// Queries (fees/limits, action status, etc.)
	GetAction(ctx context.Context, id string) (*actionv1.Action, error)
	GetParams(ctx context.Context) (*actionv1.Params, error)
	GetActionFee(ctx context.Context, dataSizeKb int64) (fee string, err error)

	// Accounts (for tx signing)
	GetAccount(ctx context.Context, addr string) (number uint64, sequence uint64, err error)

	// Node
	GetNodeConfig(ctx context.Context) (*nodev1.ConfigResponse, error)

	// Tx path
	BuildSignBroadcast(ctx context.Context, msgs []sdktypes.Msg) (txResponse *sdktypes.TxResponse, err error)
	WaitTx(ctx context.Context, txHash string, timeout time.Duration) (code uint32, events map[string][]string, err error)

	// High-level helper
	RequestActionCascade(ctx context.Context, creator, metadataJSON, price, expirationTime string) (txResponse *sdktypes.TxResponse, err error)
}
