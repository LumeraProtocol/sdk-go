package blockchain

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	abcipb "cosmossdk.io/api/cosmos/base/abci/v1beta1"
	txtypes "cosmossdk.io/api/cosmos/tx/v1beta1"
	clientconfig "github.com/LumeraProtocol/sdk-go/client/config"
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
