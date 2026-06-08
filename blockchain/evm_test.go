package blockchain

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net"
	"strings"
	"testing"

	sdkmath "cosmossdk.io/math"
	blockbase "github.com/LumeraProtocol/sdk-go/blockchain/base"
	"github.com/LumeraProtocol/sdk-go/constants"
	sdkgrpc "github.com/LumeraProtocol/sdk-go/internal/grpc"
	sdkcrypto "github.com/LumeraProtocol/sdk-go/pkg/crypto"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	precisebanktypes "github.com/cosmos/evm/x/precisebank/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// Mock the QueryClient interfaces directly. Going through bufconn breaks on
// cosmos custom types (sdkmath.Int / LegacyDec) because the default gRPC
// codec uses proto reflection v2, which does not understand gogoproto
// `customtype=` annotations. The production base client gets the right
// codec via cosmos-sdk's gRPC wiring; here we test the wrapper logic only.

type stubEVMQuery struct {
	codeResp         *evmtypes.QueryCodeResponse
	storageResp      *evmtypes.QueryStorageResponse
	balanceResp      *evmtypes.QueryBalanceResponse
	accountResp      *evmtypes.QueryAccountResponse
	cosmosAccount    *evmtypes.QueryCosmosAccountResponse
	paramsResp       *evmtypes.QueryParamsResponse
	baseFeeResp      *evmtypes.QueryBaseFeeResponse
	configResp       *evmtypes.QueryConfigResponse
	globalMinResp    *evmtypes.QueryGlobalMinGasPriceResponse
	ethCallResp      *evmtypes.MsgEthereumTxResponse
	estimateGasResp  *evmtypes.EstimateGasResponse
	traceResp        *evmtypes.QueryTraceTxResponse
	lastEthCallArgs  []byte
	lastEstimateArgs []byte
}

func (s *stubEVMQuery) Code(_ context.Context, _ *evmtypes.QueryCodeRequest, _ ...grpc.CallOption) (*evmtypes.QueryCodeResponse, error) {
	return s.codeResp, nil
}
func (s *stubEVMQuery) Storage(_ context.Context, _ *evmtypes.QueryStorageRequest, _ ...grpc.CallOption) (*evmtypes.QueryStorageResponse, error) {
	return s.storageResp, nil
}
func (s *stubEVMQuery) Balance(_ context.Context, _ *evmtypes.QueryBalanceRequest, _ ...grpc.CallOption) (*evmtypes.QueryBalanceResponse, error) {
	return s.balanceResp, nil
}
func (s *stubEVMQuery) Account(_ context.Context, _ *evmtypes.QueryAccountRequest, _ ...grpc.CallOption) (*evmtypes.QueryAccountResponse, error) {
	return s.accountResp, nil
}
func (s *stubEVMQuery) CosmosAccount(_ context.Context, _ *evmtypes.QueryCosmosAccountRequest, _ ...grpc.CallOption) (*evmtypes.QueryCosmosAccountResponse, error) {
	return s.cosmosAccount, nil
}
func (s *stubEVMQuery) ValidatorAccount(_ context.Context, _ *evmtypes.QueryValidatorAccountRequest, _ ...grpc.CallOption) (*evmtypes.QueryValidatorAccountResponse, error) {
	return nil, nil
}
func (s *stubEVMQuery) Params(_ context.Context, _ *evmtypes.QueryParamsRequest, _ ...grpc.CallOption) (*evmtypes.QueryParamsResponse, error) {
	return s.paramsResp, nil
}
func (s *stubEVMQuery) BaseFee(_ context.Context, _ *evmtypes.QueryBaseFeeRequest, _ ...grpc.CallOption) (*evmtypes.QueryBaseFeeResponse, error) {
	return s.baseFeeResp, nil
}
func (s *stubEVMQuery) Config(_ context.Context, _ *evmtypes.QueryConfigRequest, _ ...grpc.CallOption) (*evmtypes.QueryConfigResponse, error) {
	return s.configResp, nil
}
func (s *stubEVMQuery) GlobalMinGasPrice(_ context.Context, _ *evmtypes.QueryGlobalMinGasPriceRequest, _ ...grpc.CallOption) (*evmtypes.QueryGlobalMinGasPriceResponse, error) {
	return s.globalMinResp, nil
}
func (s *stubEVMQuery) EthCall(_ context.Context, req *evmtypes.EthCallRequest, _ ...grpc.CallOption) (*evmtypes.MsgEthereumTxResponse, error) {
	s.lastEthCallArgs = append([]byte(nil), req.Args...)
	return s.ethCallResp, nil
}
func (s *stubEVMQuery) EstimateGas(_ context.Context, req *evmtypes.EthCallRequest, _ ...grpc.CallOption) (*evmtypes.EstimateGasResponse, error) {
	s.lastEstimateArgs = append([]byte(nil), req.Args...)
	return s.estimateGasResp, nil
}
func (s *stubEVMQuery) TraceTx(_ context.Context, _ *evmtypes.QueryTraceTxRequest, _ ...grpc.CallOption) (*evmtypes.QueryTraceTxResponse, error) {
	return s.traceResp, nil
}
func (s *stubEVMQuery) TraceBlock(_ context.Context, _ *evmtypes.QueryTraceBlockRequest, _ ...grpc.CallOption) (*evmtypes.QueryTraceBlockResponse, error) {
	return nil, nil
}
func (s *stubEVMQuery) TraceCall(_ context.Context, _ *evmtypes.QueryTraceCallRequest, _ ...grpc.CallOption) (*evmtypes.QueryTraceCallResponse, error) {
	return nil, nil
}

func TestEVMClient_Queries(t *testing.T) {
	baseFee := sdkmath.NewInt(2_500_000_000)
	stub := &stubEVMQuery{
		codeResp:    &evmtypes.QueryCodeResponse{Code: []byte{0x60, 0x60}},
		storageResp: &evmtypes.QueryStorageResponse{Value: "0x000000000000000000000000000000000000000000000000000000000000002a"},
		balanceResp: &evmtypes.QueryBalanceResponse{Balance: "1000000000000000000"},
		accountResp: &evmtypes.QueryAccountResponse{Balance: "1000000000000000000", CodeHash: "0x0", Nonce: 7},
		cosmosAccount: &evmtypes.QueryCosmosAccountResponse{
			CosmosAddress: "lumera1abc",
			Sequence:      7,
			AccountNumber: 42,
		},
		baseFeeResp:     &evmtypes.QueryBaseFeeResponse{BaseFee: &baseFee},
		globalMinResp:   &evmtypes.QueryGlobalMinGasPriceResponse{MinGasPrice: sdkmath.NewInt(500_000_000)},
		ethCallResp:     &evmtypes.MsgEthereumTxResponse{Ret: []byte{0x01}},
		estimateGasResp: &evmtypes.EstimateGasResponse{Gas: 21_000},
	}
	c := &EVMClient{query: stub}
	ctx := context.Background()
	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")

	code, err := c.Code(ctx, addr)
	if err != nil || len(code) != 2 {
		t.Fatalf("Code: %v len=%d", err, len(code))
	}

	hash, err := c.Storage(ctx, addr, common.Hash{})
	if err != nil || hash.Big().Uint64() != 42 {
		t.Fatalf("Storage: %v hash=%s", err, hash.Hex())
	}

	bal, err := c.Balance(ctx, addr)
	if err != nil || bal.String() != "1000000000000000000" {
		t.Fatalf("Balance: %v %s", err, bal)
	}

	acct, err := c.EthAccount(ctx, addr)
	if err != nil || acct.Nonce != 7 {
		t.Fatalf("EthAccount: %v %+v", err, acct)
	}

	cacct, err := c.CosmosAccount(ctx, addr)
	if err != nil || cacct.AccountNumber != 42 {
		t.Fatalf("CosmosAccount: %v %+v", err, cacct)
	}

	bf, err := c.BaseFee(ctx)
	if err != nil || bf.String() != "2500000000" {
		t.Fatalf("BaseFee: %v %s", err, bf)
	}

	mg, err := c.GlobalMinGasPrice(ctx)
	if err != nil || mg.String() != "500000000" {
		t.Fatalf("GlobalMinGasPrice: %v %s", err, mg)
	}

	to := common.HexToAddress("0xdeadbeef00000000000000000000000000000000")
	callResp, err := c.EthCall(ctx, addr, &to, []byte{0xaa, 0xbb}, 0)
	if err != nil || len(callResp.Ret) != 1 {
		t.Fatalf("EthCall: %v %+v", err, callResp)
	}

	var decoded evmtypes.TransactionArgs
	if err := json.Unmarshal(stub.lastEthCallArgs, &decoded); err != nil {
		t.Fatalf("decode ethcall args: %v", err)
	}
	if decoded.From == nil || *decoded.From != addr {
		t.Fatalf("from mismatch: %+v", decoded.From)
	}
	if decoded.To == nil || *decoded.To != to {
		t.Fatalf("to mismatch: %+v", decoded.To)
	}
	if decoded.Input == nil || !equalBytes(*decoded.Input, hexutil.Bytes{0xaa, 0xbb}) {
		t.Fatalf("input mismatch: %x", *decoded.Input)
	}

	gas, err := c.EstimateGas(ctx, addr, &to, []byte{0x01}, big.NewInt(0), 0)
	if err != nil || gas != 21_000 {
		t.Fatalf("EstimateGas: %v %d", err, gas)
	}
}

func TestEVMClient_BaseFee_Nil(t *testing.T) {
	stub := &stubEVMQuery{baseFeeResp: &evmtypes.QueryBaseFeeResponse{}}
	c := &EVMClient{query: stub}
	bf, err := c.BaseFee(context.Background())
	if err != nil {
		t.Fatalf("BaseFee: %v", err)
	}
	if bf.Sign() != 0 {
		t.Fatalf("expected zero, got %s", bf)
	}

	nilBaseFee := sdkmath.Int{}
	stub.baseFeeResp = &evmtypes.QueryBaseFeeResponse{BaseFee: &nilBaseFee}
	bf, err = c.BaseFee(context.Background())
	if err != nil {
		t.Fatalf("BaseFee with nil sdk.Int: %v", err)
	}
	if bf.Sign() != 0 {
		t.Fatalf("expected zero for nil sdk.Int, got %s", bf)
	}
}

func TestEVMClient_GlobalMinGasPrice_NilInt(t *testing.T) {
	stub := &stubEVMQuery{globalMinResp: &evmtypes.QueryGlobalMinGasPriceResponse{}}
	c := &EVMClient{query: stub}
	got, err := c.GlobalMinGasPrice(context.Background())
	if err != nil {
		t.Fatalf("GlobalMinGasPrice: %v", err)
	}
	if got.Sign() != 0 {
		t.Fatalf("expected zero, got %s", got)
	}
}

func TestEVMNumericQueries_RoundTripThroughGRPC(t *testing.T) {
	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer(grpc.ForceServerCodec(sdkgrpc.GogoCodec()))
	t.Cleanup(func() {
		srv.Stop()
		_ = lis.Close()
	})

	evmtypes.RegisterQueryServer(srv, &evmNumericQueryServer{})
	feemarkettypes.RegisterQueryServer(srv, &feeMarketNumericQueryServer{})
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

	evm := &EVMClient{query: evmtypes.NewQueryClient(sdkgrpc.GogoClientConn(conn))}
	baseFee, err := evm.BaseFee(context.Background())
	if err != nil {
		t.Fatalf("EVM BaseFee over gRPC: %v", err)
	}
	if baseFee.String() != "2500000000" {
		t.Fatalf("EVM BaseFee = %s", baseFee)
	}

	feeMarket := &FeeMarketClient{query: feemarkettypes.NewQueryClient(sdkgrpc.GogoClientConn(conn))}
	feeMarketBaseFee, err := feeMarket.BaseFee(context.Background())
	if err != nil {
		t.Fatalf("FeeMarket BaseFee over gRPC: %v", err)
	}
	if feeMarketBaseFee.String() != "0.002500000000000000" {
		t.Fatalf("FeeMarket BaseFee = %s", feeMarketBaseFee)
	}
}

func TestEVMClient_ResolveGasCaps_UsesFeeMarketMinGasPrice(t *testing.T) {
	ctx := context.Background()
	baseClient, err := blockbase.New(ctx, blockbase.Config{
		GRPCAddr:       "localhost:9090",
		EVMChainID:     big.NewInt(1414),
		EVMNativeDenom: "ulume",
	}, nil, "")
	if err != nil {
		t.Fatalf("base.New: %v", err)
	}
	t.Cleanup(func() { _ = baseClient.Close() })

	minGasPrice := sdkmath.LegacyMustNewDecFromStr("0.0005")
	client := &Client{
		Client: baseClient,
		FeeMarket: &FeeMarketClient{query: &stubFeeMarketQuery{
			params: &feemarkettypes.QueryParamsResponse{
				Params: feemarkettypes.Params{MinGasPrice: minGasPrice},
			},
		}},
	}
	baseFee := sdkmath.NewInt(2_500_000_000)
	evm := &EVMClient{
		client: client,
		query: &stubEVMQuery{
			baseFeeResp: &evmtypes.QueryBaseFeeResponse{BaseFee: &baseFee},
		},
	}

	tip, fee, err := evm.resolveGasCaps(ctx, nil, nil)
	if err != nil {
		t.Fatalf("resolveGasCaps: %v", err)
	}
	if tip.String() != "500000000" {
		t.Fatalf("tip cap = %s, want 500000000", tip)
	}
	if fee.String() != "5500000000" {
		t.Fatalf("fee cap = %s, want 5500000000", fee)
	}
}

func TestEVMClient_ResolveEVMDenoms_QueryParamsWhenConfigMissing(t *testing.T) {
	evm := &EVMClient{query: &stubEVMQuery{
		paramsResp: &evmtypes.QueryParamsResponse{Params: evmtypes.Params{
			EvmDenom: "ulume",
			ExtendedDenomOptions: &evmtypes.ExtendedDenomOptions{
				ExtendedDenom: "alume",
			},
		}},
	}}

	nativeDenom, extendedDenom, err := evm.resolveEVMDenoms(context.Background(), Config{})
	if err != nil {
		t.Fatalf("resolveEVMDenoms: %v", err)
	}
	if nativeDenom != "ulume" || extendedDenom != "alume" {
		t.Fatalf("denoms = %q/%q, want ulume/alume", nativeDenom, extendedDenom)
	}
}

func TestEVMClient_ResolveEVMDenoms_ConfigShortCircuitsQuery(t *testing.T) {
	// query is nil on purpose: when both denoms are configured, resolveEVMDenoms
	// must not touch the query client.
	evm := &EVMClient{}
	nativeDenom, extendedDenom, err := evm.resolveEVMDenoms(context.Background(), Config{
		EVMNativeDenom:   "ulume",
		EVMExtendedDenom: "alume",
	})
	if err != nil {
		t.Fatalf("resolveEVMDenoms: %v", err)
	}
	if nativeDenom != "ulume" || extendedDenom != "alume" {
		t.Fatalf("denoms = %q/%q, want ulume/alume", nativeDenom, extendedDenom)
	}
}

func TestEVMClient_ResolveEVMDenoms_NoQueryNoConfig(t *testing.T) {
	evm := &EVMClient{}
	_, _, err := evm.resolveEVMDenoms(context.Background(), Config{})
	if err == nil {
		t.Fatalf("expected error when query client and config denoms are both absent")
	}
	if !strings.Contains(err.Error(), "EVMNativeDenom") {
		t.Fatalf("error %q should mention the config field to set", err)
	}
}

func TestEVMClient_ResolveEVMDenoms_MissingExtendedDenom(t *testing.T) {
	// Chain returns the native denom but no ExtendedDenomOptions, and config
	// supplies neither: resolveEVMDenoms must surface a clear extended-denom error.
	evm := &EVMClient{query: &stubEVMQuery{
		paramsResp: &evmtypes.QueryParamsResponse{Params: evmtypes.Params{
			EvmDenom: "ulume",
		}},
	}}
	_, _, err := evm.resolveEVMDenoms(context.Background(), Config{})
	if err == nil {
		t.Fatalf("expected error when extended denom cannot be resolved")
	}
	if !strings.Contains(err.Error(), "extended denom") {
		t.Fatalf("error %q should mention extended denom", err)
	}
}

func TestEncodeEthCallArgs_RejectsNegativeValue(t *testing.T) {
	_, err := encodeEthCallArgs(common.Address{}, nil, nil, big.NewInt(-1))
	if err == nil {
		t.Fatalf("expected negative value error")
	}
}

func TestDecodeMsgEthereumTxResponses_AcceptsHexPrefix(t *testing.T) {
	want := &evmtypes.MsgEthereumTxResponse{Ret: []byte{0x01, 0x02}}
	txData := &sdk.TxMsgData{
		MsgResponses: []*codectypes.Any{codectypes.UnsafePackAny(want)},
	}
	raw, err := proto.Marshal(txData)
	if err != nil {
		t.Fatalf("marshal tx data: %v", err)
	}

	got, err := decodeMsgEthereumTxResponses([]byte("0x" + hex.EncodeToString(raw)))
	if err != nil {
		t.Fatalf("decode tx responses: %v", err)
	}
	if len(got) != 1 || !equalBytes(got[0].Ret, want.Ret) {
		t.Fatalf("decoded response mismatch: %+v", got)
	}
}

func TestDecodeMsgEthereumTxResponses_AcceptsRawProtoBytes(t *testing.T) {
	want := &evmtypes.MsgEthereumTxResponse{Ret: []byte{0x03, 0x04}}
	txData := &sdk.TxMsgData{
		MsgResponses: []*codectypes.Any{codectypes.UnsafePackAny(want)},
	}
	raw, err := proto.Marshal(txData)
	if err != nil {
		t.Fatalf("marshal tx data: %v", err)
	}

	got, err := decodeMsgEthereumTxResponses(raw)
	if err != nil {
		t.Fatalf("decode raw tx responses: %v", err)
	}
	if len(got) != 1 || !equalBytes(got[0].Ret, want.Ret) {
		t.Fatalf("decoded response mismatch: %+v", got)
	}
}

func TestCallContract_ReturnsSenderDerivationError(t *testing.T) {
	kr, _ := newCosmosSigningKeyringForEVM(t, "alice")
	baseClient, err := blockbase.New(context.Background(), blockbase.Config{
		GRPCAddr:       "127.0.0.1:1",
		EVMChainID:     big.NewInt(1414),
		EVMNativeDenom: "ulume",
	}, kr, "alice")
	if err != nil {
		t.Fatalf("base.New: %v", err)
	}
	t.Cleanup(func() { _ = baseClient.Close() })

	evm := &EVMClient{
		client: &Client{Client: baseClient},
		query:  &stubEVMQuery{ethCallResp: &evmtypes.MsgEthereumTxResponse{}},
	}

	_, err = evm.CallContract(context.Background(), common.HexToAddress("0x000000000000000000000000000000000000dEaD"), nil)
	if err == nil {
		t.Fatalf("expected sender derivation error")
	}
}

type stubFeeMarketQuery struct {
	params  *feemarkettypes.QueryParamsResponse
	baseFee *feemarkettypes.QueryBaseFeeResponse
	gas     *feemarkettypes.QueryBlockGasResponse
}

type evmNumericQueryServer struct {
	evmtypes.UnimplementedQueryServer
}

func (evmNumericQueryServer) BaseFee(context.Context, *evmtypes.QueryBaseFeeRequest) (*evmtypes.QueryBaseFeeResponse, error) {
	baseFee := sdkmath.NewInt(2_500_000_000)
	return &evmtypes.QueryBaseFeeResponse{BaseFee: &baseFee}, nil
}

type feeMarketNumericQueryServer struct {
	feemarkettypes.UnimplementedQueryServer
}

func (feeMarketNumericQueryServer) BaseFee(context.Context, *feemarkettypes.QueryBaseFeeRequest) (*feemarkettypes.QueryBaseFeeResponse, error) {
	baseFee := sdkmath.LegacyMustNewDecFromStr("0.0025")
	return &feemarkettypes.QueryBaseFeeResponse{BaseFee: &baseFee}, nil
}

func (s *stubFeeMarketQuery) Params(_ context.Context, _ *feemarkettypes.QueryParamsRequest, _ ...grpc.CallOption) (*feemarkettypes.QueryParamsResponse, error) {
	return s.params, nil
}
func (s *stubFeeMarketQuery) BaseFee(_ context.Context, _ *feemarkettypes.QueryBaseFeeRequest, _ ...grpc.CallOption) (*feemarkettypes.QueryBaseFeeResponse, error) {
	return s.baseFee, nil
}
func (s *stubFeeMarketQuery) BlockGas(_ context.Context, _ *feemarkettypes.QueryBlockGasRequest, _ ...grpc.CallOption) (*feemarkettypes.QueryBlockGasResponse, error) {
	return s.gas, nil
}

func TestFeeMarketClient_Queries(t *testing.T) {
	bf := sdkmath.LegacyMustNewDecFromStr("0.0025")
	stub := &stubFeeMarketQuery{
		params: &feemarkettypes.QueryParamsResponse{Params: feemarkettypes.Params{
			BaseFeeChangeDenominator: 16,
		}},
		baseFee: &feemarkettypes.QueryBaseFeeResponse{BaseFee: &bf},
		gas:     &feemarkettypes.QueryBlockGasResponse{Gas: 250_000},
	}
	c := &FeeMarketClient{query: stub}
	ctx := context.Background()

	p, err := c.Params(ctx)
	if err != nil || p.Params.BaseFeeChangeDenominator != 16 {
		t.Fatalf("Params: %v %+v", err, p)
	}

	got, err := c.BaseFee(ctx)
	if err != nil || got.String() != "0.002500000000000000" {
		t.Fatalf("BaseFee: %v %s", err, got)
	}

	g, err := c.BlockGas(ctx)
	if err != nil || g != 250_000 {
		t.Fatalf("BlockGas: %v %d", err, g)
	}
}

func TestFeeMarketClient_BlockGas_RejectsNegative(t *testing.T) {
	c := &FeeMarketClient{query: &stubFeeMarketQuery{
		gas: &feemarkettypes.QueryBlockGasResponse{Gas: -1},
	}}
	if _, err := c.BlockGas(context.Background()); err == nil {
		t.Fatal("expected error for negative block gas")
	}
}

type stubPreciseBankQuery struct {
	remainder *precisebanktypes.QueryRemainderResponse
	frac      *precisebanktypes.QueryFractionalBalanceResponse
	lastAddr  string
}

func (s *stubPreciseBankQuery) Remainder(_ context.Context, _ *precisebanktypes.QueryRemainderRequest, _ ...grpc.CallOption) (*precisebanktypes.QueryRemainderResponse, error) {
	return s.remainder, nil
}
func (s *stubPreciseBankQuery) FractionalBalance(_ context.Context, req *precisebanktypes.QueryFractionalBalanceRequest, _ ...grpc.CallOption) (*precisebanktypes.QueryFractionalBalanceResponse, error) {
	s.lastAddr = req.Address
	return s.frac, nil
}

func TestPreciseBankClient_Queries(t *testing.T) {
	stub := &stubPreciseBankQuery{
		remainder: &precisebanktypes.QueryRemainderResponse{Remainder: sdk.NewInt64Coin("ulume", 0)},
		frac:      &precisebanktypes.QueryFractionalBalanceResponse{FractionalBalance: sdk.NewInt64Coin("ulume", 123)},
	}
	c := &PreciseBankClient{query: stub}
	ctx := context.Background()

	r, err := c.Remainder(ctx)
	if err != nil || r.Denom != "ulume" {
		t.Fatalf("Remainder: %v %+v", err, r)
	}

	f, err := c.FractionalBalance(ctx, "lumera1abc")
	if err != nil || f.Amount.Int64() != 123 {
		t.Fatalf("FractionalBalance: %v %+v", err, f)
	}
	if stub.lastAddr != "lumera1abc" {
		t.Fatalf("server saw wrong addr: %q", stub.lastAddr)
	}
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func newCosmosSigningKeyringForEVM(t *testing.T, keyName string) (keyring.Keyring, string) {
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
	if _, err := kr.NewAccount(keyName, evmTxTestMnemonic, "", sdk.FullFundraiserPath, hd.Secp256k1); err != nil {
		t.Fatalf("new account: %v", err)
	}
	addr, err := sdkcrypto.AddressFromKey(kr, keyName, constants.LumeraAccountHRP)
	if err != nil {
		t.Fatalf("derive address: %v", err)
	}
	return kr, addr
}
