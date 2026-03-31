package base

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type accountInfoStep struct {
	resp *authtypes.QueryAccountInfoResponse
	err  error
}

type accountInfoSequenceServer struct {
	authtypes.UnimplementedQueryServer
	mu        sync.Mutex
	steps     []accountInfoStep
	callCount int
}

func (s *accountInfoSequenceServer) AccountInfo(ctx context.Context, req *authtypes.QueryAccountInfoRequest) (*authtypes.QueryAccountInfoResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.callCount
	if idx >= len(s.steps) {
		idx = len(s.steps) - 1
	}
	s.callCount++
	step := s.steps[idx]
	return step.resp, step.err
}

func (s *accountInfoSequenceServer) calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.callCount
}

func newBufConnClient(t *testing.T, server authtypes.QueryServer) *Client {
	t.Helper()

	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)
	grpcSrv := grpc.NewServer()
	authtypes.RegisterQueryServer(grpcSrv, server)
	go func() {
		_ = grpcSrv.Serve(lis)
	}()
	t.Cleanup(func() {
		grpcSrv.Stop()
		_ = lis.Close()
	})

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

	c := &Client{conn: conn}
	c.sleepHook = func(context.Context, int64) error { return nil }
	c.jitterHook = func(int64) int64 { return 0 }
	return c
}

func TestQueryAccountInfoWithRetry_TransientEOFThenSuccess(t *testing.T) {
	srv := &accountInfoSequenceServer{steps: []accountInfoStep{
		{err: status.Error(codes.Unavailable, "connection error: desc = \"transport: authentication handshake failed: EOF\"")},
		{resp: &authtypes.QueryAccountInfoResponse{Info: &authtypes.BaseAccount{AccountNumber: 7, Sequence: 11}}},
	}}
	c := newBufConnClient(t, srv)

	reconnectCalls := 0
	c.reconnectHook = func(context.Context) error {
		reconnectCalls++
		return nil
	}

	resp, err := c.queryAccountInfoWithRetry(context.Background(), "lumera1test")
	if err != nil {
		t.Fatalf("queryAccountInfoWithRetry error: %v", err)
	}
	if resp == nil || resp.Info == nil {
		t.Fatalf("expected account info response, got %+v", resp)
	}
	if got, want := srv.calls(), 2; got != want {
		t.Fatalf("unexpected server call count: got %d want %d", got, want)
	}
	if reconnectCalls != 1 {
		t.Fatalf("expected one reconnect attempt, got %d", reconnectCalls)
	}
}

func TestQueryAccountInfoWithRetry_NonTransientNoRetry(t *testing.T) {
	srv := &accountInfoSequenceServer{steps: []accountInfoStep{{
		err: status.Error(codes.NotFound, "account not found"),
	}}}
	c := newBufConnClient(t, srv)

	reconnectCalls := 0
	c.reconnectHook = func(context.Context) error {
		reconnectCalls++
		return nil
	}

	_, err := c.queryAccountInfoWithRetry(context.Background(), "lumera1missing")
	if err == nil {
		t.Fatalf("expected error")
	}
	if got, want := srv.calls(), 1; got != want {
		t.Fatalf("unexpected server call count: got %d want %d", got, want)
	}
	if reconnectCalls != 0 {
		t.Fatalf("did not expect reconnect, got %d", reconnectCalls)
	}
}

func TestQueryAccountInfoWithRetry_MaxAttempts(t *testing.T) {
	srv := &accountInfoSequenceServer{steps: []accountInfoStep{{
		err: status.Error(codes.Unavailable, "transport: authentication handshake failed: EOF"),
	}}}
	c := newBufConnClient(t, srv)
	c.reconnectHook = func(context.Context) error { return nil }

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := c.queryAccountInfoWithRetry(ctx, "lumera1retry")
	if err == nil {
		t.Fatalf("expected error after retries")
	}
	if got, want := srv.calls(), accountInfoMaxAttempts; got != want {
		t.Fatalf("unexpected server call count: got %d want %d", got, want)
	}
}
