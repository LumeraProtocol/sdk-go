package client

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
)

// Factory keeps a base configuration and keyring so callers can easily create
// per-signer clients without re-specifying shared settings.
type Factory struct {
	baseCfg Config
	keyring keyring.Keyring
	opts    []Option
}

// NewFactory captures the shared configuration and keyring. The base config may
// omit Address/KeyName; they are supplied when creating signer-specific clients.
func NewFactory(cfg Config, kr keyring.Keyring, opts ...Option) (*Factory, error) {
	if kr == nil {
		return nil, fmt.Errorf("keyring is required")
	}
	return &Factory{
		baseCfg: cfg,
		keyring: kr,
		opts:    append([]Option{}, opts...),
	}, nil
}

// WithSigner returns a Client bound to the provided address/keyName pair. Extra
// options override/extend the factory defaults for this instance.
func (f *Factory) WithSigner(ctx context.Context, address, keyName string, extraOpts ...Option) (*Client, error) {
	if address == "" {
		return nil, fmt.Errorf("address is required")
	}
	if keyName == "" {
		return nil, fmt.Errorf("key name is required")
	}

	cfg := f.baseCfg
	cfg.Address = address
	cfg.KeyName = keyName

	opts := append([]Option{}, f.opts...)
	opts = append(opts, extraOpts...)

	return New(ctx, cfg, f.keyring, opts...)
}
