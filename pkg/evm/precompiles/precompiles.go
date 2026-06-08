// Package precompiles exposes Lumera and standard cosmos/evm precompile
// addresses, pre-parsed ABIs, and generic call helpers so SDK consumers can
// invoke precompile methods from Go without compiling Solidity themselves.
//
// Three Lumera-specific precompiles are wrapped:
//
//   - Action     0x0901  exposes x/action (Cascade, Sense)
//   - Supernode  0x0902  exposes x/supernode lifecycle and metrics
//   - Wasm       0x0903  enables bidirectional CosmWasm <-> EVM calls
//
// The eight standard cosmos/evm precompiles are exposed as named address
// constants only; their Solidity-facing surface is documented in the lumera
// repo under docs/evm-integration/precompiles.
package precompiles

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// Lumera custom precompile addresses (per precompiles/precompiles.md).
var (
	ActionAddress    = common.HexToAddress("0x0000000000000000000000000000000000000901")
	SupernodeAddress = common.HexToAddress("0x0000000000000000000000000000000000000902")
	WasmAddress      = common.HexToAddress("0x0000000000000000000000000000000000000903")
)

// Standard cosmos/evm precompile addresses (per precompiles/standard-precompiles.md).
var (
	P256Address         = common.HexToAddress("0x0000000000000000000000000000000000000100")
	Bech32Address       = common.HexToAddress("0x0000000000000000000000000000000000000400")
	StakingAddress      = common.HexToAddress("0x0000000000000000000000000000000000000800")
	DistributionAddress = common.HexToAddress("0x0000000000000000000000000000000000000801")
	ICS20Address        = common.HexToAddress("0x0000000000000000000000000000000000000802")
	BankAddress         = common.HexToAddress("0x0000000000000000000000000000000000000804")
	GovAddress          = common.HexToAddress("0x0000000000000000000000000000000000000805")
	SlashingAddress     = common.HexToAddress("0x0000000000000000000000000000000000000806")
)

//go:embed abi/action.json
var actionABIJSON []byte

//go:embed abi/supernode.json
var supernodeABIJSON []byte

//go:embed abi/wasm.json
var wasmABIJSON []byte

// Pre-parsed ABIs. Initialized at package load; init() panics if the embedded
// JSON is malformed (which would be a build-time error caught by tests).
var (
	ActionABI    abi.ABI
	SupernodeABI abi.ABI
	WasmABI      abi.ABI
)

func init() {
	var err error
	if ActionABI, err = parseHardhatABI(actionABIJSON); err != nil {
		panic(fmt.Sprintf("parse action precompile ABI: %v", err))
	}
	if SupernodeABI, err = parseHardhatABI(supernodeABIJSON); err != nil {
		panic(fmt.Sprintf("parse supernode precompile ABI: %v", err))
	}
	if WasmABI, err = parseHardhatABI(wasmABIJSON); err != nil {
		panic(fmt.Sprintf("parse wasm precompile ABI: %v", err))
	}
}

// parseHardhatABI extracts the `abi` field from a Hardhat artifact JSON and
// parses it via go-ethereum's abi package. Falls back to direct parse if the
// payload is a bare ABI array.
func parseHardhatABI(raw []byte) (abi.ABI, error) {
	var artifact struct {
		ABI json.RawMessage `json:"abi"`
	}
	// A Hardhat artifact has a non-null "abi" field. A bare ABI array
	// unmarshals into the struct without populating ABI, so it falls through
	// to the raw parse below. Guard against an explicit null so we don't try
	// to parse the literal "null" as an ABI.
	if err := json.Unmarshal(raw, &artifact); err == nil &&
		len(artifact.ABI) > 0 && string(artifact.ABI) != "null" {
		return abi.JSON(strings.NewReader(string(artifact.ABI)))
	}
	return abi.JSON(strings.NewReader(string(raw)))
}

// PackCall produces ABI-encoded calldata for a precompile method. Pass the
// method args in the order the Solidity interface declares them.
func PackCall(parsed abi.ABI, method string, args ...any) ([]byte, error) {
	data, err := parsed.Pack(method, args...)
	if err != nil {
		return nil, fmt.Errorf("pack %s: %w", method, err)
	}
	return data, nil
}

// UnpackReturn decodes the raw return bytes of a precompile call into the
// declared output types. Returns the slice of values matching the Solidity
// outputs (single return: out[0]).
func UnpackReturn(parsed abi.ABI, method string, ret []byte) ([]any, error) {
	out, err := parsed.Unpack(method, ret)
	if err != nil {
		return nil, fmt.Errorf("unpack %s: %w", method, err)
	}
	return out, nil
}
