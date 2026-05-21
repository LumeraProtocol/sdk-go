package blockchain

import (
	"context"
	"net"
	"sync"
	"testing"

	evmigrationtypes "github.com/LumeraProtocol/lumera/x/evmigration/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type fakeEVMigrationServer struct {
	evmigrationtypes.UnimplementedQueryServer

	mu sync.Mutex

	params              *evmigrationtypes.QueryParamsResponse
	record              *evmigrationtypes.QueryMigrationRecordResponse
	recordByNewAddress  *evmigrationtypes.QueryMigrationRecordByNewAddressResponse
	estimate            *evmigrationtypes.QueryMigrationEstimateResponse
	stats               *evmigrationtypes.QueryMigrationStatsResponse
	notFound            bool
	lastLegacyAddress   string
	lastNewAddress      string
	lastEstimateLegacy  string
}

func (s *fakeEVMigrationServer) Params(_ context.Context, _ *evmigrationtypes.QueryParamsRequest) (*evmigrationtypes.QueryParamsResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.params, nil
}

func (s *fakeEVMigrationServer) MigrationRecord(_ context.Context, req *evmigrationtypes.QueryMigrationRecordRequest) (*evmigrationtypes.QueryMigrationRecordResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastLegacyAddress = req.GetLegacyAddress()
	if s.notFound {
		return nil, status.Error(codes.NotFound, "no record")
	}
	return s.record, nil
}

func (s *fakeEVMigrationServer) MigrationRecordByNewAddress(_ context.Context, req *evmigrationtypes.QueryMigrationRecordByNewAddressRequest) (*evmigrationtypes.QueryMigrationRecordByNewAddressResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastNewAddress = req.GetNewAddress()
	return s.recordByNewAddress, nil
}

func (s *fakeEVMigrationServer) MigrationEstimate(_ context.Context, req *evmigrationtypes.QueryMigrationEstimateRequest) (*evmigrationtypes.QueryMigrationEstimateResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastEstimateLegacy = req.GetLegacyAddress()
	return s.estimate, nil
}

func (s *fakeEVMigrationServer) MigrationStats(_ context.Context, _ *evmigrationtypes.QueryMigrationStatsRequest) (*evmigrationtypes.QueryMigrationStatsResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stats, nil
}

func (s *fakeEVMigrationServer) reads() (legacy, newAddr, estimateLegacy string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastLegacyAddress, s.lastNewAddress, s.lastEstimateLegacy
}

func newEVMigrationTestClient(t *testing.T, handler *fakeEVMigrationServer) *EVMigrationClient {
	t.Helper()

	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	t.Cleanup(func() {
		srv.Stop()
		_ = lis.Close()
	})

	evmigrationtypes.RegisterQueryServer(srv, handler)
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

	return &EVMigrationClient{
		query: evmigrationtypes.NewQueryClient(conn),
	}
}

func TestEVMigrationClient_Params(t *testing.T) {
	handler := &fakeEVMigrationServer{
		params: &evmigrationtypes.QueryParamsResponse{
			Params: evmigrationtypes.Params{
				EnableMigration:         true,
				MigrationEndTime:        1234567890,
				MaxMigrationsPerBlock:   42,
				MaxValidatorDelegations: 1000,
			},
		},
	}
	c := newEVMigrationTestClient(t, handler)

	resp, err := c.Params(context.Background())
	if err != nil {
		t.Fatalf("Params: %v", err)
	}
	if resp == nil || !resp.Params.EnableMigration {
		t.Fatalf("unexpected params response: %+v", resp)
	}
	if resp.Params.MaxMigrationsPerBlock != 42 {
		t.Fatalf("unexpected MaxMigrationsPerBlock: got %d", resp.Params.MaxMigrationsPerBlock)
	}
}

func TestEVMigrationClient_MigrationRecord(t *testing.T) {
	const legacy = "lumera1legacyaddrxxxxxxxxxxxxxxxxxxxxxxxxxx"
	handler := &fakeEVMigrationServer{
		record: &evmigrationtypes.QueryMigrationRecordResponse{
			Record: &evmigrationtypes.MigrationRecord{
				LegacyAddress:   legacy,
				NewAddress:      "lumera1newaddrxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
				MigrationTime:   1700000000,
				MigrationHeight: 42,
			},
		},
	}
	c := newEVMigrationTestClient(t, handler)

	resp, err := c.MigrationRecord(context.Background(), legacy)
	if err != nil {
		t.Fatalf("MigrationRecord: %v", err)
	}
	if resp == nil || resp.Record == nil {
		t.Fatalf("nil record")
	}
	if resp.Record.LegacyAddress != legacy {
		t.Fatalf("unexpected legacy address: got %q want %q", resp.Record.LegacyAddress, legacy)
	}
	if resp.Record.MigrationHeight != 42 {
		t.Fatalf("unexpected height: got %d", resp.Record.MigrationHeight)
	}

	gotLegacy, _, _ := handler.reads()
	if gotLegacy != legacy {
		t.Fatalf("server saw wrong legacy address: got %q want %q", gotLegacy, legacy)
	}
}

func TestEVMigrationClient_MigrationRecord_NotFound(t *testing.T) {
	handler := &fakeEVMigrationServer{notFound: true}
	c := newEVMigrationTestClient(t, handler)

	resp, err := c.MigrationRecord(context.Background(), "lumera1missing")
	if err == nil {
		t.Fatalf("expected error, got resp=%+v", resp)
	}
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestEVMigrationClient_MigrationRecordByNewAddress(t *testing.T) {
	const newAddr = "lumera1newaddrxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	handler := &fakeEVMigrationServer{
		recordByNewAddress: &evmigrationtypes.QueryMigrationRecordByNewAddressResponse{
			Record: &evmigrationtypes.MigrationRecord{
				LegacyAddress: "lumera1legacyaddrxxxxxxxxxxxxxxxxxxxxxxxxxx",
				NewAddress:    newAddr,
			},
		},
	}
	c := newEVMigrationTestClient(t, handler)

	resp, err := c.MigrationRecordByNewAddress(context.Background(), newAddr)
	if err != nil {
		t.Fatalf("MigrationRecordByNewAddress: %v", err)
	}
	if resp == nil || resp.Record == nil || resp.Record.NewAddress != newAddr {
		t.Fatalf("unexpected response: %+v", resp)
	}

	_, gotNew, _ := handler.reads()
	if gotNew != newAddr {
		t.Fatalf("server saw wrong new address: got %q want %q", gotNew, newAddr)
	}
}

func TestEVMigrationClient_MigrationEstimate(t *testing.T) {
	const legacy = "lumera1estimatemexxxxxxxxxxxxxxxxxxxxxxxxxx"
	handler := &fakeEVMigrationServer{
		estimate: &evmigrationtypes.QueryMigrationEstimateResponse{
			IsValidator:     true,
			DelegationCount: 5,
			UnbondingCount:  1,
			TotalTouched:    6,
			WouldSucceed:    true,
		},
	}
	c := newEVMigrationTestClient(t, handler)

	resp, err := c.MigrationEstimate(context.Background(), legacy)
	if err != nil {
		t.Fatalf("MigrationEstimate: %v", err)
	}
	if !resp.WouldSucceed || !resp.IsValidator || resp.TotalTouched != 6 {
		t.Fatalf("unexpected estimate: %+v", resp)
	}

	_, _, gotLegacy := handler.reads()
	if gotLegacy != legacy {
		t.Fatalf("server saw wrong legacy address: got %q want %q", gotLegacy, legacy)
	}
}

func TestEVMigrationClient_MigrationStats(t *testing.T) {
	handler := &fakeEVMigrationServer{
		stats: &evmigrationtypes.QueryMigrationStatsResponse{
			TotalMigrated:           10,
			TotalLegacy:             100,
			TotalLegacyStaked:       50,
			TotalValidatorsMigrated: 2,
			TotalValidatorsLegacy:   3,
		},
	}
	c := newEVMigrationTestClient(t, handler)

	resp, err := c.MigrationStats(context.Background())
	if err != nil {
		t.Fatalf("MigrationStats: %v", err)
	}
	if resp.TotalMigrated != 10 || resp.TotalLegacy != 100 {
		t.Fatalf("unexpected stats: %+v", resp)
	}
}

func TestNewMsgClaimLegacyAccount(t *testing.T) {
	legacyProof := evmigrationtypes.MigrationProof{}
	newProof := evmigrationtypes.MigrationProof{}
	msg := NewMsgClaimLegacyAccount("new", "legacy", legacyProof, newProof)
	if msg.NewAddress != "new" || msg.LegacyAddress != "legacy" {
		t.Fatalf("unexpected msg: %+v", msg)
	}
}

func TestNewMsgMigrateValidator(t *testing.T) {
	legacyProof := evmigrationtypes.MigrationProof{}
	newProof := evmigrationtypes.MigrationProof{}
	msg := NewMsgMigrateValidator("new", "legacy", legacyProof, newProof)
	if msg.NewAddress != "new" || msg.LegacyAddress != "legacy" {
		t.Fatalf("unexpected msg: %+v", msg)
	}
}
