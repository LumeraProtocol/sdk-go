package config

import (
	"testing"
	"time"
)

func TestDefaultMessageSizes(t *testing.T) {
	cfg := Default()

	if cfg.MaxRecvMsgSize != 4*1024*1024 {
		t.Fatalf("MaxRecvMsgSize = %d, want 4MiB", cfg.MaxRecvMsgSize)
	}
	if cfg.MaxSendMsgSize != 50*1024*1024 {
		t.Fatalf("MaxSendMsgSize = %d, want 50MiB", cfg.MaxSendMsgSize)
	}
}

func TestValidateAppliesDefaultMessageSizes(t *testing.T) {
	cfg := Config{
		ChainID:           "lumera-devnet-1",
		GRPCEndpoint:      "localhost:9090",
		RPCEndpoint:       "http://localhost:26657",
		Address:           "lumera1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqduux2w",
		KeyName:           "alice",
		BlockchainTimeout: time.Second,
		StorageTimeout:    time.Second,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if cfg.MaxRecvMsgSize != 4*1024*1024 {
		t.Fatalf("MaxRecvMsgSize = %d, want 4MiB", cfg.MaxRecvMsgSize)
	}
	if cfg.MaxSendMsgSize != 50*1024*1024 {
		t.Fatalf("MaxSendMsgSize = %d, want 50MiB", cfg.MaxSendMsgSize)
	}
}
