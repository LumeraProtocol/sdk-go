//go:build ignore
// +build ignore

// Package client provides a thin Lumera gRPC client
// that is independent from supernode-specific adapters, while reusing
// the shared tx helper (simulate → build&sign → broadcast) to keep
// transaction code small and correct.
package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	humanize "github.com/dustin/go-humanize"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	nodev1 "cosmossdk.io/api/cosmos/base/node/v1beta1"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	actionv1 "github.com/LumeraProtocol/lumera/x/action/v1/types"

	authmod "github.com/LumeraProtocol/supernode/v2/pkg/lumera/modules/auth"
	txhelper "github.com/LumeraProtocol/supernode/v2/pkg/lumera/modules/tx"

	"github.com/LumeraProtocol/network-maker/config"
	log "github.com/LumeraProtocol/network-maker/pkg/log"
	grpcclient "github.com/LumeraProtocol/network-maker/pkg/net/grpc-client"
)

type clientImpl struct {
	logger log.Logger
	cfg    *config.Config
	opts   *grpcclient.ClientOptions
	gc     *grpcclient.Client
	cc     *grpc.ClientConn

	// typed stubs
	acctQ  authtypes.QueryClient
	actQ   actionv1.QueryClient
	nodeS  nodev1.ServiceClient
	actMsg actionv1.MsgClient
	txS    txtypes.ServiceClient

	// tx helper (simulate → sign → broadcast). We reuse the shared helper package
	// from the supernode repo so we don't have to reimplement the tx pipeline.
	txh        *txhelper.TxHelper
	nodeConfig *nodev1.ConfigResponse
}

// applyGrpcClientOptions builds grpc dial options from network-maker config options
func applyGrpcClientOptions(cfg *config.Config) (*grpcclient.ClientOptions, error) {
	opts := grpcclient.DefaultClientOptions()

	g := cfg.CfgOpts.Lumera.GRPC
	if g.MaxRecvMsgSize != "" {
		size, err := humanize.ParseBytes(g.MaxRecvMsgSize)
		if err != nil {
			return nil, fmt.Errorf("failed to parse MaxRecvMsgSize: %w", err)
		}
		opts.MaxRecvMsgSize = int(size)
	}

	if g.MaxSendMsgSize != "" {
		size, err := humanize.ParseBytes(g.MaxSendMsgSize)
		if err != nil {
			return nil, fmt.Errorf("failed to parse MaxSendMsgSize: %w", err)
		}
		opts.MaxSendMsgSize = int(size)
	}

	if g.InitialWindowSize != "" {
		size, err := humanize.ParseBytes(g.InitialWindowSize)
		if err != nil {
			return nil, fmt.Errorf("failed to parse InitialWindowSize: %w", err)
		}
		opts.InitialWindowSize = int32(size)
	}

	if g.InitialConnWindowSize != "" {
		size, err := humanize.ParseBytes(g.InitialConnWindowSize)
		if err != nil {
			return nil, fmt.Errorf("failed to parse InitialConnWindowSize: %w", err)
		}
		opts.InitialConnWindowSize = int32(size)
	}

	if g.ConnWaitTime != nil {
		opts.ConnWaitTime = *g.ConnWaitTime
	}

	if g.MaxRetries != nil {
		opts.MaxRetries = *g.MaxRetries
	}

	if g.RetryWaitTime != nil {
		opts.RetryWaitTime = *g.RetryWaitTime
	}

	if g.EnableRetries != nil {
		opts.EnableRetries = *g.EnableRetries
	}

	if g.UserAgent != "" {
		opts.UserAgent = g.UserAgent
	}

	if g.MinConnectTimeout != nil {
		opts.MinConnectTimeout = *g.MinConnectTimeout
	}

	if g.KeepAlive.Time != nil {
		opts.KeepAliveTime = *g.KeepAlive.Time
	}

	if g.KeepAlive.Timeout != nil {
		opts.KeepAliveTimeout = *g.KeepAlive.Timeout
	}

	if g.KeepAlive.AllowWithoutStream != nil {
		opts.AllowWithoutStream = *g.KeepAlive.AllowWithoutStream
	}

	if g.Backoff.BaseDelay != nil {
		opts.BackoffConfig.BaseDelay = *g.Backoff.BaseDelay
	}

	if g.Backoff.Multiplier != nil {
		opts.BackoffConfig.Multiplier = *g.Backoff.Multiplier
	}

	if g.Backoff.Jitter != nil {
		opts.BackoffConfig.Jitter = *g.Backoff.Jitter
	}

	if g.Backoff.MaxDelay != nil {
		opts.BackoffConfig.MaxDelay = *g.Backoff.MaxDelay
	}

	if g.Backoff.Jitter != nil {
		opts.BackoffConfig.Jitter = *g.Backoff.Jitter
	}

	if g.Backoff.MaxDelay != nil {
		opts.BackoffConfig.MaxDelay = *g.Backoff.MaxDelay
	}

	return opts, nil
}

// New creates a new Lumera client, opens the gRPC connection (secure if you
// pass "identity@host:port", insecure if just "host:port"), wires typed stubs,
// and initializes the tx helper from the shared package.
func New(cfg *config.Config, logger log.Logger) (LumeraClient, error) {
	opts, err := applyGrpcClientOptions(cfg)
	if err != nil {
		return nil, err
	}

	return &clientImpl{
		cfg:    cfg,
		logger: logger,
		opts:   opts,
	}, nil
}

func (c *clientImpl) Connect(ctx context.Context) error {
	if c.cc != nil {
		return nil // already connected
	}

	if c.cfg == nil || c.cfg.CfgOpts == nil {
		return fmt.Errorf("lumera: config is nil")
	}

	// Normalize target and choose creds
	hostPort, authority, scheme := normalizeGRPCTarget(c.cfg.CfgOpts.Lumera.GRPCAddr)
	if c.opts.Authority == "" && authority != "" {
		c.opts.Authority = authority
	}

	// Decide creds: TLS for https/443, otherwise follow config
	var tlsCreds credentials.TransportCredentials
	if scheme == "https" || strings.HasSuffix(hostPort, ":443") {
		tlsCfg := &tls.Config{
			ServerName: authority,
			MinVersion: tls.VersionTLS12,
			NextProtos: []string{"h2"}, // gRPC requires HTTP/2 over TLS
		}
		tlsCreds = credentials.NewTLS(tlsCfg)
	}

	// Try primary dial first (respecting the scheme/port)
	var cc *grpc.ClientConn
	var err error
	if tlsCreds != nil {
		c.opts.EnforceProvidedCreds = true
		c.gc = grpcclient.NewClient(tlsCreds)
		cc, err = c.gc.Connect(ctx, hostPort, c.opts)
		// If the remote talks HTTP/1.1 (not native gRPC), retry on :9090 insecure.
		if err != nil && isHTTP11PrefaceErr(err) {
			c.logger.Warnf("TLS gRPC preface failed against %s; retrying insecure on :9090", hostPort)
			fallback := ensurePort(authority, "9090")
			c.gc = grpcclient.NewClient(insecure.NewCredentials())
			cc, err = c.gc.Connect(ctx, fallback, c.opts)
		}
	} else {
		c.gc = grpcclient.NewClient(insecure.NewCredentials())
		cc, err = c.gc.Connect(ctx, hostPort, c.opts)
	}
	if err != nil {
		return fmt.Errorf("lumera: connect: %w", err)
	}
	c.cc = cc

	// Typed clients for modules/services we use.
	c.acctQ = authtypes.NewQueryClient(cc)
	c.actQ = actionv1.NewQueryClient(cc)
	c.actMsg = actionv1.NewMsgClient(cc)
	c.nodeS = nodev1.NewServiceClient(cc)
	c.txS = txtypes.NewServiceClient(cc)

	// Wire the tx helper:
	// - tx module is created from the same gRPC conn
	// - auth module is used by the helper to fetch account info
	// - defaults pull chainID/keyName/keyring from cfg
	txMod, err := txhelper.NewModule(cc)
	if err != nil {
		_ = cc.Close()
		return fmt.Errorf("tx module init: %w", err)
	}
	authMod, err := authmod.NewModule(cc)
	if err != nil {
		_ = cc.Close()
		return fmt.Errorf("auth module init: %w", err)
	}

	// get Lumera node configuration
	c.nodeConfig, err = c.GetNodeConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get lumera node config: %w", err)
	}
	nodeConfigJSON, err := json.Marshal(c.nodeConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal lumera node config: %w", err)
	}
	c.logger.Infof("Lumera node config: %s", string(nodeConfigJSON))

	minGasPrice := strings.TrimSuffix(c.nodeConfig.MinimumGasPrice, c.cfg.CfgOpts.Lumera.Denom)
	gasPrice, err := strconv.ParseFloat(minGasPrice, 64)
	if err != nil {
		return fmt.Errorf("failed to parse minimum gas price: %w", err)
	}
	gasPrice *= 1.1

	txHelperConfig := txhelper.TxHelperConfig{
		ChainID:       c.cfg.CfgOpts.Lumera.ChainID,
		KeyName:       c.cfg.SdkConfig.Account.KeyName,
		Keyring:       c.cfg.SdkConfig.Account.Keyring,
		FeeDenom:      c.cfg.CfgOpts.Lumera.Denom,
		GasPadding:    txhelper.DefaultGasPadding + 1000, // add 1000 to the default padding
		GasAdjustment: txhelper.DefaultGasAdjustment,
		GasPrice:      strconv.FormatFloat(gasPrice, 'f', -1, 64),
	}
	c.txh = txhelper.NewTxHelper(
		authMod,
		txMod,
		&txHelperConfig,
	)
	return nil
}

// normalizeGRPCTarget turns "https://host[/path][:port]" into ("host:port", "host", "https"|"http"|"")
func normalizeGRPCTarget(raw string) (string, string, string) {
	raw = strings.TrimSpace(raw)

	// Fast path: URL
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		if u, err := url.Parse(raw); err == nil && u.Host != "" {
			hostPort := u.Host
			if _, _, err := net.SplitHostPort(hostPort); err != nil {
				// no port; add a sensible default
				switch u.Scheme {
				case "https":
					hostPort = net.JoinHostPort(u.Hostname(), "443")
				case "http":
					hostPort = net.JoinHostPort(u.Hostname(), "80")
				}
			}
			return hostPort, u.Hostname(), u.Scheme
		}
		// Fallback if parse fails
		raw = strings.TrimPrefix(strings.TrimPrefix(raw, "https://"), "http://")
	}
	// Drop any path/query
	if i := strings.IndexByte(raw, '/'); i >= 0 {
		raw = raw[:i]
	}
	authority := raw
	if h, _, err := net.SplitHostPort(raw); err == nil {
		authority = h
	}
	return raw, authority, ""
}

// ensurePort returns "host:port" given a host (or host:port) and a default port.
func ensurePort(host string, def string) string {
	if _, _, err := net.SplitHostPort(host); err == nil {
		return host
	}
	return net.JoinHostPort(host, def)
}

// isHTTP11PrefaceErr detects the classic "HTTP/1.1 header" / "frame too large" gRPC preface mismatch.
func isHTTP11PrefaceErr(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "HTTP/1.1 header") ||
		strings.Contains(s, "server preface") ||
		strings.Contains(s, "frame too large")
}
