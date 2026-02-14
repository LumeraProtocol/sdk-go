package crypto

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/LumeraProtocol/sdk-go/constants"
	sdkethsecp256k1 "github.com/LumeraProtocol/sdk-go/pkg/crypto/ethsecp256k1"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"

	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
)

const (
	// EVMBIP44HDPath is the default Ethereum derivation path (coin type 60).
	EVMBIP44HDPath = "m/44'/60'/0'/0/0"
)

// KeyType represents the cryptographic key algorithm and HD derivation path
// to use for a chain. Controller and host chains can each be configured with
// an independent KeyType.
type KeyType int

const (
	// KeyTypeCosmos uses secp256k1 with BIP44 coin type 118 (standard Cosmos).
	KeyTypeCosmos KeyType = iota
	// KeyTypeEVM uses eth_secp256k1 with BIP44 coin type 60 (Ethereum-compatible).
	KeyTypeEVM
)

// String returns the string representation of the key type.
func (kt KeyType) String() string {
	switch kt {
	case KeyTypeEVM:
		return "evm"
	default:
		return "cosmos"
	}
}

// HDPath returns the BIP44 HD derivation path for this key type.
func (kt KeyType) HDPath() string {
	switch kt {
	case KeyTypeEVM:
		return EVMBIP44HDPath
	default:
		return sdk.FullFundraiserPath
	}
}

// SigningAlgo returns the keyring signing algorithm for this key type.
func (kt KeyType) SigningAlgo() keyring.SignatureAlgo {
	switch kt {
	case KeyTypeEVM:
		return ethSecp256k1Alg
	default:
		return hd.Secp256k1
	}
}

var (
	ethSecp256k1Alg = ethSecp256k1Algo{}
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

// NewKeyring creates a new keyring that supports both Cosmos (secp256k1)
// and EVM (eth_secp256k1) key types. The key type used is determined when
// importing or creating keys, not at keyring creation time.
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

	registry := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(registry)
	sdkethsecp256k1.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	return keyring.New(app, backend, dir, in, cdc, ethSecp256k1Option())
}

// GetKey returns metadata for the named key in the provided keyring.
func GetKey(kr keyring.Keyring, keyName string) (*keyring.Record, error) {
	return kr.Key(keyName)
}

// LoadKeyring creates a test keyring in a temporary directory under
// os.TempDir(), imports the mnemonic using the specified key type, and returns
// the keyring, pubkey bytes, and Lumera address.
//
// The temporary directory is cleaned up by the OS on reboot. For production
// use, prefer NewKeyring with an explicit directory and import keys via
// kr.NewAccount directly.
func LoadKeyring(keyName, mnemonicFile string, keyType KeyType) (keyring.Keyring, []byte, string, error) {
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
	ok := false
	defer func() {
		if !ok {
			_ = os.RemoveAll(krDir)
		}
	}()

	kr, err := NewKeyring(KeyringParams{
		AppName: "lumera",
		Backend: "test",
		Dir:     krDir,
		Input:   strings.NewReader(""),
	})
	if err != nil {
		return nil, nil, "", fmt.Errorf("create keyring: %w", err)
	}
	if _, err := kr.NewAccount(keyName, mnemonic, "", keyType.HDPath(), keyType.SigningAlgo()); err != nil {
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

	ok = true
	return kr, pub.Bytes(), addr, nil
}

// ImportKey imports a mnemonic into an existing keyring using the specified
// key type, returning the pubkey bytes and address for the provided HRP.
//
// If a key with the same name already exists, ImportKey verifies that its
// algorithm matches the requested keyType and returns an error on mismatch.
func ImportKey(kr keyring.Keyring, keyName, mnemonicFile, hrp string, keyType KeyType) ([]byte, string, error) {
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

	existing, err := kr.Key(keyName)
	if err != nil {
		// Key does not exist — import it.
		if _, err := kr.NewAccount(keyName, mnemonic, "", keyType.HDPath(), keyType.SigningAlgo()); err != nil {
			return nil, "", fmt.Errorf("import key: %w", err)
		}
	} else {
		// Key exists — verify the algorithm matches the requested key type.
		pub, err := existing.GetPubKey()
		if err != nil {
			return nil, "", fmt.Errorf("get existing pubkey: %w", err)
		}
		wantAlgo := string(keyType.SigningAlgo().Name())
		if pub.Type() != wantAlgo {
			return nil, "", fmt.Errorf("key %q already exists with algorithm %s, but %s (%s) was requested",
				keyName, pub.Type(), keyType.String(), wantAlgo)
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
	sdkethsecp256k1.RegisterInterfaces(reg)
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

func ethSecp256k1Option() keyring.Option {
	return func(options *keyring.Options) {
		options.SupportedAlgos = keyring.SigningAlgoList{ethSecp256k1Alg, hd.Secp256k1}
		options.SupportedAlgosLedger = keyring.SigningAlgoList{ethSecp256k1Alg, hd.Secp256k1}
	}
}

type ethSecp256k1Algo struct{}

func (s ethSecp256k1Algo) Name() hd.PubKeyType {
	return hd.PubKeyType(sdkethsecp256k1.KeyType)
}

func (s ethSecp256k1Algo) Derive() hd.DeriveFn {
	// Reuse Cosmos derivation function with Ethereum BIP44 path.
	return hd.Secp256k1.Derive()
}

func (s ethSecp256k1Algo) Generate() hd.GenerateFn {
	return func(bz []byte) cryptotypes.PrivKey {
		bzArr := make([]byte, sdkethsecp256k1.PrivKeySize)
		copy(bzArr, bz)
		return &sdkethsecp256k1.PrivKey{Key: bzArr}
	}
}
