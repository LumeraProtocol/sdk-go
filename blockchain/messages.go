package blockchain

import (
	"encoding/json"
	"strconv"

	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
	"google.golang.org/protobuf/types/known/anypb"
)

// NewMsgRequestAction constructs a MsgRequestAction with the provided parameters.
// Converts typed inputs to the string format required by the protobuf message.
func NewMsgRequestAction(
	creator string,
	actionType actiontypes.ActionType,
	metadata *anypb.Any,
	price string,
	expirationTime int64,
	superNodes []string,
) *actiontypes.MsgRequestAction {
	// Convert ActionType enum to string
	actionTypeStr := actionType.String()

	// Convert metadata Any to JSON string
	metadataStr := ""
	if metadata != nil {
		if metadataBytes, err := json.Marshal(metadata); err == nil {
			metadataStr = string(metadataBytes)
		}
	}

	// Convert expiration time to string
	expirationTimeStr := strconv.FormatInt(expirationTime, 10)

	return &actiontypes.MsgRequestAction{
		Creator:        creator,
		ActionType:     actionTypeStr,
		Metadata:       metadataStr,
		Price:          price,
		ExpirationTime: expirationTimeStr,
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