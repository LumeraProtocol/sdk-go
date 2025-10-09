package crypto

import (
	"bufio"
	"io"
	"os"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/std"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"

	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
)

// KeyringParams holds configuration for initializing a Cosmos keyring.
type KeyringParams struct {
	// AppName names the keyring namespace. Default: "lumera"
	AppName string
	// Backend selects the keyring backend ("os" | "file" | "test"). Default: "os"
	Backend string
	// Dir is the root directory for the keyring (if Backend="file"). Default: $HOME/.lumera
	Dir string
	// Input is an optional io.Reader for interactive backends (nil for non-interactive)
	Input io.Reader
}

// DefaultKeyringParams returns sensible defaults:
//   - AppName: "lumera"
//   - Backend: "os"
//   - Dir: $HOME/.lumera
func DefaultKeyringParams() KeyringParams {
	home, _ := os.UserHomeDir()
	return KeyringParams{
		AppName: "lumera",
		Backend: "os",
		Dir:     filepath.Join(home, ".lumera"),
		Input:   nil,
	}
}

// NewKeyring creates a new Cosmos keyring with the provided parameters.
func NewKeyring(p KeyringParams) (keyring.Keyring, error) {
	app := p.AppName
	if app == "" {
		app = "lumera"
	}
	backend := p.Backend
	if backend == "" {
		backend = "os"
	}
	dir := p.Dir
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".lumera")
	}
	in := p.Input
	if in == nil {
		in = bufio.NewReader(os.Stdin)
	}

	// Create a proto codec for keyring operations
	reg := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(reg)
	cdc := codec.NewProtoCodec(reg)

	return keyring.New(app, backend, dir, in, cdc)
}

// GetKey returns metadata for the named key in the provided keyring.
func GetKey(kr keyring.Keyring, keyName string) (*keyring.Record, error) {
	return kr.Key(keyName)
}

// NewDefaultTxConfig constructs a client.TxConfig backed by a protobuf codec,
// registering Lumera action message interfaces as required for signing/encoding.
func NewDefaultTxConfig() client.TxConfig {
	reg := codectypes.NewInterfaceRegistry()
	// Register crypto and module interfaces
	cryptocodec.RegisterInterfaces(reg)
	actiontypes.RegisterInterfaces(reg)

	proto := codec.NewProtoCodec(reg)
	return authtx.NewTxConfig(proto, authtx.DefaultSignModes)
}
