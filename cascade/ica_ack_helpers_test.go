package cascade

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseTxHashJSON(t *testing.T) {
	txJSON := []byte(`{"txhash":"ABC","code":0}`)
	hash, err := ParseTxHashJSON(txJSON)
	require.NoError(t, err)
	require.Equal(t, "ABC", hash)

	txJSON = []byte(`{"tx_response":{"txhash":"DEF","code":0}}`)
	hash, err = ParseTxHashJSON(txJSON)
	require.NoError(t, err)
	require.Equal(t, "DEF", hash)

	txJSON = []byte(`{"txhash":"ERR","code":1,"raw_log":"boom"}`)
	_, err = ParseTxHashJSON(txJSON)
	require.Error(t, err)
}

func TestExtractPacketInfoFromTxJSON(t *testing.T) {
	attrs := []map[string]string{
		{
			"key":   base64.StdEncoding.EncodeToString([]byte("packet_sequence")),
			"value": base64.StdEncoding.EncodeToString([]byte("10")),
		},
		{
			"key":   base64.StdEncoding.EncodeToString([]byte("packet_src_port")),
			"value": base64.StdEncoding.EncodeToString([]byte("icacontroller")),
		},
		{
			"key":   base64.StdEncoding.EncodeToString([]byte("packet_src_channel")),
			"value": base64.StdEncoding.EncodeToString([]byte("channel-0")),
		},
	}
	payload := map[string]any{
		"tx_response": map[string]any{
			"events": []map[string]any{
				{"type": "send_packet", "attributes": attrs},
			},
		},
	}
	raw, err := json.Marshal(payload)
	require.NoError(t, err)

	info, err := ExtractPacketInfoFromTxJSON(raw)
	require.NoError(t, err)
	require.Equal(t, uint64(10), info.Sequence)
	require.Equal(t, "icacontroller", info.Port)
	require.Equal(t, "channel-0", info.Channel)

	_, err = ExtractPacketInfoFromTxJSON([]byte(`{"tx_response":{"events":[]}}`))
	require.Error(t, err)
}

func TestDecodePacketAcknowledgementJSON(t *testing.T) {
	ack := []byte("ack-bytes")
	ackJSON := []byte(`{"acknowledgement":"` + base64.StdEncoding.EncodeToString(ack) + `"}`)

	out, err := DecodePacketAcknowledgementJSON(ackJSON)
	require.NoError(t, err)
	require.Equal(t, ack, out)

	_, err = DecodePacketAcknowledgementJSON([]byte(`{"acknowledgement":""}`))
	require.Error(t, err)
}
