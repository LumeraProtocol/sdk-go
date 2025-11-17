//go:build ignore
// +build ignore

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	sdktypes "github.com/cosmos/cosmos-sdk/types"

	actionv1 "github.com/LumeraProtocol/lumera/x/action/v1/types"
)

// GetAction fetches an action by ID (simple passthrough to query service).
func (c *clientImpl) GetAction(ctx context.Context, id string) (*actionv1.Action, error) {
	resp, err := c.actQ.GetAction(ctx, &actionv1.QueryGetActionRequest{ActionID: id})
	if err != nil {
		return nil, err
	}
	return resp.Action, nil
}

// GetParams fetches on-chain action module parameters (fees, limits, etc.).
func (c *clientImpl) GetParams(ctx context.Context) (*actionv1.Params, error) {
	resp, err := c.actQ.Params(ctx, &actionv1.QueryParamsRequest{})
	if err != nil {
		return nil, err
	}
	return &resp.Params, nil
}

func (c *clientImpl) GetActionFee(ctx context.Context, dataSizeKb int64) (string, error) {
	resp, err := c.actQ.GetActionFee(ctx, &actionv1.QueryGetActionFeeRequest{
		DataSize: strconv.FormatInt(dataSizeKb, 10),
	})
	if err != nil {
		return "", err
	}
	return resp.Amount, nil
}

// RequestActionCascade builds MsgRequestAction for Cascade and broadcasts it.
//
// metadataJSON: JSON-encoded actionv1.CascadeMetadata (RequestAction required fields).
// price: string amount with denom, e.g., "1000000ulume"
// expireAt: absolute time limit (sdk side expects string-formatted time; we’ll encode in RFC3339).
func (c *clientImpl) RequestActionCascade(
	ctx context.Context,
	creator string,
	metadataJSON string,
	price string,
	expirationTime string,
) (*sdktypes.TxResponse, error) {
	// Validate basic fields
	if strings.TrimSpace(metadataJSON) == "" {
		return nil, fmt.Errorf("metadata is empty")
	}
	// Quick sanity-check that metadata is valid JSON (catch common errors early).
	var tmp map[string]any
	if err := json.Unmarshal([]byte(metadataJSON), &tmp); err != nil {
		return nil, fmt.Errorf("invalid cascade metadata json: %w", err)
	}

	// Build the action request message
	msg := &actionv1.MsgRequestAction{
		Creator:        creator,
		ActionType:     "CASCADE",    // module expects string enum in pb.go
		Metadata:       metadataJSON, // raw JSON string
		Price:          price,        // "amountdenom" (module parses)
		ExpirationTime: expirationTime,
	}

	// Delegate to tx helper that will sign + broadcast.
	txResponse, err := c.BuildSignBroadcast(ctx, []sdktypes.Msg{msg})
	if err != nil {
		return nil, err
	}
	return txResponse, nil
}
