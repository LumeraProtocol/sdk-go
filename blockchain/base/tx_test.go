package base

import (
	"context"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	abcipb "cosmossdk.io/api/cosmos/base/abci/v1beta1"
	txtypes "cosmossdk.io/api/cosmos/tx/v1beta1"
	sdkmath "cosmossdk.io/math"
	clientconfig "github.com/LumeraProtocol/sdk-go/client/config"
	"github.com/LumeraProtocol/sdk-go/constants"
	sdkcrypto "github.com/LumeraProtocol/sdk-go/pkg/crypto"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type getTxSequenceServer struct {
	txtypes.UnimplementedServiceServer
	mu        sync.Mutex
	responses []getTxStep
	calls     int
}

type getTxStep struct {
	resp *txtypes.GetTxResponse
	err  error
}

type txBuildServer struct {
	txtypes.UnimplementedServiceServer
	authtypes.UnimplementedQueryServer
	mu               sync.Mutex
	gasUsed          uint64
	accountNumber    uint64
	sequence         uint64
	simulateCalls    int
	accountInfoCalls int
}

func (s *getTxSequenceServer) GetTx(ctx context.Context, req *txtypes.GetTxRequest) (*txtypes.GetTxResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.calls
	if idx >= len(s.responses) {
		idx = len(s.responses) - 1
	}
	s.calls++
	step := s.responses[idx]
	return step.resp, step.err
}

func (s *getTxSequenceServer) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func (s *txBuildServer) Simulate(context.Context, *txtypes.SimulateRequest) (*txtypes.SimulateResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.simulateCalls++
	return &txtypes.SimulateResponse{
		GasInfo: &abcipb.GasInfo{GasUsed: s.gasUsed},
	}, nil
}

func (s *txBuildServer) AccountInfo(context.Context, *authtypes.QueryAccountInfoRequest) (*authtypes.QueryAccountInfoResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accountInfoCalls++
	return &authtypes.QueryAccountInfoResponse{
		Info: &authtypes.BaseAccount{
			AccountNumber: s.accountNumber,
			Sequence:      s.sequence,
		},
	}, nil
}

func (s *txBuildServer) counts() (simulateCalls int, accountInfoCalls int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.simulateCalls, s.accountInfoCalls
}

func TestWaitForTxInclusionRetriesNotFoundAfterWaitSuccess(t *testing.T) {
	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	t.Cleanup(func() {
		srv.Stop()
		_ = lis.Close()
	})

	txHash := "hash"
	successResp := &txtypes.GetTxResponse{
		TxResponse: &abcipb.TxResponse{Txhash: txHash},
	}

	handler := &getTxSequenceServer{
		responses: []getTxStep{
			{resp: successResp}, // websocket/poller observes inclusion
			{err: status.Error(codes.NotFound, "not indexed yet")}, // first post-wait fetch hits slow index
			{err: status.Error(codes.NotFound, "still indexing")},  // retry still not ready
			{resp: successResp}, // eventual success once indexed
		},
	}
	txtypes.RegisterServiceServer(srv, handler)
	go func() {
		_ = srv.Serve(lis)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufnet: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	c := &Client{
		conn: conn,
		config: Config{
			RPCEndpoint: "",
			WaitTx: clientconfig.WaitTxConfig{
				PollInterval:          time.Millisecond,
				PollMaxRetries:        5,
				PollBackoffMultiplier: 1,
			},
		},
	}

	resp, err := c.WaitForTxInclusion(ctx, txHash)
	if err != nil {
		t.Fatalf("WaitForTxInclusion error: %v", err)
	}
	if resp == nil || resp.TxResponse == nil || resp.TxResponse.Txhash != txHash {
		t.Fatalf("unexpected tx response: %+v", resp)
	}

	if got, want := handler.callCount(), len(handler.responses); got != want {
		t.Fatalf("unexpected GetTx call count: got %d, want %d", got, want)
	}
}

func TestBuildAndSignTxWithOptions_ManualSignerInfo(t *testing.T) {
	kr, addr := newSigningTestKeyring(t, "alice")
	c := &Client{
		keyring: kr,
		keyName: "alice",
		config: Config{
			ChainID:    "lumera-devnet-1",
			AccountHRP: constants.LumeraAccountHRP,
			FeeDenom:   "ulume",
			GasPrice:   sdkmath.LegacyMustNewDecFromStr("0.025"),
		},
	}

	accountNumber := uint64(7)
	sequence := uint64(9)
	txBytes, err := c.BuildAndSignTxWithOptions(context.Background(), TxBuildOptions{
		Messages:       []sdk.Msg{newMsgSend(addr, addr)},
		Memo:           "manual",
		GasLimit:       250000,
		SkipSimulation: true,
		AccountNumber:  &accountNumber,
		Sequence:       &sequence,
	})
	if err != nil {
		t.Fatalf("BuildAndSignTxWithOptions error: %v", err)
	}

	assertDecodedTx(t, txBytes, 250000, 6250, sequence, 1)
}

func TestBuildAndSignTxWithOptions_QueriesAccountInfoAndSimulates(t *testing.T) {
	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	t.Cleanup(func() {
		srv.Stop()
		_ = lis.Close()
	})

	handler := &txBuildServer{
		gasUsed:       2000,
		accountNumber: 3,
		sequence:      4,
	}
	txtypes.RegisterServiceServer(srv, handler)
	authtypes.RegisterQueryServer(srv, handler)
	go func() {
		_ = srv.Serve(lis)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufnet: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	kr, addr := newSigningTestKeyring(t, "alice")
	c := &Client{
		conn:    conn,
		keyring: kr,
		keyName: "alice",
		config: Config{
			ChainID:    "lumera-devnet-1",
			AccountHRP: constants.LumeraAccountHRP,
			FeeDenom:   "ulume",
			GasPrice:   sdkmath.LegacyMustNewDecFromStr("0.025"),
		},
	}

	txBytes, err := c.BuildAndSignTxWithOptions(context.Background(), TxBuildOptions{
		Messages:      []sdk.Msg{newMsgSend(addr, addr)},
		Memo:          "queried",
		GasAdjustment: 1.5,
	})
	if err != nil {
		t.Fatalf("BuildAndSignTxWithOptions error: %v", err)
	}

	assertDecodedTx(t, txBytes, 3000, 75, 4, 1)
	simCalls, acctCalls := handler.counts()
	if simCalls != 1 || acctCalls != 1 {
		t.Fatalf("unexpected query/simulate call counts: simulate=%d account_info=%d", simCalls, acctCalls)
	}
}

func TestNewAppliesDefaultMessageSizes(t *testing.T) {
	c, err := New(context.Background(), Config{GRPCAddr: "localhost:9090"}, nil, "")
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	if c.config.MaxRecvMsgSize != defaultMaxMessageSize || c.config.MaxSendMsgSize != defaultMaxMessageSize {
		t.Fatalf("unexpected max message sizes: recv=%d send=%d", c.config.MaxRecvMsgSize, c.config.MaxSendMsgSize)
	}
}

func newSigningTestKeyring(t *testing.T, keyName string) (keyring.Keyring, string) {
	t.Helper()

	kr, err := sdkcrypto.NewKeyring(sdkcrypto.KeyringParams{
		AppName: "lumera",
		Backend: "test",
		Dir:     t.TempDir(),
		Input:   strings.NewReader(""),
	})
	if err != nil {
		t.Fatalf("create keyring: %v", err)
	}

	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	if _, err := kr.NewAccount(keyName, mnemonic, "", sdk.FullFundraiserPath, hd.Secp256k1); err != nil {
		t.Fatalf("new account: %v", err)
	}

	addr, err := sdkcrypto.AddressFromKey(kr, keyName, constants.LumeraAccountHRP)
	if err != nil {
		t.Fatalf("derive address: %v", err)
	}

	return kr, addr
}

func newMsgSend(fromAddr, toAddr string) *banktypes.MsgSend {
	return &banktypes.MsgSend{
		FromAddress: fromAddr,
		ToAddress:   toAddr,
		Amount:      sdk.NewCoins(sdk.NewInt64Coin("ulume", 1)),
	}
}

func assertDecodedTx(t *testing.T, txBytes []byte, wantGas uint64, wantFee int64, wantSequence uint64, wantMsgs int) {
	t.Helper()

	txCfg := sdkcrypto.NewDefaultTxConfig()
	decoded, err := txCfg.TxDecoder()(txBytes)
	if err != nil {
		t.Fatalf("decode tx: %v", err)
	}

	feeTx, ok := decoded.(sdk.FeeTx)
	if !ok {
		t.Fatalf("decoded tx does not implement sdk.FeeTx")
	}
	if feeTx.GetGas() != wantGas {
		t.Fatalf("unexpected gas: got %d want %d", feeTx.GetGas(), wantGas)
	}
	fee := feeTx.GetFee()
	if len(fee) != 1 || fee[0].Denom != "ulume" || !fee[0].Amount.Equal(sdkmath.NewInt(wantFee)) {
		t.Fatalf("unexpected fee: %s", fee)
	}

	sigTx, ok := decoded.(authsigning.SigVerifiableTx)
	if !ok {
		t.Fatalf("decoded tx does not implement authsigning.SigVerifiableTx")
	}
	sigs, err := sigTx.GetSignaturesV2()
	if err != nil {
		t.Fatalf("get signatures: %v", err)
	}
	if len(sigs) != 1 || sigs[0].Sequence != wantSequence {
		t.Fatalf("unexpected signatures: %+v", sigs)
	}

	if got := len(decoded.GetMsgs()); got != wantMsgs {
		t.Fatalf("unexpected msg count: got %d want %d", got, wantMsgs)
	}
}
