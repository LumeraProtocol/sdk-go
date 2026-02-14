package ica

import (
	"errors"
	"time"

	"github.com/LumeraProtocol/sdk-go/blockchain/base"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
)

const (
	// defaultRelativeTimeout is the default ICA packet timeout (10 minutes).
	defaultRelativeTimeout = 10 * time.Minute
	// defaultPollDelay is the default interval between polling attempts (2 seconds).
	defaultPollDelay = 2 * time.Second
	// defaultPollRetries is the default max attempts when polling for ICA registration.
	defaultPollRetries = 120
	// defaultAckRetries is the default max attempts when polling for an IBC acknowledgement.
	defaultAckRetries = 120
)

// Config configures the ICA controller for a controller/host chain pair.
// Controller and host chains can use different key types (Cosmos or EVM)
// by importing keys under separate names into the same keyring and setting
// KeyName / HostKeyName independently.
type Config struct {
	// Controller holds the gRPC/chain configuration for the controller chain
	// (the chain that signs and broadcasts MsgSendTx).
	Controller base.Config
	// Host holds the gRPC/chain configuration for the host chain
	// (the chain where the ICA executes messages).
	Host base.Config
	// Keyring provides access to signing keys for both chains.
	Keyring keyring.Keyring
	// KeyName is the key name used for signing on the controller chain.
	KeyName string
	// HostKeyName is an optional separate key name for host chain operations.
	// When empty, KeyName is used for both chains.
	HostKeyName string

	// ConnectionID is the IBC connection identifier on the controller chain.
	ConnectionID string
	// CounterpartyConnectionID is the IBC connection identifier on the host chain.
	// When set, it is included in the ICA registration metadata.
	CounterpartyConnectionID string
	// Ordering specifies the IBC channel ordering (ORDERED or UNORDERED).
	// Defaults to ORDERED.
	Ordering channeltypes.Order
	// RelativeTimeout is the timeout duration for ICA packets, relative to
	// the current block time. Defaults to 10 minutes.
	RelativeTimeout time.Duration
	// PollDelay is the interval between polling attempts when waiting for
	// ICA registration or acknowledgements. Defaults to 2 seconds.
	PollDelay time.Duration
	// PollRetries is the maximum number of polling attempts when waiting
	// for ICA address registration. Defaults to 120.
	PollRetries int
	// AckRetries is the maximum number of polling attempts when waiting
	// for an IBC packet acknowledgement. Defaults to 120.
	AckRetries int
}

// Controller manages ICA registration and message execution via gRPC.
type Controller struct {
	cfg          Config       // full configuration snapshot
	controllerBC *base.Client // gRPC client for the controller chain
	hostBC       *base.Client // gRPC client for the host chain
	ownerAddr    string       // bech32 owner address on the controller chain
	appPubkey    []byte       // compressed public key bytes for ICA creator validation
}

// ErrAckNotFound is returned when no acknowledgement event is present for the packet.
var ErrAckNotFound = errors.New("acknowledgement event not found")

// ErrPacketInfoNotFound is returned when no send_packet event is found in a tx.
var ErrPacketInfoNotFound = errors.New("send_packet event not found")

// ErrICAAddressNotFound is returned when no ICA address is registered yet.
var ErrICAAddressNotFound = errors.New("ica address not found")
