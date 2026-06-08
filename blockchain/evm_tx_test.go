package blockchain

import (
	"context"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	abcipb "cosmossdk.io/api/cosmos/base/abci/v1beta1"
	txtypes "cosmossdk.io/api/cosmos/tx/v1beta1"
	blockbase "github.com/LumeraProtocol/sdk-go/blockchain/base"
	clientconfig "github.com/LumeraProtocol/sdk-go/client/config"
	sdkcrypto "github.com/LumeraProtocol/sdk-go/pkg/crypto"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
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
	withExt, ok := decoded.(interface {
		GetExtensionOptions() []*codectypes.Any
	})
	require.True(t, ok)
	require.Len(t, withExt.GetExtensionOptions(), 1)
	require.Equal(t, codectypes.MsgTypeURL(&evmtypes.ExtensionOptionsEthereumTx{}), withExt.GetExtensionOptions()[0].TypeUrl)
}

func TestBuildEthereumTxBytes_RequiresExtendedDenom(t *testing.T) {
	mnemonicFile := writeMnemonicForEVM(t)
	kr, _, _, err := sdkcrypto.LoadKeyring("alice", mnemonicFile, sdkcrypto.KeyTypeEVM)
	require.NoError(t, err)

	chainID := big.NewInt(1414)
	tx := ethtypes.NewTx(&ethtypes.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     3,
		GasTipCap: big.NewInt(500_000_000),
		GasFeeCap: big.NewInt(2_500_000_000),
		Gas:       50_000,
	})
	signed, err := sdkcrypto.SignEthereumTx(kr, "alice", chainID, tx)
	require.NoError(t, err)

	_, err = buildEthereumTxBytes(signed, "ulume", "", "ulume")
	require.Error(t, err)
	require.Contains(t, err.Error(), "extended denom")
}

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

func TestRawEthereumTx_RejectsChainIDMismatch(t *testing.T) {
	baseClient, err := blockbase.New(t.Context(), blockbase.Config{
		GRPCAddr:         "127.0.0.1:1",
		EVMChainID:       big.NewInt(1414),
		EVMNativeDenom:   "ulume",
		EVMExtendedDenom: "alume",
		FeeDenom:         "ulume",
		InsecureGRPC:     true,
	}, nil, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = baseClient.Close() })

	signed := signedEVMTestTx(t, big.NewInt(1415))
	_, err = (&EVMClient{client: &Client{Client: baseClient}}).RawEthereumTx(t.Context(), signed)
	require.Error(t, err)
	require.Contains(t, err.Error(), "chain ID")
}

func TestSendEthereumTransaction_HappyPath(t *testing.T) {
	srv, endpoint := newEVMGRPCTestServer(t)
	defer srv.Stop()

	mnemonicFile := writeMnemonicForEVM(t)
	kr, _, _, err := sdkcrypto.LoadKeyring("alice", mnemonicFile, sdkcrypto.KeyTypeEVM)
	require.NoError(t, err)

	baseClient, err := blockbase.New(t.Context(), blockbase.Config{
		ChainID:          "lumera-devnet-1",
		GRPCAddr:         endpoint,
		AccountHRP:       "lumera",
		FeeDenom:         "ulume",
		EVMChainID:       big.NewInt(1414),
		EVMNativeDenom:   "ulume",
		EVMExtendedDenom: "alume",
		InsecureGRPC:     true,
		WaitTx: clientconfig.WaitTxConfig{
			PollInterval:          time.Millisecond,
			PollMaxRetries:        3,
			PollBackoffMultiplier: 1,
		},
	}, kr, "alice")
	require.NoError(t, err)
	t.Cleanup(func() { _ = baseClient.Close() })

	to := common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	evm := &EVMClient{client: &Client{Client: baseClient}}
	res, err := evm.SendEthereumTransaction(t.Context(), &to, []byte{0xaa}, &EthereumTxOptions{
		Nonce:     uint64Ptr(7),
		GasLimit:  50_000,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(2),
		Value:     big.NewInt(3),
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotEmpty(t, res.EthTxHash)
	require.Equal(t, "COSMOS_HASH", res.CosmosHash)
	require.Equal(t, int64(12), res.Height)
}

type evmTxServiceServer struct {
	txtypes.UnimplementedServiceServer
}

func (s evmTxServiceServer) BroadcastTx(context.Context, *txtypes.BroadcastTxRequest) (*txtypes.BroadcastTxResponse, error) {
	return &txtypes.BroadcastTxResponse{TxResponse: &abcipb.TxResponse{Txhash: "COSMOS_HASH"}}, nil
}

func (s evmTxServiceServer) GetTx(context.Context, *txtypes.GetTxRequest) (*txtypes.GetTxResponse, error) {
	return &txtypes.GetTxResponse{TxResponse: &abcipb.TxResponse{Txhash: "COSMOS_HASH", Height: 12}}, nil
}

func newEVMGRPCTestServer(t *testing.T) (*grpc.Server, string) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	txtypes.RegisterServiceServer(srv, evmTxServiceServer{})
	go func() {
		_ = srv.Serve(lis)
	}()
	t.Cleanup(func() { _ = lis.Close() })
	return srv, lis.Addr().String()
}

func signedEVMTestTx(t *testing.T, chainID *big.Int) *ethtypes.Transaction {
	t.Helper()
	mnemonicFile := writeMnemonicForEVM(t)
	kr, _, _, err := sdkcrypto.LoadKeyring("alice", mnemonicFile, sdkcrypto.KeyTypeEVM)
	require.NoError(t, err)
	tx := ethtypes.NewTx(&ethtypes.DynamicFeeTx{
		ChainID:   chainID,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(2),
		Gas:       21_000,
	})
	signed, err := sdkcrypto.SignEthereumTx(kr, "alice", chainID, tx)
	require.NoError(t, err)
	return signed
}

func uint64Ptr(v uint64) *uint64 {
	return &v
}
