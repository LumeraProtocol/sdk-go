package ica

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	abcitypes "cosmossdk.io/api/cosmos/base/abci/v1beta1"
	txtypes "cosmossdk.io/api/cosmos/tx/v1beta1"
)

func extractPacketInfoFromTxResponse(tx *txtypes.GetTxResponse) (PacketInfo, error) {
	if tx == nil || tx.TxResponse == nil {
		return PacketInfo{}, fmt.Errorf("nil tx response")
	}
	for _, evt := range tx.TxResponse.GetEvents() {
		evtType := strings.TrimSpace(decodeEventValue(evt.GetType_()))
		if evtType == "" {
			evtType = evt.GetType_()
		}
		if evtType != "send_packet" {
			continue
		}
		attr := make(map[string]string)
		for _, a := range evt.GetAttributes() {
			key := strings.TrimSpace(decodeEventValue(a.GetKey()))
			val := strings.TrimSpace(decodeEventValue(a.GetValue()))
			if key != "" {
				attr[key] = val
			}
		}
		seqStr := attr["packet_sequence"]
		port := attr["packet_src_port"]
		channel := attr["packet_src_channel"]
		if seqStr == "" || port == "" || channel == "" {
			continue
		}
		seq, err := strconv.ParseUint(seqStr, 10, 64)
		if err != nil {
			return PacketInfo{}, err
		}
		return PacketInfo{Port: port, Channel: channel, Sequence: seq}, nil
	}
	return PacketInfo{}, ErrPacketInfoNotFound
}

func extractAcknowledgement(txs []*abcitypes.TxResponse, port, channel string, sequence uint64) ([]byte, error) {
	seqStr := strconv.FormatUint(sequence, 10)
	for _, tx := range txs {
		var ackHex string
		var ackB64 string
		var ackErr string
		var ackSuccess *bool
		for _, evt := range tx.GetEvents() {
			evtType := strings.TrimSpace(decodeEventValue(evt.GetType_()))
			if evtType == "" {
				evtType = evt.GetType_()
			}
			if evtType != "write_acknowledgement" {
				if strings.Contains(evtType, "ics27_packet") {
					attr := make(map[string]string)
					for _, a := range evt.GetAttributes() {
						key := strings.TrimSpace(decodeEventValue(a.GetKey()))
						val := strings.TrimSpace(decodeEventValue(a.GetValue()))
						if key != "" {
							attr[key] = val
						}
					}
					if v, ok := attr["success"]; ok {
						success := strings.EqualFold(v, "true")
						ackSuccess = &success
					}
					if v, ok := attr["ibccallbackerror-success"]; ok {
						success := strings.EqualFold(v, "true")
						ackSuccess = &success
					}
					if v := attr["error"]; v != "" {
						ackErr = v
					}
					if v := attr["ibccallbackerror-error"]; v != "" {
						ackErr = v
					}
				}
				continue
			}
			attr := make(map[string]string)
			for _, a := range evt.GetAttributes() {
				key := strings.TrimSpace(decodeEventValue(a.GetKey()))
				val := strings.TrimSpace(decodeEventValue(a.GetValue()))
				if key != "" {
					attr[key] = val
				}
			}
			if attr["packet_dst_port"] != port ||
				attr["packet_dst_channel"] != channel ||
				attr["packet_sequence"] != seqStr {
				continue
			}
			ackHex = attr["packet_ack_hex"]
			if ackHex == "" {
				ackB64 = attr["packet_ack"]
			}
		}
		if ackHex == "" && ackB64 == "" {
			continue
		}
		if ackSuccess != nil && !*ackSuccess {
			if ackErr != "" {
				return nil, fmt.Errorf("ica host ack error: %s", ackErr)
			}
			return nil, fmt.Errorf("ica host ack error: unknown failure")
		}
		if ackHex != "" {
			ack, err := hex.DecodeString(ackHex)
			if err != nil {
				return nil, fmt.Errorf("decode acknowledgement hex: %w", err)
			}
			return ack, nil
		}
		ack, err := base64.StdEncoding.DecodeString(ackB64)
		if err != nil {
			return nil, fmt.Errorf("decode acknowledgement base64: %w", err)
		}
		return ack, nil
	}
	return nil, ErrAckNotFound
}

func decodeEventValue(raw string) string {
	if raw == "" {
		return ""
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return raw
	}
	if !isMostlyPrintableASCII(decoded) {
		return raw
	}
	return string(decoded)
}

func isMostlyPrintableASCII(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	printable := 0
	for _, b := range data {
		if b == '\n' || b == '\r' || b == '\t' || (b >= 32 && b <= 126) {
			printable++
		}
	}
	return printable*100/len(data) >= 90
}
