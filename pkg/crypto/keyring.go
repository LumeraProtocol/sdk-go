package crypto

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/LumeraProtocol/sdk-go/constants"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

// LoadKeyringFromMnemonic creates a test keyring, imports the mnemonic, and
// returns the keyring, pubkey bytes, and Lumera address.
func LoadKeyringFromMnemonic(keyName, mnemonicFile string) (keyring.Keyring, []byte, string, error) {
	if keyName == "" {
		return nil, nil, "", fmt.Errorf("key name is required")
	}
	mnemonic, err := readMnemonicFile(mnemonicFile)
	if err != nil {
		return nil, nil, "", err
	}

	krDir, err := os.MkdirTemp("", "lumera-keyring-*")
	if err != nil {
		return nil, nil, "", fmt.Errorf("create keyring dir: %w", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	krCodec := codec.NewProtoCodec(registry)
	kr, err := keyring.New("lumera", "test", krDir, strings.NewReader(""), krCodec)
	if err != nil {
		return nil, nil, "", fmt.Errorf("create keyring: %w", err)
	}
	if _, err := kr.NewAccount(keyName, mnemonic, "", sdk.FullFundraiserPath, hd.Secp256k1); err != nil {
		return nil, nil, "", fmt.Errorf("import key: %w", err)
	}

	addr, err := AddressFromKey(kr, keyName, constants.LumeraAccountHRP)
	if err != nil {
		return nil, nil, "", fmt.Errorf("derive address: %w", err)
	}

	rec, err := kr.Key(keyName)
	if err != nil {
		return nil, nil, "", fmt.Errorf("load key: %w", err)
	}
	pub, err := rec.GetPubKey()
	if err != nil {
		return nil, nil, "", fmt.Errorf("get pubkey: %w", err)
	}
	if pub == nil {
		return nil, nil, "", fmt.Errorf("pubkey is nil")
	}

	return kr, pub.Bytes(), addr, nil
}

// ImportKeyFromMnemonic imports the mnemonic into an existing keyring (if needed),
// returning the pubkey bytes and address for the provided HRP.
func ImportKeyFromMnemonic(kr keyring.Keyring, keyName, mnemonicFile, hrp string) ([]byte, string, error) {
	if kr == nil {
		return nil, "", fmt.Errorf("keyring is nil")
	}
	if keyName == "" {
		return nil, "", fmt.Errorf("key name is required")
	}
	mnemonic, err := readMnemonicFile(mnemonicFile)
	if err != nil {
		return nil, "", err
	}

	if _, err := kr.Key(keyName); err != nil {
		if _, err := kr.NewAccount(keyName, mnemonic, "", sdk.FullFundraiserPath, hd.Secp256k1); err != nil {
			return nil, "", fmt.Errorf("import key: %w", err)
		}
	}

	addr, err := AddressFromKey(kr, keyName, hrp)
	if err != nil {
		return nil, "", fmt.Errorf("derive address: %w", err)
	}
	rec, err := kr.Key(keyName)
	if err != nil {
		return nil, "", fmt.Errorf("load key: %w", err)
	}
	pub, err := rec.GetPubKey()
	if err != nil {
		return nil, "", fmt.Errorf("get pubkey: %w", err)
	}
	if pub == nil {
		return nil, "", fmt.Errorf("pubkey is nil")
	}
	return pub.Bytes(), addr, nil
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

func readMnemonicFile(mnemonicFile string) (string, error) {
	mnemonicRaw, err := os.ReadFile(mnemonicFile)
	if err != nil {
		return "", fmt.Errorf("read mnemonic file: %w", err)
	}
	mnemonic := strings.TrimSpace(string(mnemonicRaw))
	if mnemonic == "" {
		return "", fmt.Errorf("mnemonic file is empty")
	}
	return mnemonic, nil
}
