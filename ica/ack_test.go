package ica

import (
	"fmt"
	"testing"

	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gogoproto "github.com/cosmos/gogoproto/proto"
	chantypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"
)

func TestExtractRequestActionIDsFromTxMsgData(t *testing.T) {
	resp := actiontypes.MsgRequestActionResponse{ActionId: "action-1"}
	respBz, err := gogoproto.Marshal(&resp)
	require.NoError(t, err)

	any := &codectypes.Any{
		TypeUrl: "/lumera.action.v1.MsgRequestActionResponse",
		Value:   respBz,
	}
	msgData := &sdk.TxMsgData{MsgResponses: []*codectypes.Any{any}}

	ids := ExtractRequestActionIDsFromTxMsgData(msgData)
	require.Equal(t, []string{"action-1"}, ids)

	other := &codectypes.Any{
		TypeUrl: "/cosmos.bank.v1beta1.MsgSendResponse",
		Value:   []byte("noop"),
	}
	msgData = &sdk.TxMsgData{MsgResponses: []*codectypes.Any{other}}
	ids = ExtractRequestActionIDsFromTxMsgData(msgData)
	require.Empty(t, ids)
}

func TestExtractRequestActionIDsFromAck(t *testing.T) {
	resp := actiontypes.MsgRequestActionResponse{ActionId: "action-1"}
	respBz, err := gogoproto.Marshal(&resp)
	require.NoError(t, err)

	any := &codectypes.Any{
		TypeUrl: "/lumera.action.v1.MsgRequestActionResponse",
		Value:   respBz,
	}
	msgData := sdk.TxMsgData{MsgResponses: []*codectypes.Any{any}}
	msgDataBz, err := gogoproto.Marshal(&msgData)
	require.NoError(t, err)

	ack := chantypes.NewResultAcknowledgement(msgDataBz)
	ackJSON, err := chantypes.SubModuleCdc.MarshalJSON(&ack)
	require.NoError(t, err)

	ids, err := ExtractRequestActionIDsFromAck(ackJSON)
	require.NoError(t, err)
	require.Equal(t, []string{"action-1"}, ids)
}

func TestExtractRequestActionIDsFromAckErrors(t *testing.T) {
	ack := chantypes.NewErrorAcknowledgement(fmt.Errorf("boom"))
	ackJSON, err := chantypes.SubModuleCdc.MarshalJSON(&ack)
	require.NoError(t, err)

	_, err = ExtractRequestActionIDsFromAck(ackJSON)
	require.Error(t, err)

	ack = chantypes.NewResultAcknowledgement(nil)
	ackJSON, err = chantypes.SubModuleCdc.MarshalJSON(&ack)
	require.NoError(t, err)

	_, err = ExtractRequestActionIDsFromAck(ackJSON)
	require.Error(t, err)
}
