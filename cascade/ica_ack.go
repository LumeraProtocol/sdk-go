package cascade

import (
	"fmt"
	"strings"

	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gogoproto "github.com/cosmos/gogoproto/proto"
	chantypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
)

// ExtractRequestActionIDsFromAck decodes an IBC acknowledgement containing a TxMsgData
// and returns any MsgRequestActionResponse action IDs found in the message responses.
func ExtractRequestActionIDsFromAck(ackBytes []byte) ([]string, error) {
	var ack chantypes.Acknowledgement
	if err := chantypes.SubModuleCdc.UnmarshalJSON(ackBytes, &ack); err != nil {
		if err := gogoproto.Unmarshal(ackBytes, &ack); err != nil {
			return nil, err
		}
	}
	if ack.GetError() != "" {
		return nil, fmt.Errorf("ack error: %s", ack.GetError())
	}
	result := ack.GetResult()
	if len(result) == 0 {
		return nil, fmt.Errorf("ack result is empty")
	}
	var msgData sdk.TxMsgData
	if err := gogoproto.Unmarshal(result, &msgData); err != nil {
		return nil, err
	}
	ids := ExtractRequestActionIDsFromTxMsgData(&msgData)
	if len(ids) == 0 {
		return nil, fmt.Errorf("no action ids found in ack result")
	}
	return ids, nil
}

// ExtractRequestActionIDsFromTxMsgData scans TxMsgData for MsgRequestActionResponse
// entries and returns the corresponding action IDs.
func ExtractRequestActionIDsFromTxMsgData(msgData *sdk.TxMsgData) []string {
	var ids []string
	if msgData == nil {
		return ids
	}

	for _, any := range msgData.MsgResponses {
		if any == nil || any.TypeUrl == "" {
			continue
		}
		if !strings.HasSuffix(any.TypeUrl, "MsgRequestActionResponse") {
			continue
		}
		var resp actiontypes.MsgRequestActionResponse
		if err := gogoproto.Unmarshal(any.Value, &resp); err != nil {
			continue
		}
		if resp.ActionId != "" {
			ids = append(ids, resp.ActionId)
		}
	}

	if len(ids) > 0 {
		return ids
	}

	for _, data := range msgData.Data {
		if data == nil || data.MsgType == "" {
			continue
		}
		if !strings.HasSuffix(data.MsgType, "MsgRequestAction") {
			continue
		}
		var resp actiontypes.MsgRequestActionResponse
		if err := gogoproto.Unmarshal(data.Data, &resp); err != nil {
			continue
		}
		if resp.ActionId != "" {
			ids = append(ids, resp.ActionId)
		}
	}
	return ids
}
