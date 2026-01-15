package ica

import (
	"testing"

	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	gogoproto "github.com/cosmos/gogoproto/proto"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	"github.com/stretchr/testify/require"
)

func TestPackRequestForICA(t *testing.T) {
	_, err := PackRequestForICA(nil)
	require.Error(t, err)

	msg := &actiontypes.MsgRequestAction{
		Creator:    "creator",
		ActionType: "CASCADE",
		Metadata:   "{}",
		Price:      "1ulume",
	}
	anyBz, err := PackRequestForICA(msg)
	require.NoError(t, err)

	var any codectypes.Any
	require.NoError(t, gogoproto.Unmarshal(anyBz, &any))
	require.Contains(t, any.TypeUrl, "MsgRequestAction")

	var unpacked actiontypes.MsgRequestAction
	require.NoError(t, gogoproto.Unmarshal(any.Value, &unpacked))
	require.Equal(t, msg.Creator, unpacked.Creator)
	require.Equal(t, msg.ActionType, unpacked.ActionType)
}

func TestPackApproveForICA(t *testing.T) {
	_, err := PackApproveForICA(nil)
	require.Error(t, err)

	msg := &actiontypes.MsgApproveAction{
		Creator:  "creator",
		ActionId: "action-1",
	}
	anyBz, err := PackApproveForICA(msg)
	require.NoError(t, err)

	var any codectypes.Any
	require.NoError(t, gogoproto.Unmarshal(anyBz, &any))
	require.Contains(t, any.TypeUrl, "MsgApproveAction")

	var unpacked actiontypes.MsgApproveAction
	require.NoError(t, gogoproto.Unmarshal(any.Value, &unpacked))
	require.Equal(t, msg.ActionId, unpacked.ActionId)
}

func TestBuildICAPacketData(t *testing.T) {
	_, err := BuildICAPacketData(nil)
	require.Error(t, err)

	msg := &actiontypes.MsgRequestAction{
		Creator:    "creator",
		ActionType: "CASCADE",
		Metadata:   "{}",
	}
	any, err := codectypes.NewAnyWithValue(msg)
	require.NoError(t, err)

	packet, err := BuildICAPacketData([]*codectypes.Any{any})
	require.NoError(t, err)
	require.Equal(t, icatypes.EXECUTE_TX, packet.Type)

	var tx icatypes.CosmosTx
	require.NoError(t, gogoproto.Unmarshal(packet.Data, &tx))
	require.Len(t, tx.Messages, 1)
}

func TestBuildMsgSendTx(t *testing.T) {
	packet := icatypes.InterchainAccountPacketData{Type: icatypes.EXECUTE_TX}
	_, err := BuildMsgSendTx("", "conn-0", 1, packet)
	require.Error(t, err)
	_, err = BuildMsgSendTx("owner", "", 1, packet)
	require.Error(t, err)
	_, err = BuildMsgSendTx("owner", "conn-0", 0, packet)
	require.Error(t, err)

	msg, err := BuildMsgSendTx("owner", "conn-0", 10, packet)
	require.NoError(t, err)
	require.Equal(t, "owner", msg.Owner)
	require.Equal(t, "conn-0", msg.ConnectionId)
	require.Equal(t, uint64(10), msg.RelativeTimeout)
}
