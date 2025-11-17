//go:build ignore
// +build ignore

package waittx

import (
	"testing"
	"time"
)

func TestExponentialBackoffSequence(t *testing.T) {
	b := ExponentialBackoff{
		Initial:    time.Second,
		Multiplier: 2,
		Max:        5 * time.Second,
	}
	want := []time.Duration{
		time.Second,
		2 * time.Second,
		4 * time.Second,
		5 * time.Second,
		5 * time.Second,
	}
	for i, expected := range want {
		got := b.Next(i + 1)
		if got != expected {
			t.Fatalf("attempt %d: want %v; got %v", i+1, expected, got)
		}
	}
}

func TestExponentialBackoffJitter(t *testing.T) {
	base := time.Second
	b := ExponentialBackoff{
		Initial: base,
		Jitter:  0.5,
	}
	b.Rand = func() float64 { return 0 }
	if got := b.Next(1); got != base/2 {
		t.Fatalf("jitter low bound: want %v; got %v", base/2, got)
	}
	b.Rand = func() float64 { return 1 }
	if got := b.Next(1); got != base+base/2 {
		t.Fatalf("jitter high bound: want %v; got %v", base+base/2, got)
	}
}

func TestExponentialBackoffDefaults(t *testing.T) {
	b := ExponentialBackoff{}
	if got := b.Next(0); got != 500*time.Millisecond {
		t.Fatalf("default initial: want %v; got %v", 500*time.Millisecond, got)
	}

	b = ExponentialBackoff{Multiplier: 0.5}
	if got := b.Next(3); got != 500*time.Millisecond {
		t.Fatalf("multiplier <= 1 should not shrink delay: want %v; got %v", 500*time.Millisecond, got)
	}
}
