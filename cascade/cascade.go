package cascade

import (
	"context"
	"fmt"

	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
	"github.com/LumeraProtocol/sdk-go/blockchain"
	sdkEvent "github.com/LumeraProtocol/sdk-go/cascade/event"
	sdklog "github.com/LumeraProtocol/sdk-go/pkg/log"
	"github.com/LumeraProtocol/sdk-go/types"
)

// CreateApproveActionMessage constructs a MsgApproveAction without broadcasting it.
func (c *Client) CreateApproveActionMessage(_ context.Context, creator string, actionID string) (*actiontypes.MsgApproveAction, error) {
	if creator == "" || actionID == "" {
		return nil, fmt.Errorf("creator and actionID are required")
	}
	return blockchain.NewMsgApproveAction(creator, actionID), nil
}

// SendApproveActionMessage signs, simulates and broadcasts the provided approve message.
func (c *Client) SendApproveActionMessage(ctx context.Context, bc *blockchain.Client, msg *actiontypes.MsgApproveAction, memo string) (*types.ActionResult, error) {
	if bc == nil || msg == nil {
		return nil, fmt.Errorf("blockchain client and msg are required")
	}
	return bc.ApproveActionTx(ctx, msg.Creator, msg.ActionId, memo)
}

func (c *Client) emitClientEvent(ctx context.Context, evt sdkEvent.Event) {
	if c.isLocalEventType(evt.Type) {
		c.emitLocalEvent(ctx, evt)
	}
}

func (c *Client) logf(format string, args ...interface{}) {
	if c.logger == nil {
		return
	}
	sdklog.Infof(c.logger, format, args...)
}

func normalizeTaskType(raw string) string {
	if raw == "" {
		return ""
	}
	if val, ok := actiontypes.ActionType_value[raw]; ok {
		switch actiontypes.ActionType(val) {
		case actiontypes.ActionTypeCascade:
			return string(types.ActionTypeCascade)
		case actiontypes.ActionTypeSense:
			return string(types.ActionTypeSense)
		}
	}
	return raw
}
