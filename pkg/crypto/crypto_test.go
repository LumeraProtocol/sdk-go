package crypto

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LumeraProtocol/sdk-go/constants"
	sdkethsecp256k1 "github.com/LumeraProtocol/sdk-go/pkg/crypto/ethsecp256k1"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/go-bip39"
	"github.com/stretchr/testify/require"
)

var testMnemonic = func() string {
	entropy := make([]byte, 32)
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		panic(err)
	}
	return mnemonic
}()

// ---------------------------------------------------------------------------
// KeyType tests
// ---------------------------------------------------------------------------

func TestKeyType_String(t *testing.T) {
	require.Equal(t, "cosmos", KeyTypeCosmos.String())
	require.Equal(t, "evm", KeyTypeEVM.String())
}

func TestKeyType_HDPath(t *testing.T) {
	require.Equal(t, sdk.FullFundraiserPath, KeyTypeCosmos.HDPath())
	require.Equal(t, EVMBIP44HDPath, KeyTypeEVM.HDPath())
}

func TestKeyType_SigningAlgo(t *testing.T) {
	require.Equal(t, hd.Secp256k1.Name(), KeyTypeCosmos.SigningAlgo().Name())
	require.Equal(t, hd.PubKeyType(sdkethsecp256k1.KeyType), KeyTypeEVM.SigningAlgo().Name())
}

// ---------------------------------------------------------------------------
// Keyring creation
// ---------------------------------------------------------------------------

func TestDefaultKeyringParams(t *testing.T) {
	params := DefaultKeyringParams()
	require.Equal(t, "lumera", params.AppName)
	require.Equal(t, "os", params.Backend)
	if home, err := os.UserHomeDir(); err == nil {
		require.Equal(t, filepath.Join(home, ".lumera"), params.Dir)
	}
}

func TestNewKeyring(t *testing.T) {
	kr := newTestKeyring(t)
	require.NotNil(t, kr)
	_, err := kr.Key("missing")
	require.Error(t, err)
}

func TestNewKeyring_SupportsBothAlgos(t *testing.T) {
	kr := newTestKeyring(t)

	// Cosmos key
	_, err := kr.NewAccount("cosmos-key", testMnemonic, "", KeyTypeCosmos.HDPath(), KeyTypeCosmos.SigningAlgo())
	require.NoError(t, err)

	rec, err := kr.Key("cosmos-key")
	require.NoError(t, err)
	pk, err := rec.GetPubKey()
	require.NoError(t, err)
	require.Equal(t, string(hd.Secp256k1Type), pk.Type())

	// EVM key (different name, same mnemonic)
	_, err = kr.NewAccount("evm-key", testMnemonic, "", KeyTypeEVM.HDPath(), KeyTypeEVM.SigningAlgo())
	require.NoError(t, err)

	rec2, err := kr.Key("evm-key")
	require.NoError(t, err)
	pk2, err := rec2.GetPubKey()
	require.NoError(t, err)
	require.Equal(t, sdkethsecp256k1.KeyType, pk2.Type())
}

// ---------------------------------------------------------------------------
// Address derivation
// ---------------------------------------------------------------------------

func TestAddressFromKey(t *testing.T) {
	kr := newTestKeyring(t)
	_, err := kr.NewAccount("alice", testMnemonic, "", sdk.FullFundraiserPath, hd.Secp256k1)
	require.NoError(t, err)

	addr, err := AddressFromKey(kr, "alice", constants.LumeraAccountHRP)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(addr, constants.LumeraAccountHRP))

	_, err = AddressFromKey(nil, "alice", constants.LumeraAccountHRP)
	require.Error(t, err)
	_, err = AddressFromKey(kr, "", constants.LumeraAccountHRP)
	require.Error(t, err)
	_, err = AddressFromKey(kr, "missing", constants.LumeraAccountHRP)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Unified LoadKeyring / ImportKey
// ---------------------------------------------------------------------------

func TestLoadKeyring_Cosmos(t *testing.T) {
	mnemonicFile := writeMnemonicFile(t)
	kr, pub, addr, err := LoadKeyring("alice", mnemonicFile, KeyTypeCosmos)
	require.NoError(t, err)
	require.NotNil(t, kr)
	require.NotEmpty(t, pub)
	require.True(t, strings.HasPrefix(addr, constants.LumeraAccountHRP))

	rec, err := kr.Key("alice")
	require.NoError(t, err)
	pk, err := rec.GetPubKey()
	require.NoError(t, err)
	require.Equal(t, string(hd.Secp256k1Type), pk.Type())
}

func TestLoadKeyring_EVM(t *testing.T) {
	mnemonicFile := writeMnemonicFile(t)
	kr, pub, addr, err := LoadKeyring("alice", mnemonicFile, KeyTypeEVM)
	require.NoError(t, err)
	require.NotNil(t, kr)
	require.NotEmpty(t, pub)
	require.True(t, strings.HasPrefix(addr, constants.LumeraAccountHRP))

	rec, err := kr.Key("alice")
	require.NoError(t, err)
	pk, err := rec.GetPubKey()
	require.NoError(t, err)
	require.Equal(t, sdkethsecp256k1.KeyType, pk.Type())
}

func TestLoadKeyring_DifferentKeyTypesDerivesDifferentAddress(t *testing.T) {
	mnemonicFile := writeMnemonicFile(t)

	_, _, cosmosAddr, err := LoadKeyring("alice", mnemonicFile, KeyTypeCosmos)
	require.NoError(t, err)

	_, _, evmAddr, err := LoadKeyring("alice", mnemonicFile, KeyTypeEVM)
	require.NoError(t, err)

	require.NotEqual(t, cosmosAddr, evmAddr)
}

func TestImportKey_Cosmos(t *testing.T) {
	kr := newTestKeyring(t)
	mnemonicFile := writeMnemonicFile(t)

	pub, addr, err := ImportKey(kr, "alice", mnemonicFile, "cosmos", KeyTypeCosmos)
	require.NoError(t, err)
	require.NotEmpty(t, pub)
	require.True(t, strings.HasPrefix(addr, "cosmos"))

	// Idempotent: second import returns the same key.
	pub2, addr2, err := ImportKey(kr, "alice", mnemonicFile, "cosmos", KeyTypeCosmos)
	require.NoError(t, err)
	require.Equal(t, pub, pub2)
	require.Equal(t, addr, addr2)
}

func TestImportKey_EVM(t *testing.T) {
	kr := newTestKeyring(t)
	mnemonicFile := writeMnemonicFile(t)

	pub, addr, err := ImportKey(kr, "alice", mnemonicFile, "cosmos", KeyTypeEVM)
	require.NoError(t, err)
	require.NotEmpty(t, pub)
	require.True(t, strings.HasPrefix(addr, "cosmos"))

	rec, err := kr.Key("alice")
	require.NoError(t, err)
	pk, err := rec.GetPubKey()
	require.NoError(t, err)
	require.Equal(t, sdkethsecp256k1.KeyType, pk.Type())
}

// TestImportKey_MultiChain imports both Cosmos and EVM keys into a single
// keyring under different names (simulating independent controller/host
// chain configuration).
func TestImportKey_MultiChain(t *testing.T) {
	kr := newTestKeyring(t)
	mnemonicFile := writeMnemonicFile(t)

	// Controller chain: Cosmos key
	cosmosPub, cosmosAddr, err := ImportKey(kr, "controller", mnemonicFile, constants.LumeraAccountHRP, KeyTypeCosmos)
	require.NoError(t, err)
	require.NotEmpty(t, cosmosPub)

	// Host chain: EVM key
	evmPub, evmAddr, err := ImportKey(kr, "host", mnemonicFile, constants.LumeraAccountHRP, KeyTypeEVM)
	require.NoError(t, err)
	require.NotEmpty(t, evmPub)

	// Keys must differ (different algo + derivation path).
	require.NotEqual(t, cosmosPub, evmPub)
	require.NotEqual(t, cosmosAddr, evmAddr)

	// Verify key types stored in the keyring.
	cosmosRec, err := kr.Key("controller")
	require.NoError(t, err)
	cosmosPK, err := cosmosRec.GetPubKey()
	require.NoError(t, err)
	require.Equal(t, string(hd.Secp256k1Type), cosmosPK.Type())

	evmRec, err := kr.Key("host")
	require.NoError(t, err)
	evmPK, err := evmRec.GetPubKey()
	require.NoError(t, err)
	require.Equal(t, sdkethsecp256k1.KeyType, evmPK.Type())
}

// ---------------------------------------------------------------------------
// Behavior matrix
// ---------------------------------------------------------------------------

func TestKeyBehaviorMatrix(t *testing.T) {
	type keyMode struct {
		name        string
		keyType     KeyType
		expectedAlg string
	}

	mnemonicFile := writeMnemonicFile(t)
	modes := []keyMode{
		{
			name:        "cosmos",
			keyType:     KeyTypeCosmos,
			expectedAlg: string(hd.Secp256k1Type),
		},
		{
			name:        "evm",
			keyType:     KeyTypeEVM,
			expectedAlg: sdkethsecp256k1.KeyType,
		},
	}

	loadAddrs := make(map[string]string, len(modes))
	loadPubs := make(map[string][]byte, len(modes))

	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			// LoadKeyring should be deterministic and set the expected key algorithm.
			kr1, pub1, addr1, err := LoadKeyring("alice", mnemonicFile, mode.keyType)
			require.NoError(t, err)
			_, pub2, addr2, err := LoadKeyring("alice", mnemonicFile, mode.keyType)
			require.NoError(t, err)
			require.Equal(t, pub1, pub2)
			require.Equal(t, addr1, addr2)
			require.True(t, strings.HasPrefix(addr1, constants.LumeraAccountHRP))

			rec1, err := kr1.Key("alice")
			require.NoError(t, err)
			pk1, err := rec1.GetPubKey()
			require.NoError(t, err)
			require.Equal(t, mode.expectedAlg, pk1.Type())

			loadAddrs[mode.name] = addr1
			loadPubs[mode.name] = pub1

			// ImportKey should be deterministic in a single keyring.
			krImport := newTestKeyring(t)
			pubImp1, addrImp1, err := ImportKey(krImport, "alice", mnemonicFile, "cosmos", mode.keyType)
			require.NoError(t, err)
			pubImp2, addrImp2, err := ImportKey(krImport, "alice", mnemonicFile, "cosmos", mode.keyType)
			require.NoError(t, err)
			require.Equal(t, pubImp1, pubImp2)
			require.Equal(t, addrImp1, addrImp2)
			require.True(t, strings.HasPrefix(addrImp1, "cosmos"))

			recImp, err := krImport.Key("alice")
			require.NoError(t, err)
			pkImp, err := recImp.GetPubKey()
			require.NoError(t, err)
			require.Equal(t, mode.expectedAlg, pkImp.Type())

			// ImportKey with Lumera HRP should match LoadKeyring outputs.
			krLumera := newTestKeyring(t)
			pubLumera, addrLumera, err := ImportKey(krLumera, "alice", mnemonicFile, constants.LumeraAccountHRP, mode.keyType)
			require.NoError(t, err)
			require.Equal(t, pub1, pubLumera)
			require.Equal(t, addr1, addrLumera)
		})
	}

	require.NotEqual(t, loadAddrs["cosmos"], loadAddrs["evm"])
	require.NotEqual(t, loadPubs["cosmos"], loadPubs["evm"])
}

// ---------------------------------------------------------------------------
// Error paths
// ---------------------------------------------------------------------------

func TestLoadKeyring_Errors(t *testing.T) {
	mnemonicFile := writeMnemonicFile(t)

	// Empty key name.
	_, _, _, err := LoadKeyring("", mnemonicFile, KeyTypeCosmos)
	require.Error(t, err)
	require.Contains(t, err.Error(), "key name is required")

	// Non-existent mnemonic file.
	_, _, _, err = LoadKeyring("alice", "/no/such/file.txt", KeyTypeCosmos)
	require.Error(t, err)

	// Empty mnemonic file.
	emptyFile := filepath.Join(t.TempDir(), "empty.txt")
	require.NoError(t, os.WriteFile(emptyFile, []byte("   \n"), 0o600))
	_, _, _, err = LoadKeyring("alice", emptyFile, KeyTypeCosmos)
	require.Error(t, err)
	require.Contains(t, err.Error(), "mnemonic file is empty")
}

func TestImportKey_Errors(t *testing.T) {
	mnemonicFile := writeMnemonicFile(t)
	kr := newTestKeyring(t)

	// Nil keyring.
	_, _, err := ImportKey(nil, "alice", mnemonicFile, "cosmos", KeyTypeCosmos)
	require.Error(t, err)
	require.Contains(t, err.Error(), "keyring is nil")

	// Empty key name.
	_, _, err = ImportKey(kr, "", mnemonicFile, "cosmos", KeyTypeCosmos)
	require.Error(t, err)
	require.Contains(t, err.Error(), "key name is required")

	// Non-existent mnemonic file.
	_, _, err = ImportKey(kr, "alice", "/no/such/file.txt", "cosmos", KeyTypeCosmos)
	require.Error(t, err)
}

func TestAddressFromKey_Errors(t *testing.T) {
	kr := newTestKeyring(t)

	_, err := AddressFromKey(nil, "alice", "cosmos")
	require.Error(t, err)

	_, err = AddressFromKey(kr, "", "cosmos")
	require.Error(t, err)

	_, err = AddressFromKey(kr, "nonexistent", "cosmos")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// GetKey
// ---------------------------------------------------------------------------

func TestGetKey(t *testing.T) {
	kr := newTestKeyring(t)
	_, err := kr.NewAccount("bob", testMnemonic, "", KeyTypeCosmos.HDPath(), KeyTypeCosmos.SigningAlgo())
	require.NoError(t, err)

	rec, err := GetKey(kr, "bob")
	require.NoError(t, err)
	require.Equal(t, "bob", rec.Name)

	_, err = GetKey(kr, "missing")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Cross-HRP address derivation
// ---------------------------------------------------------------------------

func TestAddressFromKey_MultipleHRPs(t *testing.T) {
	kr := newTestKeyring(t)
	_, err := kr.NewAccount("alice", testMnemonic, "", KeyTypeCosmos.HDPath(), KeyTypeCosmos.SigningAlgo())
	require.NoError(t, err)

	lumeraAddr, err := AddressFromKey(kr, "alice", constants.LumeraAccountHRP)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(lumeraAddr, constants.LumeraAccountHRP+"1"))

	cosmosAddr, err := AddressFromKey(kr, "alice", "cosmos")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(cosmosAddr, "cosmos1"))

	osmosisAddr, err := AddressFromKey(kr, "alice", "osmo")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(osmosisAddr, "osmo1"))

	// Same key, different HRPs produce different bech32 strings but all are valid.
	require.NotEqual(t, lumeraAddr, cosmosAddr)
	require.NotEqual(t, cosmosAddr, osmosisAddr)
}

// ---------------------------------------------------------------------------
// NewKeyring with custom params
// ---------------------------------------------------------------------------

func TestNewKeyring_CustomParams(t *testing.T) {
	kr, err := NewKeyring(KeyringParams{
		AppName: "custom-app",
		Backend: "test",
		Dir:     t.TempDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, kr)

	// Keyring should support both key types.
	_, err = kr.NewAccount("c", testMnemonic, "", KeyTypeCosmos.HDPath(), KeyTypeCosmos.SigningAlgo())
	require.NoError(t, err)
}

func TestNewKeyring_DefaultsApplied(t *testing.T) {
	// All-zero params: defaults should be applied without panic.
	// We use "test" backend to avoid OS keyring interaction.
	kr, err := NewKeyring(KeyringParams{Backend: "test", Dir: t.TempDir()})
	require.NoError(t, err)
	require.NotNil(t, kr)
}

// ---------------------------------------------------------------------------
// Unified keyring sign-and-verify round trip
// ---------------------------------------------------------------------------

func TestUnifiedKeyring_SignVerifyRoundTrip(t *testing.T) {
	kr := newTestKeyring(t)
	msg := []byte("hello unified keyring")

	tests := []struct {
		name    string
		keyType KeyType
	}{
		{"cosmos", KeyTypeCosmos},
		{"evm", KeyTypeEVM},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyName := "sign-" + tt.name
			_, err := kr.NewAccount(keyName, testMnemonic, "", tt.keyType.HDPath(), tt.keyType.SigningAlgo())
			require.NoError(t, err)

			rec, err := kr.Key(keyName)
			require.NoError(t, err)
			pub, err := rec.GetPubKey()
			require.NoError(t, err)

			addr, err := rec.GetAddress()
			require.NoError(t, err)

			sig, _, err := kr.SignByAddress(addr, msg, 0)
			require.NoError(t, err)
			require.NotEmpty(t, sig)

			valid := pub.VerifySignature(msg, sig)
			require.True(t, valid, "signature verification should pass for %s key", tt.name)
		})
	}
}

// ---------------------------------------------------------------------------
// TxConfig and signing
// ---------------------------------------------------------------------------

func TestNewDefaultTxConfig(t *testing.T) {
	txCfg := NewDefaultTxConfig()
	require.NotNil(t, txCfg)

	builder := txCfg.NewTxBuilder()
	require.NotNil(t, builder)
}

func TestSignTxWithKeyring(t *testing.T) {
	kr := newTestKeyring(t)
	_, err := kr.NewAccount("alice", testMnemonic, "", sdk.FullFundraiserPath, hd.Secp256k1)
	require.NoError(t, err)

	txCfg := NewDefaultTxConfig()
	builder := txCfg.NewTxBuilder()
	err = SignTxWithKeyring(context.Background(), txCfg, kr, "alice", builder, "chain-id", 1, 0, false)
	require.NoError(t, err)

	sigs, err := builder.GetTx().GetSignaturesV2()
	require.NoError(t, err)
	require.Len(t, sigs, 1)
}

func TestSignTxWithKeyring_EVM(t *testing.T) {
	kr := newTestKeyring(t)
	_, err := kr.NewAccount("alice", testMnemonic, "", KeyTypeEVM.HDPath(), KeyTypeEVM.SigningAlgo())
	require.NoError(t, err)

	txCfg := NewDefaultTxConfig()
	builder := txCfg.NewTxBuilder()
	err = SignTxWithKeyring(context.Background(), txCfg, kr, "alice", builder, "chain-id", 1, 0, false)
	require.NoError(t, err)

	sigs, err := builder.GetTx().GetSignaturesV2()
	require.NoError(t, err)
	require.Len(t, sigs, 1)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestKeyring(t *testing.T) keyring.Keyring {
	t.Helper()
	kr, err := NewKeyring(KeyringParams{
		AppName: "lumera",
		Backend: "test",
		Dir:     t.TempDir(),
	})
	require.NoError(t, err)
	return kr
}

func writeMnemonicFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mnemonic.txt")
	require.NoError(t, os.WriteFile(path, []byte(testMnemonic), 0o600))
	return path
}
