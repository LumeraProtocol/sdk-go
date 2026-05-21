package blockchain

import (
	"context"
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	erc20types "github.com/cosmos/evm/x/erc20/types"
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc"
)

type stubERC20Query struct {
	pairs    *erc20types.QueryTokenPairsResponse
	pair     *erc20types.QueryTokenPairResponse
	params   *erc20types.QueryParamsResponse
	lastReq  string
	lastPage *query.PageRequest
}

func (s *stubERC20Query) TokenPairs(_ context.Context, req *erc20types.QueryTokenPairsRequest, _ ...grpc.CallOption) (*erc20types.QueryTokenPairsResponse, error) {
	s.lastPage = req.Pagination
	return s.pairs, nil
}
func (s *stubERC20Query) TokenPair(_ context.Context, req *erc20types.QueryTokenPairRequest, _ ...grpc.CallOption) (*erc20types.QueryTokenPairResponse, error) {
	s.lastReq = req.Token
	return s.pair, nil
}
func (s *stubERC20Query) Params(_ context.Context, _ *erc20types.QueryParamsRequest, _ ...grpc.CallOption) (*erc20types.QueryParamsResponse, error) {
	return s.params, nil
}

func TestERC20Client_Queries(t *testing.T) {
	stub := &stubERC20Query{
		pairs: &erc20types.QueryTokenPairsResponse{
			TokenPairs: []erc20types.TokenPair{{
				Erc20Address: "0x0000000000000000000000000000000000000001",
				Denom:        "ulume",
				Enabled:      true,
			}},
		},
		pair: &erc20types.QueryTokenPairResponse{
			TokenPair: erc20types.TokenPair{
				Erc20Address: "0x0000000000000000000000000000000000000001",
				Denom:        "ulume",
			},
		},
		params: &erc20types.QueryParamsResponse{
			Params: erc20types.Params{EnableErc20: true},
		},
	}
	c := &ERC20Client{query: stub}
	ctx := context.Background()

	pairs, _, err := c.TokenPairs(ctx, &query.PageRequest{Limit: 10})
	if err != nil || len(pairs) != 1 || pairs[0].Denom != "ulume" {
		t.Fatalf("TokenPairs: %v %+v", err, pairs)
	}
	if stub.lastPage == nil || stub.lastPage.Limit != 10 {
		t.Fatalf("pagination not forwarded: %+v", stub.lastPage)
	}

	pair, err := c.TokenPair(ctx, "ulume")
	if err != nil || pair.Denom != "ulume" {
		t.Fatalf("TokenPair: %v %+v", err, pair)
	}
	if stub.lastReq != "ulume" {
		t.Fatalf("server saw %q", stub.lastReq)
	}

	p, err := c.Params(ctx)
	if err != nil || !p.Params.EnableErc20 {
		t.Fatalf("Params: %v %+v", err, p)
	}
}

func TestNewMsgConvertCoin(t *testing.T) {
	coin := sdk.NewCoin("ulume", sdkmath.NewInt(1_000_000))
	receiver := common.HexToAddress("0x1234567890123456789012345678901234567890")
	msg := NewMsgConvertCoin(coin, receiver, "lumera1sender")

	if msg.Sender != "lumera1sender" {
		t.Fatalf("sender mismatch: %s", msg.Sender)
	}
	if msg.Receiver != receiver.Hex() {
		t.Fatalf("receiver mismatch: %s vs %s", msg.Receiver, receiver.Hex())
	}
	if msg.Coin.Denom != "ulume" || msg.Coin.Amount.Int64() != 1_000_000 {
		t.Fatalf("coin mismatch: %+v", msg.Coin)
	}
}

func TestNewMsgConvertERC20(t *testing.T) {
	contract := common.HexToAddress("0x000000000000000000000000000000000000aaaa")
	sender := common.HexToAddress("0x000000000000000000000000000000000000bbbb")
	msg := NewMsgConvertERC20(sdkmath.NewInt(42), "lumera1recv", contract, sender)

	if msg.ContractAddress != contract.Hex() {
		t.Fatalf("contract mismatch: %s", msg.ContractAddress)
	}
	if msg.Sender != sender.Hex() {
		t.Fatalf("sender mismatch: %s", msg.Sender)
	}
	if msg.Receiver != "lumera1recv" {
		t.Fatalf("receiver mismatch: %s", msg.Receiver)
	}
	if msg.Amount.Int64() != 42 {
		t.Fatalf("amount mismatch: %s", msg.Amount)
	}
}

func TestNewMsgRegisterERC20(t *testing.T) {
	addrs := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		common.HexToAddress("0x0000000000000000000000000000000000000002"),
	}
	msg := NewMsgRegisterERC20("lumera1auth", addrs)
	if msg.Signer != "lumera1auth" {
		t.Fatalf("signer mismatch: %s", msg.Signer)
	}
	if len(msg.Erc20Addresses) != 2 {
		t.Fatalf("addrs len: %d", len(msg.Erc20Addresses))
	}
	if msg.Erc20Addresses[0] != addrs[0].Hex() {
		t.Fatalf("addr mismatch: %s", msg.Erc20Addresses[0])
	}
}

func TestNewMsgToggleConversion(t *testing.T) {
	msg := NewMsgToggleConversion("lumera1auth", "ulume")
	if msg.Authority != "lumera1auth" || msg.Token != "ulume" {
		t.Fatalf("unexpected msg: %+v", msg)
	}
}

func TestErc20ABI_PackUnpackUint256(t *testing.T) {
	// Confirm the embedded ABI can pack balanceOf and unpack a uint256 result.
	holder := common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	calldata, err := erc20ABI.Pack("balanceOf", holder)
	if err != nil {
		t.Fatalf("pack: %v", err)
	}
	// First 4 bytes = selector; remaining must end with the holder address.
	if len(calldata) != 4+32 {
		t.Fatalf("calldata len %d, want 36", len(calldata))
	}

	// Synthesize a 32-byte uint256 = 12345 and unpack.
	ret := make([]byte, 32)
	big.NewInt(12345).FillBytes(ret)
	out, err := erc20ABI.Unpack("balanceOf", ret)
	if err != nil {
		t.Fatalf("unpack: %v", err)
	}
	if got := out[0].(*big.Int).Int64(); got != 12345 {
		t.Fatalf("unpack got %d", got)
	}
}
