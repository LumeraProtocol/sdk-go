package client

import clientconfig "github.com/LumeraProtocol/sdk-go/client/config"

// Config re-exports the config.Config type for backwards compatibility.
type Config = clientconfig.Config

// WaitTxConfig re-exports the wait-tx config type for backwards compatibility.
type WaitTxConfig = clientconfig.WaitTxConfig

// DefaultConfig mirrors config.Default.
func DefaultConfig() Config {
	return clientconfig.Default()
}

// DefaultWaitTxConfig mirrors config.DefaultWaitTxConfig.
func DefaultWaitTxConfig() WaitTxConfig {
	return clientconfig.DefaultWaitTxConfig()
}

// ApplyWaitTxDefaults mirrors config.ApplyWaitTxDefaults.
func ApplyWaitTxDefaults(cfg *WaitTxConfig) {
	clientconfig.ApplyWaitTxDefaults(cfg)
}
