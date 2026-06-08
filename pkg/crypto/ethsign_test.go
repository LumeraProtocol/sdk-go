package crypto

import (
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

const evmSignTestMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
const evmSignMismatchMnemonic = "legal winner thank year wave sausage worth useful legal winner thank yellow"

func writeMnemonic(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "mnemonic.txt")
	require.NoError(t, os.WriteFile(path, []byte(evmSignTestMnemonic), 0o600))
	return path
}

func TestSignEthereumTx_RoundTrip(t *testing.T) {
	mnemonicFile := writeMnemonic(t)
	kr, _, _, err := LoadKeyring("alice", mnemonicFile, KeyTypeEVM)
	require.NoError(t, err)

	from, err := EVMAddressFromKey(kr, "alice")
	require.NoError(t, err)

	chainID := big.NewInt(1414)
	to := common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	tx := ethtypes.NewTx(&ethtypes.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     0,
		GasTipCap: big.NewInt(1_000_000_000),
		GasFeeCap: big.NewInt(2_000_000_000),
		Gas:       21_000,
		To:        &to,
		Value:     big.NewInt(0),
		Data:      nil,
	})

	signed, err := SignEthereumTx(kr, "alice", chainID, tx)
	require.NoError(t, err)
	require.NotNil(t, signed)
	require.Equal(t, chainID, signed.ChainId())

	recovered, err := RecoverSender(signed)
	require.NoError(t, err)
	require.Equal(t, strings.ToLower(from), strings.ToLower(recovered.Hex()))
}

type mismatchedSignerKeyring struct {
	keyring.Keyring
	reported *keyring.Record
	signer   keyring.Keyring
	signAs   string
}

func (k mismatchedSignerKeyring) Key(string) (*keyring.Record, error) {
	return k.reported, nil
}

func (k mismatchedSignerKeyring) Sign(_ string, msg []byte, mode signingtypes.SignMode) ([]byte, cryptotypes.PubKey, error) {
	sig, _, err := k.signer.Sign(k.signAs, msg, mode)
	return sig, nil, err
}

func TestSignEthereumTx_RejectsRecoveredSenderMismatch(t *testing.T) {
	mnemonicFile := writeMnemonic(t)
	alice, _, _, err := LoadKeyring("alice", mnemonicFile, KeyTypeEVM)
	require.NoError(t, err)
	aliceRec, err := alice.Key("alice")
	require.NoError(t, err)

	bob, err := NewKeyring(KeyringParams{
		AppName: "lumera",
		Backend: "test",
		Dir:     t.TempDir(),
		Input:   strings.NewReader(""),
	})
	require.NoError(t, err)
	_, err = bob.NewAccount("bob", evmSignMismatchMnemonic, "", KeyTypeEVM.HDPath(), KeyTypeEVM.SigningAlgo())
	require.NoError(t, err)

	chainID := big.NewInt(1414)
	tx := ethtypes.NewTx(&ethtypes.DynamicFeeTx{
		ChainID:   chainID,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(2),
		Gas:       21_000,
	})

	_, err = SignEthereumTx(mismatchedSignerKeyring{
		reported: aliceRec,
		signer:   bob,
		signAs:   "bob",
	}, "alice", chainID, tx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "recovered sender")
}

func TestSignEthereumTx_RejectsCosmosKey(t *testing.T) {
	mnemonicFile := writeMnemonic(t)
	kr, _, _, err := LoadKeyring("bob", mnemonicFile, KeyTypeCosmos)
	require.NoError(t, err)

	chainID := big.NewInt(1414)
	tx := ethtypes.NewTx(&ethtypes.DynamicFeeTx{ChainID: chainID, Gas: 21_000})

	_, err = SignEthereumTx(kr, "bob", chainID, tx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "EVM signing requires")
}

func TestSignEthereumTx_RequiresPositiveChainID(t *testing.T) {
	mnemonicFile := writeMnemonic(t)
	kr, _, _, err := LoadKeyring("alice", mnemonicFile, KeyTypeEVM)
	require.NoError(t, err)

	tx := ethtypes.NewTx(&ethtypes.DynamicFeeTx{})
	_, err = SignEthereumTx(kr, "alice", nil, tx)
	require.Error(t, err)

	_, err = SignEthereumTx(kr, "alice", big.NewInt(0), tx)
	require.Error(t, err)
}

func TestWrapAsMsgEthereumTx(t *testing.T) {
	mnemonicFile := writeMnemonic(t)
	kr, _, _, err := LoadKeyring("alice", mnemonicFile, KeyTypeEVM)
	require.NoError(t, err)

	from, err := EVMAddressFromKey(kr, "alice")
	require.NoError(t, err)

	chainID := big.NewInt(1414)
	to := common.HexToAddress("0x0000000000000000000000000000000000000123")
	tx := ethtypes.NewTx(&ethtypes.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     7,
		GasTipCap: big.NewInt(500_000_000),
		GasFeeCap: big.NewInt(2_500_000_000),
		Gas:       50_000,
		To:        &to,
		Value:     big.NewInt(42),
	})
	signed, err := SignEthereumTx(kr, "alice", chainID, tx)
	require.NoError(t, err)

	msg, err := WrapAsMsgEthereumTx(signed)
	require.NoError(t, err)
	require.NotNil(t, msg)

	sender := msg.GetSender()
	require.Equal(t, strings.ToLower(from), strings.ToLower(sender.Hex()))

	asTx := msg.AsTransaction()
	require.Equal(t, uint64(7), asTx.Nonce())
	require.Equal(t, big.NewInt(42), asTx.Value())
}

func TestWrapAsMsgEthereumTx_RejectsUnsigned(t *testing.T) {
	tx := ethtypes.NewTx(&ethtypes.DynamicFeeTx{ChainID: big.NewInt(1414)})
	_, err := WrapAsMsgEthereumTx(tx)
	require.Error(t, err)
}
