package ica

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
)

// PacketInfo captures the packet identifiers needed to query acknowledgements.
type PacketInfo struct {
	Port     string
	Channel  string
	Sequence uint64
}

// ParseTxHashJSON extracts the tx hash from CLI JSON output and returns an error on failed codes.
func ParseTxHashJSON(txJSON []byte) (string, error) {
	var resp struct {
		TxHash string `json:"txhash"`
		Code   uint32 `json:"code"`
		RawLog string `json:"raw_log"`
	}
	if err := json.Unmarshal(txJSON, &resp); err == nil {
		if resp.Code != 0 {
			return "", fmt.Errorf("tx failed code=%d raw_log=%s", resp.Code, resp.RawLog)
		}
		if resp.TxHash != "" {
			return resp.TxHash, nil
		}
	}

	var nested struct {
		TxResponse struct {
			TxHash string `json:"txhash"`
			Code   uint32 `json:"code"`
			RawLog string `json:"raw_log"`
		} `json:"tx_response"`
	}
	if err := json.Unmarshal(txJSON, &nested); err != nil {
		return "", err
	}
	if nested.TxResponse.Code != 0 {
		return "", fmt.Errorf("tx failed code=%d raw_log=%s", nested.TxResponse.Code, nested.TxResponse.RawLog)
	}
	if nested.TxResponse.TxHash == "" {
		return "", fmt.Errorf("txhash missing from response")
	}
	return nested.TxResponse.TxHash, nil
}

// ExtractPacketInfoFromTxJSON parses a tx response JSON payload and returns the send_packet identifiers.
func ExtractPacketInfoFromTxJSON(txJSON []byte) (PacketInfo, error) {
	type eventAttr struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	type event struct {
		Type       string      `json:"type"`
		Attributes []eventAttr `json:"attributes"`
	}
	var resp struct {
		TxResponse struct {
			Events []event `json:"events"`
		} `json:"tx_response"`
		Events []event `json:"events"`
	}
	if err := json.Unmarshal(txJSON, &resp); err != nil {
		return PacketInfo{}, err
	}

	events := resp.TxResponse.Events
	if len(events) == 0 {
		events = resp.Events
	}

	for _, evt := range events {
		if evt.Type != "send_packet" {
			continue
		}
		attr := make(map[string]string)
		for _, a := range evt.Attributes {
			key := decodeEventValueBase64(a.Key)
			val := decodeEventValueBase64(a.Value)
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

	return PacketInfo{}, fmt.Errorf("send_packet event not found")
}

// DecodePacketAcknowledgementJSON extracts and base64-decodes the acknowledgement field.
func DecodePacketAcknowledgementJSON(ackJSON []byte) ([]byte, error) {
	var resp struct {
		Acknowledgement string `json:"acknowledgement"`
	}
	if err := json.Unmarshal(ackJSON, &resp); err != nil {
		return nil, err
	}
	if resp.Acknowledgement == "" {
		return nil, fmt.Errorf("empty acknowledgement")
	}
	ackBytes, err := base64.StdEncoding.DecodeString(resp.Acknowledgement)
	if err != nil {
		return nil, err
	}
	return ackBytes, nil
}

func decodeEventValueBase64(raw string) string {
	if raw == "" {
		return ""
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return raw
	}
	return string(decoded)
}
