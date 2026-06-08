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
	evmtypes "github.com/cosmos/evm/x/vm/types"
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
	simErr           error
	accountNumber    uint64
	sequence         uint64
	simulateCalls    int
	accountInfoCalls int
}

type broadcastWaitServer struct {
	txtypes.UnimplementedServiceServer
	broadcast *abcipb.TxResponse
	get       *abcipb.TxResponse
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
	if s.simErr != nil {
		return nil, s.simErr
	}
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

func (s *broadcastWaitServer) BroadcastTx(context.Context, *txtypes.BroadcastTxRequest) (*txtypes.BroadcastTxResponse, error) {
	return &txtypes.BroadcastTxResponse{TxResponse: s.broadcast}, nil
}

func (s *broadcastWaitServer) GetTx(context.Context, *txtypes.GetTxRequest) (*txtypes.GetTxResponse, error) {
	return &txtypes.GetTxResponse{TxResponse: s.get}, nil
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

func TestBuildAndSignTxWithOptions_ZeroFee(t *testing.T) {
	kr, addr := newSigningTestKeyring(t, "alice")
	c := &Client{
		keyring: kr,
		keyName: "alice",
		config: Config{
			ChainID:    "lumera-devnet-1",
			AccountHRP: constants.LumeraAccountHRP,
		},
	}

	accountNumber := uint64(7)
	sequence := uint64(9)
	txBytes, err := c.BuildAndSignTxWithOptions(context.Background(), TxBuildOptions{
		Messages:       []sdk.Msg{newMsgSend(addr, addr)},
		GasLimit:       250000,
		SkipSimulation: true,
		AccountNumber:  &accountNumber,
		Sequence:       &sequence,
		ZeroFee:        true,
	})
	if err != nil {
		t.Fatalf("BuildAndSignTxWithOptions zero fee error: %v", err)
	}

	assertDecodedTx(t, txBytes, 250000, 0, sequence, 1)
}

func TestBroadcastAndWait_ChecksIncludedTxCode(t *testing.T) {
	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	t.Cleanup(func() {
		srv.Stop()
		_ = lis.Close()
	})

	handler := &broadcastWaitServer{
		broadcast: &abcipb.TxResponse{Txhash: "hash"},
		get:       &abcipb.TxResponse{Txhash: "hash", Code: 7, RawLog: "deliver failed"},
	}
	txtypes.RegisterServiceServer(srv, handler)
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

	c := &Client{
		conn: conn,
		config: Config{
			WaitTx: clientconfig.WaitTxConfig{
				PollInterval:          time.Millisecond,
				PollMaxRetries:        1,
				PollBackoffMultiplier: 1,
			},
		},
	}

	_, _, err = c.BroadcastAndWait(context.Background(), []byte{0x01}, txtypes.BroadcastMode_BROADCAST_MODE_SYNC)
	if err == nil {
		t.Fatalf("expected included tx code error")
	}
	if !strings.Contains(err.Error(), "tx failed with code 7") {
		t.Fatalf("unexpected error: %v", err)
	}
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

func TestBuildAndSignTxWithOptions_SimulationErrorPropagates(t *testing.T) {
	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	t.Cleanup(func() {
		srv.Stop()
		_ = lis.Close()
	})

	handler := &txBuildServer{
		simErr:        status.Error(codes.InvalidArgument, "insufficient funds"),
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

	// A failed simulation must surface as an error, not silently fall back to
	// the fixed default gas limit (which could under-gas the tx on-chain).
	_, err = c.BuildAndSignTxWithOptions(context.Background(), TxBuildOptions{
		Messages: []sdk.Msg{newMsgSend(addr, addr)},
	})
	if err == nil {
		t.Fatal("expected simulation error to propagate, got nil")
	}
	if !strings.Contains(err.Error(), "insufficient funds") {
		t.Fatalf("error %q should include the simulation failure reason", err)
	}

	// Setting an explicit gas limit must bypass simulation and succeed.
	handler2 := &txBuildServer{accountNumber: 3, sequence: 4}
	lis2 := bufconn.Listen(bufSize)
	srv2 := grpc.NewServer()
	t.Cleanup(func() {
		srv2.Stop()
		_ = lis2.Close()
	})
	txtypes.RegisterServiceServer(srv2, handler2)
	authtypes.RegisterQueryServer(srv2, handler2)
	go func() { _ = srv2.Serve(lis2) }()
	conn2, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis2.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufnet: %v", err)
	}
	t.Cleanup(func() { _ = conn2.Close() })
	c.conn = conn2

	if _, err := c.BuildAndSignTxWithOptions(context.Background(), TxBuildOptions{
		Messages: []sdk.Msg{newMsgSend(addr, addr)},
		GasLimit: 50_000,
	}); err != nil {
		t.Fatalf("explicit GasLimit should bypass simulation: %v", err)
	}
	if sim, _ := handler2.counts(); sim != 0 {
		t.Fatalf("explicit GasLimit should not call Simulate, got %d calls", sim)
	}
}

func TestValidateTxBuildOptions_RejectsMsgEthereumTx(t *testing.T) {
	kr, _ := newSigningTestKeyring(t, "alice")
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
	_, err := c.BuildAndSignTxWithOptions(context.Background(), TxBuildOptions{
		Messages: []sdk.Msg{&evmtypes.MsgEthereumTx{}},
	})
	if err == nil {
		t.Fatalf("expected error rejecting MsgEthereumTx")
	}
	if !strings.Contains(err.Error(), "EVMClient.SendEthereumTransaction") {
		t.Fatalf("expected guard error, got %v", err)
	}
}

func TestResolveFeeAmount_LargeGasDoesNotOverflow(t *testing.T) {
	c := &Client{config: Config{
		FeeDenom: "ulume",
		GasPrice: sdkmath.LegacyOneDec(),
	}}

	gas := ^uint64(0)
	fees, err := c.resolveFeeAmount(gas, TxBuildOptions{})
	if err != nil {
		t.Fatalf("resolveFeeAmount: %v", err)
	}
	if len(fees) != 1 || fees[0].Denom != "ulume" {
		t.Fatalf("unexpected fees: %s", fees)
	}
	if !fees[0].Amount.Equal(sdkmath.NewIntFromUint64(gas)) {
		t.Fatalf("fee amount = %s, want %s", fees[0].Amount, sdkmath.NewIntFromUint64(gas))
	}
}

func TestNewAppliesDefaultMessageSizes(t *testing.T) {
	c, err := New(context.Background(), Config{GRPCAddr: "localhost:9090"}, nil, "")
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	if c.config.MaxRecvMsgSize != defaultMaxRecvMessageSize || c.config.MaxSendMsgSize != defaultMaxSendMessageSize {
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
	if wantFee == 0 {
		if !fee.Empty() {
			t.Fatalf("unexpected fee: %s", fee)
		}
	} else if len(fee) != 1 || fee[0].Denom != "ulume" || !fee[0].Amount.Equal(sdkmath.NewInt(wantFee)) {
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
