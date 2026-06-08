package client

import (
	"testing"

	sdkmath "cosmossdk.io/math"
)

func TestWithAccountSettings(t *testing.T) {
	cfg := Config{}

	WithAccountHRP("cosmos")(&cfg)
	WithFeeDenom("uatom")(&cfg)
	gp := sdkmath.LegacyNewDecWithPrec(5, 2) // 0.05
	WithGasPrice(gp)(&cfg)

	if cfg.AccountHRP != "cosmos" {
		t.Fatalf("AccountHRP = %q, want cosmos", cfg.AccountHRP)
	}
	if cfg.FeeDenom != "uatom" {
		t.Fatalf("FeeDenom = %q, want uatom", cfg.FeeDenom)
	}
	if cfg.GasPrice.IsNil() || !cfg.GasPrice.Equal(gp) {
		t.Fatalf("GasPrice = %s, want %s", cfg.GasPrice, gp)
	}
}

// An unset GasPrice must be a nil LegacyDec so that blockchain.New treats it as
// "not configured" and applies the chain default rather than a literal zero.
func TestGasPriceZeroValueIsNil(t *testing.T) {
	cfg := Config{}
	if !cfg.GasPrice.IsNil() {
		t.Fatal("unset GasPrice should be nil so the chain default is applied")
	}
}
