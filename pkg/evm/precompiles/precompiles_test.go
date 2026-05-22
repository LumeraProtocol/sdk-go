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

func TestWasmABI_ExecuteIsPhase1NonPayable(t *testing.T) {
	execute := WasmABI.Methods["execute"]
	require.Equal(t, "nonpayable", execute.StateMutability)
	require.False(t, execute.Payable)
	require.Len(t, execute.Inputs, 2)
	require.Equal(t, "contractAddr", execute.Inputs[0].Name)
	require.Equal(t, "string", execute.Inputs[0].Type.String())
	require.Equal(t, "msg", execute.Inputs[1].Name)
	require.Equal(t, "bytes", execute.Inputs[1].Type.String())
}

func TestSupernodeABI_RegisterUsesStringAddresses(t *testing.T) {
	register := SupernodeABI.Methods["registerSupernode"]
	require.Len(t, register.Inputs, 4)
	for _, input := range register.Inputs {
		require.Equal(t, "string", input.Type.String(), "input %s should remain a bech32/string field", input.Name)
	}
}

func TestActionABI_FeeAmountsAreUint256(t *testing.T) {
	getActionFee := ActionABI.Methods["getActionFee"]
	require.Len(t, getActionFee.Outputs, 3)
	for _, output := range getActionFee.Outputs {
		require.Equal(t, "uint256", output.Type.String(), "output %s should remain an integer ulume amount", output.Name)
	}
}

func TestPackUnpack_RoundTrip(t *testing.T) {
	// getParams takes no args, returns one struct. Just verify Pack does
	// not error and produces a 4-byte selector.
	data, err := PackCall(ActionABI, "getParams")
	require.NoError(t, err)
	require.Len(t, data, 4)
}
