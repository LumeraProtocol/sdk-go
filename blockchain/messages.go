package blockchain

import (
	"encoding/json"

	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
	"google.golang.org/protobuf/types/known/anypb"
)

// NewMsgRequestAction constructs a MsgRequestAction with the provided parameters.
// Converts typed inputs to the string format required by the protobuf message.
func NewMsgRequestAction(
	creator string,
	actionType actiontypes.ActionType,
	metadata string,
	price string,
	expiration string,
) *actiontypes.MsgRequestAction {
	// Convert ActionType enum to string
	actionTypeStr := actionType.String()

	return &actiontypes.MsgRequestAction{
		Creator:        creator,
		ActionType:     actionTypeStr,
		Metadata:       metadata,
		Price:          price,
		ExpirationTime: expiration,
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
