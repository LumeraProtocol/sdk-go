package ica

import (
	"encoding/base64"
	"encoding/hex"
	"testing"

	abcitypes "cosmossdk.io/api/cosmos/base/abci/v1beta1"
	txtypes "cosmossdk.io/api/cosmos/tx/v1beta1"
	abci "cosmossdk.io/api/tendermint/abci"
)

func TestExtractPacketInfoFromTxResponse(t *testing.T) {
	attrs := []*abci.EventAttribute{
		{Key: base64.StdEncoding.EncodeToString([]byte("packet_src_port")), Value: base64.StdEncoding.EncodeToString([]byte("icacontroller-0"))},
		{Key: base64.StdEncoding.EncodeToString([]byte("packet_src_channel")), Value: base64.StdEncoding.EncodeToString([]byte("channel-7"))},
		{Key: base64.StdEncoding.EncodeToString([]byte("packet_sequence")), Value: base64.StdEncoding.EncodeToString([]byte("42"))},
	}
	tx := &txtypes.GetTxResponse{TxResponse: &abcitypes.TxResponse{Events: []*abci.Event{{Type_: "send_packet", Attributes: attrs}}}}

	info, err := extractPacketInfoFromTxResponse(tx)
	if err != nil {
		t.Fatalf("extract packet info: %v", err)
	}
	if info.Port != "icacontroller-0" || info.Channel != "channel-7" || info.Sequence != 42 {
		t.Fatalf("unexpected packet info: %+v", info)
	}
}

func TestExtractAcknowledgementHex(t *testing.T) {
	ack := []byte{0x01, 0x02, 0x03}
	attrs := []*abci.EventAttribute{
		{Key: "packet_dst_port", Value: "icahost"},
		{Key: "packet_dst_channel", Value: "channel-9"},
		{Key: "packet_sequence", Value: "7"},
		{Key: "packet_ack_hex", Value: hex.EncodeToString(ack)},
	}
	tx := &abcitypes.TxResponse{Events: []*abci.Event{{Type_: "write_acknowledgement", Attributes: attrs}}}

	got, err := extractAcknowledgement([]*abcitypes.TxResponse{tx}, "icahost", "channel-9", 7)
	if err != nil {
		t.Fatalf("extract acknowledgement: %v", err)
	}
	if hex.EncodeToString(got) != hex.EncodeToString(ack) {
		t.Fatalf("ack mismatch: got %x want %x", got, ack)
	}
}

func TestExtractAcknowledgementBase64(t *testing.T) {
	ack := []byte{0x0a, 0x0b}
	attrs := []*abci.EventAttribute{
		{Key: "packet_dst_port", Value: "icahost"},
		{Key: "packet_dst_channel", Value: "channel-2"},
		{Key: "packet_sequence", Value: "3"},
		{Key: "packet_ack", Value: base64.StdEncoding.EncodeToString(ack)},
	}
	tx := &abcitypes.TxResponse{Events: []*abci.Event{{Type_: "write_acknowledgement", Attributes: attrs}}}

	got, err := extractAcknowledgement([]*abcitypes.TxResponse{tx}, "icahost", "channel-2", 3)
	if err != nil {
		t.Fatalf("extract acknowledgement: %v", err)
	}
	if hex.EncodeToString(got) != hex.EncodeToString(ack) {
		t.Fatalf("ack mismatch: got %x want %x", got, ack)
	}
}
