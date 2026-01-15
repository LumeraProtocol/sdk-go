package ica

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	txtypes "cosmossdk.io/api/cosmos/tx/v1beta1"
	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
	"github.com/LumeraProtocol/sdk-go/blockchain/base"
	sdkcrypto "github.com/LumeraProtocol/sdk-go/pkg/crypto"
	sdktypes "github.com/LumeraProtocol/sdk-go/types"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	controllertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
)

const (
	defaultRelativeTimeout = 10 * time.Minute
	defaultPollDelay       = 2 * time.Second
	defaultPollRetries     = 120
	defaultAckRetries      = 120
)

// Config configures the ICA controller for a controller/host chain pair.
type Config struct {
	Controller base.Config
	Host       base.Config
	Keyring    keyring.Keyring
	KeyName    string

	ConnectionID             string
	CounterpartyConnectionID string
	Ordering                 channeltypes.Order
	RelativeTimeout          time.Duration
	PollDelay                time.Duration
	PollRetries              int
	AckRetries               int
}

// Controller manages ICA registration and message execution via gRPC.
type Controller struct {
	cfg          Config
	controllerBC *base.Client
	hostBC       *base.Client
	ownerAddr    string
	appPubkey    []byte
}

// NewController creates a new ICA controller using gRPC-based queries and txs.
func NewController(ctx context.Context, cfg Config) (*Controller, error) {
	if cfg.Keyring == nil {
		return nil, fmt.Errorf("keyring is required")
	}
	if strings.TrimSpace(cfg.KeyName) == "" {
		return nil, fmt.Errorf("key name is required")
	}
	if strings.TrimSpace(cfg.ConnectionID) == "" {
		return nil, fmt.Errorf("connection id is required")
	}
	if strings.TrimSpace(cfg.Controller.GRPCAddr) == "" {
		return nil, fmt.Errorf("controller gRPC address is required")
	}
	if strings.TrimSpace(cfg.Controller.ChainID) == "" {
		return nil, fmt.Errorf("controller chain id is required")
	}
	if strings.TrimSpace(cfg.Controller.AccountHRP) == "" {
		return nil, fmt.Errorf("controller account HRP is required")
	}
	if cfg.RelativeTimeout == 0 {
		cfg.RelativeTimeout = defaultRelativeTimeout
	}
	if cfg.PollDelay <= 0 {
		cfg.PollDelay = defaultPollDelay
	}
	if cfg.PollRetries <= 0 {
		cfg.PollRetries = defaultPollRetries
	}
	if cfg.AckRetries <= 0 {
		cfg.AckRetries = defaultAckRetries
	}
	if cfg.Ordering == 0 {
		cfg.Ordering = channeltypes.ORDERED
	}

	controllerBC, err := base.New(ctx, cfg.Controller, cfg.Keyring, cfg.KeyName)
	if err != nil {
		return nil, fmt.Errorf("create controller blockchain client: %w", err)
	}
	hostBC, err := base.New(ctx, cfg.Host, cfg.Keyring, cfg.KeyName)
	if err != nil {
		_ = controllerBC.Close()
		return nil, fmt.Errorf("create host blockchain client: %w", err)
	}

	rec, err := cfg.Keyring.Key(cfg.KeyName)
	if err != nil {
		_ = controllerBC.Close()
		_ = hostBC.Close()
		return nil, fmt.Errorf("load key %s: %w", cfg.KeyName, err)
	}
	pub, err := rec.GetPubKey()
	if err != nil {
		_ = controllerBC.Close()
		_ = hostBC.Close()
		return nil, fmt.Errorf("get pubkey: %w", err)
	}
	if pub == nil {
		_ = controllerBC.Close()
		_ = hostBC.Close()
		return nil, fmt.Errorf("pubkey is nil")
	}
	ownerAddr, err := sdkcrypto.AddressFromKey(cfg.Keyring, cfg.KeyName, cfg.Controller.AccountHRP)
	if err != nil {
		_ = controllerBC.Close()
		_ = hostBC.Close()
		return nil, fmt.Errorf("derive owner address: %w", err)
	}

	return &Controller{
		cfg:          cfg,
		controllerBC: controllerBC,
		hostBC:       hostBC,
		ownerAddr:    ownerAddr,
		appPubkey:    pub.Bytes(),
	}, nil
}

// Close releases gRPC connections held by the controller.
func (c *Controller) Close() error {
	if c == nil {
		return nil
	}
	var err error
	if c.controllerBC != nil {
		err = c.controllerBC.Close()
	}
	if c.hostBC != nil {
		if closeErr := c.hostBC.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}
	return err
}

// OwnerAddress returns the controller owner address used for ICA registration.
func (c *Controller) OwnerAddress() string {
	return c.ownerAddr
}

// AppPubkey returns the pubkey bytes for ICA creator validation.
func (c *Controller) AppPubkey() []byte {
	return append([]byte(nil), c.appPubkey...)
}

// ICAAddress returns the ICA address if already registered.
func (c *Controller) ICAAddress(ctx context.Context) (string, error) {
	addr, err := c.queryICAAddress(ctx)
	if err != nil {
		return "", fmt.Errorf("query ICA address: %w", err)
	}
	if addr == "" {
		return "", ErrICAAddressNotFound
	}
	return addr, nil
}

// EnsureICAAddress resolves or registers an ICA address and waits for availability.
func (c *Controller) EnsureICAAddress(ctx context.Context) (string, error) {
	addr, err := c.queryICAAddress(ctx)
	if err == nil && addr != "" {
		return addr, nil
	}
	if err := c.preflightConnection(ctx); err != nil {
		return "", err
	}
	if err := c.registerICA(ctx); err != nil {
		return "", err
	}
	var lastErr error
	for i := 0; i < c.cfg.PollRetries; i++ {
		addr, err = c.queryICAAddress(ctx)
		if err == nil && addr != "" {
			return addr, nil
		}
		if err != nil {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(c.cfg.PollDelay):
		}
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", ErrICAAddressNotFound
}

// SendRequestAction sends a request action over ICA and parses the ack for the action ID.
func (c *Controller) SendRequestAction(ctx context.Context, msg *actiontypes.MsgRequestAction) (*sdktypes.ActionResult, error) {
	if msg == nil {
		return nil, fmt.Errorf("msg is nil")
	}
	any, err := PackRequestAny(msg)
	if err != nil {
		return nil, err
	}
	txHash, _, ackBytes, err := c.sendICAAnysWithAck(ctx, []*types.Any{any})
	if err != nil {
		return nil, err
	}
	ids, err := ExtractRequestActionIDsFromAck(ackBytes)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("no action ids in ack")
	}
	return &sdktypes.ActionResult{ActionID: ids[0], TxHash: txHash}, nil
}

// SendApproveAction sends approve messages over ICA and returns the controller tx hash.
func (c *Controller) SendApproveAction(ctx context.Context, msg *actiontypes.MsgApproveAction) (string, error) {
	if msg == nil {
		return "", fmt.Errorf("msg is nil")
	}
	any, err := PackApproveAny(msg)
	if err != nil {
		return "", err
	}
	txHash, _, _, err := c.sendICAAnysWithAck(ctx, []*types.Any{any})
	if err != nil {
		return "", err
	}
	return txHash, nil
}

func (c *Controller) sendICAAnysWithAck(ctx context.Context, anys []*types.Any) (string, PacketInfo, []byte, error) {
	if len(anys) == 0 {
		return "", PacketInfo{}, nil, fmt.Errorf("at least one message is required")
	}
	packet, err := BuildICAPacketData(anys)
	if err != nil {
		return "", PacketInfo{}, nil, err
	}
	msgSendTx, err := BuildMsgSendTx(c.ownerAddr, c.cfg.ConnectionID, uint64(c.cfg.RelativeTimeout.Nanoseconds()), packet)
	if err != nil {
		return "", PacketInfo{}, nil, err
	}
	txBytes, err := c.controllerBC.BuildAndSignTx(ctx, msgSendTx, "")
	if err != nil {
		return "", PacketInfo{}, nil, fmt.Errorf("build and sign tx: %w", err)
	}
	txHash, err := c.controllerBC.Broadcast(ctx, txBytes, txtypes.BroadcastMode_BROADCAST_MODE_SYNC)
	if err != nil {
		return "", PacketInfo{}, nil, fmt.Errorf("broadcast tx: %w", err)
	}
	txResp, err := c.controllerBC.WaitForTxInclusion(ctx, txHash)
	if err != nil {
		return "", PacketInfo{}, nil, fmt.Errorf("wait for tx inclusion: %w", err)
	}
	packetInfo, err := extractPacketInfoFromTxResponse(txResp)
	if err != nil {
		return "", PacketInfo{}, nil, err
	}
	hostPort, hostChannel := c.resolveHostPacketRoute(ctx, packetInfo)
	ackBytes, err := c.waitForAcknowledgement(ctx, hostPort, hostChannel, packetInfo.Sequence)
	if err != nil {
		return "", PacketInfo{}, nil, err
	}
	return txHash, packetInfo, ackBytes, nil
}

func (c *Controller) queryICAAddress(ctx context.Context) (string, error) {
	query := controllertypes.NewQueryClient(c.controllerBC.GRPCConn())
	resp, err := query.InterchainAccount(ctx, &controllertypes.QueryInterchainAccountRequest{
		Owner:        c.ownerAddr,
		ConnectionId: c.cfg.ConnectionID,
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.GetAddress()), nil
}

func (c *Controller) registerICA(ctx context.Context) error {
	version := ""
	if strings.TrimSpace(c.cfg.CounterpartyConnectionID) != "" {
		version = icatypes.NewDefaultMetadataString(c.cfg.ConnectionID, c.cfg.CounterpartyConnectionID)
	}
	msg := controllertypes.NewMsgRegisterInterchainAccount(c.cfg.ConnectionID, c.ownerAddr, version, c.cfg.Ordering)
	txBytes, err := c.controllerBC.BuildAndSignTx(ctx, msg, "")
	if err != nil {
		return fmt.Errorf("build and sign tx: %w", err)
	}
	if _, err := c.controllerBC.Broadcast(ctx, txBytes, txtypes.BroadcastMode_BROADCAST_MODE_SYNC); err != nil {
		return fmt.Errorf("broadcast tx: %w", err)
	}
	return nil
}

func (c *Controller) preflightConnection(ctx context.Context) error {
	connQuery := connectiontypes.NewQueryClient(c.controllerBC.GRPCConn())
	connResp, err := connQuery.Connection(ctx, &connectiontypes.QueryConnectionRequest{ConnectionId: c.cfg.ConnectionID})
	if err != nil {
		return fmt.Errorf("query ibc connection %s: %w", c.cfg.ConnectionID, err)
	}
	conn := connResp.GetConnection()
	if conn == nil {
		return fmt.Errorf("ibc connection %s is empty", c.cfg.ConnectionID)
	}
	clientID := strings.TrimSpace(conn.ClientId)
	if clientID == "" {
		return fmt.Errorf("ibc connection %s has empty client_id", c.cfg.ConnectionID)
	}
	statusResp, err := clienttypes.NewQueryClient(c.controllerBC.GRPCConn()).ClientStatus(ctx, &clienttypes.QueryClientStatusRequest{ClientId: clientID})
	if err != nil {
		return fmt.Errorf("query ibc client status %s: %w", clientID, err)
	}
	if !strings.EqualFold(statusResp.GetStatus(), "Active") {
		return fmt.Errorf("ibc client %s status is %s (expected Active)", clientID, statusResp.GetStatus())
	}
	return nil
}

func (c *Controller) resolveHostPacketRoute(ctx context.Context, info PacketInfo) (string, string) {
	hostPort := info.Port
	hostChannel := info.Channel
	query := channeltypes.NewQueryClient(c.controllerBC.GRPCConn())
	resp, err := query.Channel(ctx, &channeltypes.QueryChannelRequest{PortId: info.Port, ChannelId: info.Channel})
	if err == nil && resp != nil {
		channel := resp.GetChannel()
		if channel != nil {
			cp := channel.Counterparty
			if cp.PortId != "" {
				hostPort = cp.PortId
			}
			if cp.ChannelId != "" {
				hostChannel = cp.ChannelId
			}
		}
	}
	if hostPort == info.Port && strings.HasPrefix(info.Port, "icacontroller-") {
		hostPort = "icahost"
	}
	return hostPort, hostChannel
}

func (c *Controller) waitForAcknowledgement(ctx context.Context, port, channel string, sequence uint64) ([]byte, error) {
	var lastErr error
	for i := 0; i < c.cfg.AckRetries; i++ {
		ack, err := c.queryAcknowledgement(ctx, port, channel, sequence)
		if err == nil {
			return ack, nil
		}
		lastErr = err
		if !errors.Is(err, ErrAckNotFound) {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(c.cfg.PollDelay):
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("acknowledgement not found for %s/%s/%d: %v", port, channel, sequence, lastErr)
	}
	return nil, fmt.Errorf("acknowledgement not found for %s/%s/%d", port, channel, sequence)
}

func (c *Controller) queryAcknowledgement(ctx context.Context, port, channel string, sequence uint64) ([]byte, error) {
	events := []string{
		fmt.Sprintf("write_acknowledgement.packet_dst_port='%s'", port),
		fmt.Sprintf("write_acknowledgement.packet_dst_channel='%s'", channel),
		fmt.Sprintf("write_acknowledgement.packet_sequence='%d'", sequence),
	}
	resp, err := c.hostBC.GetTxsByEvents(ctx, events, 1, 5)
	if err != nil {
		return nil, err
	}
	ack, err := extractAcknowledgement(resp.GetTxResponses(), port, channel, sequence)
	if err != nil {
		return nil, err
	}
	return ack, nil
}

// ErrAckNotFound is returned when no acknowledgement event is present for the packet.
var ErrAckNotFound = errors.New("acknowledgement event not found")

// ErrPacketInfoNotFound is returned when no send_packet event is found in a tx.
var ErrPacketInfoNotFound = errors.New("send_packet event not found")

// ErrICAAddressNotFound is returned when no ICA address is registered yet.
var ErrICAAddressNotFound = errors.New("ica address not found")
