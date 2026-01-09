package cascade

import (
	"fmt"

	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	gogoproto "github.com/cosmos/gogoproto/proto"
	controllertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
)

// PackRequestForICA packs a Lumera MsgRequestAction into protobuf Any bytes suitable for embedding
// into an ICS-27 controller transaction (MsgSendTx) on a remote chain.
//
// The returned bytes are the protobuf serialization of google.protobuf.Any, with type_url
// set to the Lumera MsgRequestAction URL and value set to the message bytes.
func PackRequestForICA(msg *actiontypes.MsgRequestAction) ([]byte, error) {
	if msg == nil {
		return nil, fmt.Errorf("msg is required")
	}
	val, err := gogoproto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal msg: %w", err)
	}
	any := &codectypes.Any{
		TypeUrl: "/" + gogoproto.MessageName(msg),
		Value:   val,
	}
	out, err := gogoproto.Marshal(any)
	if err != nil {
		return nil, fmt.Errorf("marshal Any: %w", err)
	}
	return out, nil
}

// PackRequestAny wraps PackRequestForICA and returns the decoded Any.
func PackRequestAny(msg *actiontypes.MsgRequestAction) (*codectypes.Any, error) {
	anyBytes, err := PackRequestForICA(msg)
	if err != nil {
		return nil, err
	}
	var any codectypes.Any
	if err := gogoproto.Unmarshal(anyBytes, &any); err != nil {
		return nil, fmt.Errorf("unmarshal Any: %w", err)
	}
	return &any, nil
}

// PackApproveForICA packs a Lumera MsgApproveAction into protobuf Any bytes suitable for ICS-27 MsgSendTx.
func PackApproveForICA(msg *actiontypes.MsgApproveAction) ([]byte, error) {
	if msg == nil {
		return nil, fmt.Errorf("msg is required")
	}
	val, err := gogoproto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal msg: %w", err)
	}
	any := &codectypes.Any{
		TypeUrl: "/" + gogoproto.MessageName(msg),
		Value:   val,
	}
	out, err := gogoproto.Marshal(any)
	if err != nil {
		return nil, fmt.Errorf("marshal Any: %w", err)
	}
	return out, nil
}

// PackApproveAny wraps PackApproveForICA and returns the decoded Any.
func PackApproveAny(msg *actiontypes.MsgApproveAction) (*codectypes.Any, error) {
	anyBytes, err := PackApproveForICA(msg)
	if err != nil {
		return nil, err
	}
	var any codectypes.Any
	if err := gogoproto.Unmarshal(anyBytes, &any); err != nil {
		return nil, fmt.Errorf("unmarshal Any: %w", err)
	}
	return &any, nil
}

// BuildICAPacketData builds InterchainAccountPacketData for EXECUTE_TX with provided Any messages.
func BuildICAPacketData(msgs []*codectypes.Any) (icatypes.InterchainAccountPacketData, error) {
	if len(msgs) == 0 {
		return icatypes.InterchainAccountPacketData{}, fmt.Errorf("at least one message is required")
	}
	// Build CosmosTx envelope with Anys and marshal to bytes
	tx := &icatypes.CosmosTx{Messages: msgs}
	data, err := gogoproto.Marshal(tx)
	if err != nil {
		return icatypes.InterchainAccountPacketData{}, fmt.Errorf("marshal CosmosTx: %w", err)
	}
	return icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: data,
		Memo: "",
	}, nil
}

// BuildMsgSendTx constructs the controller-side MsgSendTx using provided owner/connection and packet data.
func BuildMsgSendTx(owner, connectionID string, relativeTimeout uint64, packet icatypes.InterchainAccountPacketData) (*controllertypes.MsgSendTx, error) {
	if owner == "" {
		return nil, fmt.Errorf("owner is required")
	}
	if connectionID == "" {
		return nil, fmt.Errorf("connection_id is required")
	}
	if relativeTimeout == 0 {
		return nil, fmt.Errorf("relative_timeout must be non-zero")
	}
	return &controllertypes.MsgSendTx{
		Owner:           owner,
		ConnectionId:    connectionID,
		PacketData:      packet,
		RelativeTimeout: relativeTimeout,
	}, nil
}
