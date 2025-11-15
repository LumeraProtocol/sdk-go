package cascade

import (
	"context"
	"fmt"

	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
	"github.com/LumeraProtocol/sdk-go/blockchain"
)

// ApproveOptions configures approve helpers
// Note: these helpers are package-level to avoid clashing with existing method names on Client.
// You must specify the creator address via WithApproveCreator; memo and blockchain client are optional.
// The package also exposes existing (*Client).CreateApproveActionMessage / SendApproveActionMessage
// methods that take explicit parameters.

type ApproveOptions struct {
	Creator    string
	Memo       string
	Blockchain *blockchain.Client
}

// ApproveOption is a functional option for approve helpers
// (package-level variants requested by the ICA design doc)

type ApproveOption func(*ApproveOptions)

// WithApproveCreator sets the creator (Lumera account / ICA host address) for approve helpers
func WithApproveCreator(creator string) ApproveOption {
	return func(o *ApproveOptions) { o.Creator = creator }
}

// WithApproveMemo sets an optional memo (idempotency key or note)
func WithApproveMemo(memo string) ApproveOption {
	return func(o *ApproveOptions) { o.Memo = memo }
}

// WithApproveBlockchain provides the blockchain client used for signing/broadcasting
func WithApproveBlockchain(bc *blockchain.Client) ApproveOption {
	return func(o *ApproveOptions) { o.Blockchain = bc }
}

// CreateApproveActionMessage builds a MsgApproveAction using options for creator address.
// Required: WithApproveCreator(...)
func CreateApproveActionMessage(_ context.Context, actionID string, opts ...ApproveOption) (*actiontypes.MsgApproveAction, error) {
	options := &ApproveOptions{}
	for _, opt := range opts {
		opt(options)
	}
	if options.Creator == "" {
		return nil, fmt.Errorf("creator is required (use WithApproveCreator)")
	}
	if actionID == "" {
		return nil, fmt.Errorf("actionID is required")
	}
	return blockchain.NewMsgApproveAction(options.Creator, actionID), nil
}

// SendApproveActionMessage signs, simulates and broadcasts the approve message using the provided blockchain client option.
// Returns tx hash.
func SendApproveActionMessage(ctx context.Context, msg *actiontypes.MsgApproveAction, opts ...ApproveOption) (string, error) {
	if msg == nil {
		return "", fmt.Errorf("msg is required")
	}
	options := &ApproveOptions{}
	for _, opt := range opts {
		opt(options)
	}
	if options.Blockchain == nil {
		return "", fmt.Errorf("blockchain client is required (use WithApproveBlockchain)")
	}
	ar, err := options.Blockchain.ApproveActionTx(ctx, msg.Creator, msg.ActionId, options.Memo)
	if err != nil {
		return "", err
	}
	return ar.TxHash, nil
}
