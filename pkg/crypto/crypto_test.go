package crypto

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cosmos/go-bip39"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

func TestAddressFromKey(t *testing.T) {
	kr := newTestKeyring(t)
	_, err := kr.NewAccount("alice", testMnemonic, "", sdk.FullFundraiserPath, hd.Secp256k1)
	require.NoError(t, err)

	addr, err := AddressFromKey(kr, "alice", "lumera")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(addr, "lumera"))

	_, err = AddressFromKey(nil, "alice", "lumera")
	require.Error(t, err)
	_, err = AddressFromKey(kr, "", "lumera")
	require.Error(t, err)
	_, err = AddressFromKey(kr, "missing", "lumera")
	require.Error(t, err)
}

func TestLoadKeyringFromMnemonic(t *testing.T) {
	mnemonicFile := writeMnemonicFile(t)
	kr, pub, addr, err := LoadKeyringFromMnemonic("alice", mnemonicFile)
	require.NoError(t, err)
	require.NotNil(t, kr)
	require.NotEmpty(t, pub)
	require.True(t, strings.HasPrefix(addr, "lumera"))

	_, err = kr.Key("alice")
	require.NoError(t, err)
}

func TestImportKeyFromMnemonic(t *testing.T) {
	kr := newTestKeyring(t)
	mnemonicFile := writeMnemonicFile(t)

	pub, addr, err := ImportKeyFromMnemonic(kr, "alice", mnemonicFile, "cosmos")
	require.NoError(t, err)
	require.NotEmpty(t, pub)
	require.True(t, strings.HasPrefix(addr, "cosmos"))

	pub2, addr2, err := ImportKeyFromMnemonic(kr, "alice", mnemonicFile, "cosmos")
	require.NoError(t, err)
	require.Equal(t, addr, addr2)
	require.Equal(t, pub, pub2)
}

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
