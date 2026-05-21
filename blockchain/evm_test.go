package blockchain

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	precisebanktypes "github.com/cosmos/evm/x/precisebank/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"google.golang.org/grpc"
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
}

type stubFeeMarketQuery struct {
	params  *feemarkettypes.QueryParamsResponse
	baseFee *feemarkettypes.QueryBaseFeeResponse
	gas     *feemarkettypes.QueryBlockGasResponse
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
