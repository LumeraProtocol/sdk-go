package precompiles

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmbeddedABIs_Parse(t *testing.T) {
	// init() panics on parse failure; reaching here confirms all three
	// ABIs decoded. Sanity-check that each contains at least one method.
	require.NotEmpty(t, ActionABI.Methods, "action ABI has no methods")
	require.NotEmpty(t, SupernodeABI.Methods, "supernode ABI has no methods")
	require.NotEmpty(t, WasmABI.Methods, "wasm ABI has no methods")
}

func TestAddresses(t *testing.T) {
	require.Equal(t, "0x0000000000000000000000000000000000000901", ActionAddress.Hex())
	require.Equal(t, "0x0000000000000000000000000000000000000902", SupernodeAddress.Hex())
	require.Equal(t, "0x0000000000000000000000000000000000000903", WasmAddress.Hex())

	require.Equal(t, "0x0000000000000000000000000000000000000800", StakingAddress.Hex())
	require.Equal(t, "0x0000000000000000000000000000000000000804", BankAddress.Hex())
}

func TestActionABI_HasCoreMethods(t *testing.T) {
	for _, m := range []string{"approveAction", "getAction", "getActionFee", "getParams"} {
		if _, ok := ActionABI.Methods[m]; !ok {
			t.Errorf("ActionABI missing method %q", m)
		}
	}
}

func TestSupernodeABI_HasCoreMethods(t *testing.T) {
	for _, m := range []string{"registerSupernode", "getSuperNode", "listSuperNodes", "getParams"} {
		if _, ok := SupernodeABI.Methods[m]; !ok {
			t.Errorf("SupernodeABI missing method %q", m)
		}
	}
}

func TestWasmABI_HasCoreMethods(t *testing.T) {
	for _, m := range []string{"execute", "query", "contractInfo", "rawQuery"} {
		if _, ok := WasmABI.Methods[m]; !ok {
			t.Errorf("WasmABI missing method %q", m)
		}
	}
}

func TestPackUnpack_RoundTrip(t *testing.T) {
	// getParams takes no args, returns one struct. Just verify Pack does
	// not error and produces a 4-byte selector.
	data, err := PackCall(ActionABI, "getParams")
	require.NoError(t, err)
	require.Len(t, data, 4)
}
