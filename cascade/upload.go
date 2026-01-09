package cascade

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
	"github.com/LumeraProtocol/sdk-go/blockchain"
	sdkEvent "github.com/LumeraProtocol/sdk-go/cascade/event"
	"github.com/LumeraProtocol/sdk-go/types"
)

// UploadOptions configures cascade upload
type UploadOptions struct {
	ID                string // optional custom ID for the upload
	Public            bool   // whether the uploaded file will be accessible publicly
	ICACreatorAddress string // optional ICA creator address used in MsgRequestAction
	AppPubkey         []byte // optional app pubkey for ICA creator validation
	ICASendFunc       ICASendFunc
}

// UploadOption is a functional option for Upload
type UploadOption func(*UploadOptions)

// ICASendFunc sends a MsgRequestAction via ICA and returns the resulting action result.
// It should be used when registering actions through an interchain account.
type ICASendFunc func(ctx context.Context, msg *actiontypes.MsgRequestAction, meta []byte, filePath string, options *UploadOptions) (*types.ActionResult, error)

// WithPublic sets the public flag
func WithPublic(public bool) UploadOption {
	return func(o *UploadOptions) {
		o.Public = public
	}
}

// WithID sets a custom ID
func WithID(id string) UploadOption {
	return func(o *UploadOptions) {
		o.ID = id
	}
}

// WithICACreatorAddress sets the ICA creator address on MsgRequestAction.
func WithICACreatorAddress(addr string) UploadOption {
	return func(o *UploadOptions) {
		o.ICACreatorAddress = addr
	}
}

// WithAppPubkey sets the app pubkey used for ICA creator signature validation.
func WithAppPubkey(pubkey []byte) UploadOption {
	return func(o *UploadOptions) {
		o.AppPubkey = pubkey
	}
}

// WithICASendFunc provides a hook to send the request message via ICA.
func WithICASendFunc(fn ICASendFunc) UploadOption {
	return func(o *UploadOptions) {
		o.ICASendFunc = fn
	}
}

// CreateRequestActionMessage builds Cascade metadata and constructs a MsgRequestAction without broadcasting it.
// Returns the built Cosmos message and the serialized metadata bytes used in the message.
func (c *Client) CreateRequestActionMessage(ctx context.Context, creator string, filePath string, options *UploadOptions) (*actiontypes.MsgRequestAction, []byte, error) {
	var isPublic bool
	effectiveCreator := creator
	if options != nil {
		isPublic = options.Public
		if options.ICACreatorAddress != "" {
			effectiveCreator = options.ICACreatorAddress
		}
	}
	if effectiveCreator == "" {
		return nil, nil, fmt.Errorf("creator is required")
	}
	// Build metadata with SuperNode SDK
	signerAddr := ""
	if options != nil {
		signerAddr = options.ICACreatorAddress
	}
	meta, price, expiration, err := c.snClient.BuildCascadeMetadataFromFile(ctx, filePath, isPublic, signerAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build metadata: %w", err)
	}
	metaBytes, err := json.Marshal(&meta)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	c.logf("cascade: built metadata for %s (public=%t price=%s expires=%s)", filePath, isPublic, price, expiration)

	// Construct the action message
	fileSizeKbs := int64(0)
	if sizeKB, err := fileSizeKB(filePath); err == nil {
		fileSizeKbs = sizeKB
	}
	msg := blockchain.NewMsgRequestAction(effectiveCreator, actiontypes.ActionTypeCascade, string(metaBytes), price, expiration, fileSizeKbs)
	if options != nil && len(options.AppPubkey) > 0 {
		msg.AppPubkey = options.AppPubkey
	}
	return msg, metaBytes, nil
}

// SendRequestActionMessage signs, simulates and broadcasts the provided request message.
// "memo" can be used to pass an optional filename or idempotency key.
func (c *Client) SendRequestActionMessage(ctx context.Context, bc *blockchain.Client, msg *actiontypes.MsgRequestAction,
	memo string, options *UploadOptions) (*types.ActionResult, error) {
	if bc == nil || msg == nil {
		return nil, fmt.Errorf("blockchain client and msg are required")
	}
	// Convert string ActionType back to enum expected by blockchain client
	at := actiontypes.ActionType(0)
	if v, ok := actiontypes.ActionType_value[msg.ActionType]; ok {
		at = actiontypes.ActionType(v)
	}

	var id string
	if options != nil {
		id = options.ID
	}

	// Request Action transaction
	taskType := normalizeTaskType(msg.GetActionType())
	c.emitClientEvent(ctx, sdkEvent.Event{
		Type:      sdkEvent.SDKGoActionRegistrationRequested,
		TaskType:  taskType,
		TaskID:    id,
		Timestamp: time.Now(),
		Data: sdkEvent.EventData{
			sdkEvent.KeyPrice:      msg.Price,
			sdkEvent.KeyExpiration: msg.ExpirationTime,
			sdkEvent.KeyMessage:    "Action registration requested",
		},
	})

	c.logf("cascade: submitting request action tx creator=%s memo=%s price=%s expires=%s", msg.Creator, memo, msg.Price, msg.ExpirationTime)
	fileSizeKbs := int64(0)
	if msg.FileSizeKbs != "" {
		// Msg already validated in chain; best-effort parse here.
		if parsed, err := strconv.ParseInt(msg.FileSizeKbs, 10, 64); err == nil {
			fileSizeKbs = parsed
		}
	}
	ar, err := bc.RequestActionTx(ctx, msg.Creator, at, msg.Metadata, msg.Price, msg.ExpirationTime, fileSizeKbs, memo)
	if err != nil {
		return nil, err
	}

	actionID := ar.ActionID
	c.emitClientEvent(ctx, sdkEvent.Event{
		Type:      sdkEvent.SDKGoActionRegistrationConfirmed,
		ActionID:  actionID,
		TaskID:    id,
		TaskType:  taskType,
		Timestamp: time.Now(),
		Data: sdkEvent.EventData{
			sdkEvent.KeyTxHash:      ar.TxHash,
			sdkEvent.KeyBlockHeight: ar.Height,
			sdkEvent.KeyMessage:     "Action registration confirmed",
		},
	})
	c.logf("cascade: request action confirmed action_id=%s height=%d tx=%s", actionID, ar.Height, ar.TxHash)

	return ar, err
}

// UploadToSupernode uploads the file bytes to SuperNodes keyed by actionID and waits for completion.
// Optional signerAddr overrides the bech32 address used for ADR-36 signing.
// Returns the resulting taskID upon success.
func (c *Client) UploadToSupernode(ctx context.Context, actionID string, filePath string, signerAddr ...string) (string, error) {
	if actionID == "" || filePath == "" {
		return "", fmt.Errorf("actionID and filePath are required")
	}

	// Create a file signature for upload
	adrSigner := ""
	if len(signerAddr) > 0 {
		adrSigner = strings.TrimSpace(signerAddr[0])
	}
	var signature string
	var err error
	if adrSigner != "" {
		if icaSigner, ok := c.snClient.(interface {
			GenerateStartCascadeSignatureFromFileWithSigner(context.Context, string, string) (string, error)
		}); ok {
			signature, err = icaSigner.GenerateStartCascadeSignatureFromFileWithSigner(ctx, filePath, adrSigner)
		} else {
			c.logf("cascade: signer override requested but not supported by supernode client; using default signer")
			signature, err = c.snClient.GenerateStartCascadeSignatureFromFile(ctx, filePath)
		}
	} else {
		signature, err = c.snClient.GenerateStartCascadeSignatureFromFile(ctx, filePath)
	}
	if err != nil {
		return "", fmt.Errorf("failed to create signature: %w", err)
	}

	// Start cascade upload via SuperNode SDK
	c.logf("cascade: starting supernode upload action_id=%s file=%s", actionID, filePath)
	taskID, err := c.snClient.StartCascade(ctx, filePath, actionID, signature)
	if err != nil {
		return "", fmt.Errorf("failed to start cascade: %w", err)
	}
	// emit upload started event
	c.emitClientEvent(ctx, sdkEvent.Event{
		Type:     sdkEvent.SDKGoUploadStarted,
		ActionID: actionID,
		TaskID:   taskID,
		Data: sdkEvent.EventData{
			sdkEvent.KeyMessage: "Cascade upload task started",
		},
		Timestamp: time.Now(),
	})

	// Wait for task completion
	if task, err := c.tasks.Wait(ctx, taskID); err != nil {
		return "", fmt.Errorf("cascade failed: %w", err)
	} else {
		c.logf("cascade: supernode upload completed action_id=%s task_id=%s", actionID, task.TaskID)
		return task.TaskID, nil
	}
}

func fileSizeKB(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return (info.Size() + 1023) / 1024, nil
}

// Upload provides a one-shot convenience helper that performs:
//  1. CreateRequestActionMessage
//  2. SendRequestActionMessage
//  3. UploadToSupernode
func (c *Client) Upload(ctx context.Context, creator string, bc *blockchain.Client, filePath string, opts ...UploadOption) (*types.CascadeResult, error) {

	// Apply upload options
	options := &UploadOptions{Public: false}
	for _, opt := range opts {
		opt(options)
	}

	// Build message
	msg, meta, err := c.CreateRequestActionMessage(ctx, creator, filePath, options)
	if err != nil {
		return nil, err
	}

	// extract filename from path
	fileName := filepath.Base(filePath)

	var ar *types.ActionResult
	if options.ICASendFunc != nil {
		ar, err = options.ICASendFunc(ctx, msg, meta, filePath, options)
		if err != nil {
			return nil, fmt.Errorf("ica request action tx: %w", err)
		}
		if ar == nil || ar.ActionID == "" {
			return nil, fmt.Errorf("ica send func returned empty action id")
		}
	} else {
		if options.ICACreatorAddress != "" || len(options.AppPubkey) > 0 {
			return nil, fmt.Errorf("ica options require WithICASendFunc")
		}
		// Broadcast
		ar, err = c.SendRequestActionMessage(ctx, bc, msg, fileName, options)
		if err != nil {
			return nil, fmt.Errorf("request action tx: %w", err)
		}
	}

	// Upload bytes off-chain
	signer := ""
	if options.ICACreatorAddress != "" {
		signer = options.ICACreatorAddress
	}
	taskID, err := c.UploadToSupernode(ctx, ar.ActionID, filePath, signer)
	if err != nil {
		return nil, err
	}

	return &types.CascadeResult{
		ActionResult: types.ActionResult{
			ActionID: ar.ActionID,
			TxHash:   ar.TxHash,
			Height:   ar.Height,
		},
		TaskID: taskID,
	}, nil
}
