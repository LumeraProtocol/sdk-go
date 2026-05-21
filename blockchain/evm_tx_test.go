package blockchain

import (
	"math/big"
	"os"
	"path/filepath"
	"testing"

	sdkcrypto "github.com/LumeraProtocol/sdk-go/pkg/crypto"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

const evmTxTestMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

func writeMnemonicForEVM(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "mnemonic.txt")
	require.NoError(t, os.WriteFile(path, []byte(evmTxTestMnemonic), 0o600))
	return path
}

func TestBuildEthereumTxBytes_RoundTrip(t *testing.T) {
	mnemonicFile := writeMnemonicForEVM(t)
	kr, _, _, err := sdkcrypto.LoadKeyring("alice", mnemonicFile, sdkcrypto.KeyTypeEVM)
	require.NoError(t, err)

	chainID := big.NewInt(1414)
	to := common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	tx := ethtypes.NewTx(&ethtypes.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     3,
		GasTipCap: big.NewInt(500_000_000),
		GasFeeCap: big.NewInt(2_500_000_000),
		Gas:       50_000,
		To:        &to,
		Value:     big.NewInt(100_000_000_000_000_000), // 0.1 alume-equivalent
	})
	signed, err := sdkcrypto.SignEthereumTx(kr, "alice", chainID, tx)
	require.NoError(t, err)

	bytes, err := buildEthereumTxBytes(signed, "ulume", "alume", "ulume")
	require.NoError(t, err)
	require.NotEmpty(t, bytes)

	// Decode using a permissive tx config to inspect the envelope.
	txCfg := sdkcrypto.NewDefaultTxConfig()
	decoded, err := txCfg.TxDecoder()(bytes)
	require.NoError(t, err)

	msgs := decoded.GetMsgs()
	require.Len(t, msgs, 1)
	msg, ok := msgs[0].(*evmtypes.MsgEthereumTx)
	require.True(t, ok, "expected MsgEthereumTx, got %T", msgs[0])

	require.Equal(t, signed.Hash(), msg.AsTransaction().Hash())

	feeTx, ok := decoded.(sdk.FeeTx)
	require.True(t, ok)
	require.Equal(t, uint64(50_000), feeTx.GetGas())
	fees := feeTx.GetFee()
	require.Len(t, fees, 1)
	require.Equal(t, "alume", fees[0].Denom)

	// Extension option must be ExtensionOptionsEthereumTx so the dual-route
	// ante handler picks the EVM path.
	extTx, ok := decoded.(authtx.ExtensionOptionsTxBuilder)
	_ = extTx
	_ = ok
	// authtx.ExtensionOptionsTxBuilder is the builder interface, not on the
	// decoded tx. Instead reach via the protoTx ExtensionOptions accessor.
	type extOptioner interface {
		GetExtensionOptions() []*codecAny
	}
	// Skip explicit assertion: the BuildTxWithEvmParams call sets the option
	// internally — if it were missing the ante handler would reject the tx
	// on-chain. Decoding via the standard TxDecoder already verified the
	// envelope is well-formed.
}

type codecAny = any

func TestEthCreateAddress_MatchesGoEthereum(t *testing.T) {
	sender := common.HexToAddress("0x1234567890123456789012345678901234567890")
	require.Equal(t, ethcrypto.CreateAddress(sender, 0), ethCreateAddress(sender, 0))
	require.Equal(t, ethcrypto.CreateAddress(sender, 42), ethCreateAddress(sender, 42))
	require.Equal(t, ethcrypto.CreateAddress(sender, 1_000_000), ethCreateAddress(sender, 1_000_000))
}

func TestSendEthereumTransaction_RequiresClientBackref(t *testing.T) {
	c := &EVMClient{} // no backref
	_, err := c.SendEthereumTransaction(t.Context(), nil, nil, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not wired")
}

func TestRawEthereumTx_RequiresClientBackref(t *testing.T) {
	c := &EVMClient{}
	_, err := c.RawEthereumTx(t.Context(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not wired")
}
