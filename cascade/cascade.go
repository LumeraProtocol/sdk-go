package cascade

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
	"github.com/LumeraProtocol/sdk-go/blockchain"
	sdkEvent "github.com/LumeraProtocol/sdk-go/cascade/event"
	sdklog "github.com/LumeraProtocol/sdk-go/pkg/log"
	"github.com/LumeraProtocol/sdk-go/types"
)

// UploadOptions configures cascade upload
type UploadOptions struct {
	Public   bool
	FileName string
}

// UploadOption is a functional option for Upload
type UploadOption func(*UploadOptions)

// WithPublic sets the public flag
func WithPublic(public bool) UploadOption {
	return func(o *UploadOptions) {
		o.Public = public
	}
}

// WithFileName sets a custom filename
func WithFileName(name string) UploadOption {
	return func(o *UploadOptions) {
		o.FileName = name
	}
}

// CreateRequestActionMessage builds Cascade metadata and constructs a MsgRequestAction without broadcasting it.
// Returns the built Cosmos message and the serialized metadata bytes used in the message.
func (c *Client) CreateRequestActionMessage(ctx context.Context, creator string, filePath string, opts ...UploadOption) (*actiontypes.MsgRequestAction, []byte, error) {
	// Apply options
	options := &UploadOptions{Public: false}
	for _, opt := range opts {
		opt(options)
	}

	// Build metadata with SuperNode SDK
	meta, price, expiration, err := c.snClient.BuildCascadeMetadataFromFile(ctx, filePath, options.Public)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build metadata: %w", err)
	}
	metaBytes, err := json.Marshal(&meta)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	c.logf("cascade: built metadata for %s (public=%t price=%s expires=%s)", filePath, options.Public, price, expiration)

	// Construct the action message
	msg := blockchain.NewMsgRequestAction(creator, actiontypes.ActionTypeCascade, string(metaBytes), price, expiration)
	return msg, metaBytes, nil
}

// SendRequestActionMessage signs, simulates and broadcasts the provided request message.
// "memo" can be used to pass an optional filename or idempotency key.
func (c *Client) SendRequestActionMessage(ctx context.Context, bc *blockchain.Client, msg *actiontypes.MsgRequestAction, memo string) (*types.ActionResult, error) {
	if bc == nil || msg == nil {
		return nil, fmt.Errorf("blockchain client and msg are required")
	}
	// Convert string ActionType back to enum expected by blockchain client
	at := actiontypes.ActionType(0)
	if v, ok := actiontypes.ActionType_value[msg.ActionType]; ok {
		at = actiontypes.ActionType(v)
	}

	// Request Action transaction
	taskType := normalizeTaskType(msg.GetActionType())
	c.emitClientEvent(ctx, sdkEvent.Event{
		Type:      sdkEvent.SDKActionRegistrationRequested,
		TaskType:  taskType,
		Timestamp: time.Now(),
		Data: sdkEvent.EventData{
			sdkEvent.KeyEventType:  taskType,
			sdkEvent.KeyPrice:      msg.Price,
			sdkEvent.KeyExpiration: msg.ExpirationTime,
		},
	})

	c.logf("cascade: submitting request action tx creator=%s memo=%s price=%s expires=%s", msg.Creator, memo, msg.Price, msg.ExpirationTime)
	ar, err := bc.RequestActionTx(ctx, msg.Creator, at, msg.Metadata, msg.Price, msg.ExpirationTime, memo)
	if err != nil {
		return nil, err
	}

	actionID := ar.ActionID
	c.emitClientEvent(ctx, sdkEvent.Event{
		Type:      sdkEvent.SDKActionRegistrationConfirmed,
		ActionID:  actionID,
		TaskType:  taskType,
		Timestamp: time.Now(),
		Data: sdkEvent.EventData{
			sdkEvent.KeyActionID:    actionID,
			sdkEvent.KeyTxHash:      ar.TxHash,
			sdkEvent.KeyBlockHeight: ar.Height,
		},
	})
	c.logf("cascade: request action confirmed action_id=%s height=%d tx=%s", actionID, ar.Height, ar.TxHash)

	return ar, err
}

// CreateApproveActionMessage constructs a MsgApproveAction without broadcasting it.
func (c *Client) CreateApproveActionMessage(_ context.Context, creator string, actionID string) (*actiontypes.MsgApproveAction, error) {
	if creator == "" || actionID == "" {
		return nil, fmt.Errorf("creator and actionID are required")
	}
	return blockchain.NewMsgApproveAction(creator, actionID), nil
}

// SendApproveActionMessage signs, simulates and broadcasts the provided approve message.
func (c *Client) SendApproveActionMessage(ctx context.Context, bc *blockchain.Client, msg *actiontypes.MsgApproveAction, memo string) (*types.ActionResult, error) {
	if bc == nil || msg == nil {
		return nil, fmt.Errorf("blockchain client and msg are required")
	}
	return bc.ApproveActionTx(ctx, msg.Creator, msg.ActionId, memo)
}

// UploadToSupernode uploads the file bytes to SuperNodes keyed by actionID and waits for completion.
// Returns the resulting taskID upon success.
func (c *Client) UploadToSupernode(ctx context.Context, actionID string, filePath string) (string, error) {
	if actionID == "" || filePath == "" {
		return "", fmt.Errorf("actionID and filePath are required")
	}

	// Create a file signature for upload
	signature, err := c.snClient.GenerateStartCascadeSignatureFromFile(ctx, filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create signature: %w", err)
	}

	// Start cascade upload via SuperNode SDK
	c.logf("cascade: starting supernode upload action_id=%s file=%s", actionID, filePath)
	taskID, err := c.snClient.StartCascade(ctx, filePath, actionID, signature)
	if err != nil {
		return "", fmt.Errorf("failed to start cascade: %w", err)
	}

	// Wait for task completion
	if task, err := c.tasks.Wait(ctx, taskID); err != nil {
		return "", fmt.Errorf("cascade failed: %w", err)
	} else {
		c.logf("cascade: supernode upload completed action_id=%s task_id=%s", actionID, task.TaskID)
		return task.TaskID, nil
	}
}

// Upload provides a one-shot convenience helper that performs:
// 1) CreateRequestActionMessage 2) SendRequestActionMessage 3) UploadToSupernode
func (c *Client) Upload(ctx context.Context, creator string, bc *blockchain.Client, filePath string, opts ...UploadOption) (*types.CascadeResult, error) {
	// Build message
	msg, _, err := c.CreateRequestActionMessage(ctx, creator, filePath, opts...)
	if err != nil {
		return nil, err
	}

	// Broadcast
	options := &UploadOptions{Public: false}
	for _, opt := range opts {
		opt(options)
	}
	ar, err := c.SendRequestActionMessage(ctx, bc, msg, options.FileName)
	if err != nil {
		return nil, fmt.Errorf("request action tx: %w", err)
	}

	// Upload bytes off-chain
	taskID, err := c.UploadToSupernode(ctx, ar.ActionID, filePath)
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

// DownloadOptions configures cascade download
type DownloadOptions struct {
	// Add download options as needed
}

// DownloadOption is a functional option for Download
type DownloadOption func(*DownloadOptions)

// Download downloads a file from Cascade
func (c *Client) Download(ctx context.Context, actionID string, outputDir string, opts ...DownloadOption) (*types.DownloadResult, error) {
	// Apply options
	options := &DownloadOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Create download signature
	signature, err := c.snClient.GenerateDownloadSignature(ctx, actionID, c.config.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to create signature: %w", err)
	}

	// Start download via SuperNode SDK
	c.logf("cascade: starting download action_id=%s dest=%s", actionID, outputDir)
	taskID, err := c.snClient.DownloadCascade(ctx, actionID, outputDir, signature)
	if err != nil {
		return nil, fmt.Errorf("failed to start download: %w", err)
	}

	// Wait for completion
	task, err := c.tasks.Wait(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	c.logf("cascade: download completed action_id=%s task_id=%s", actionID, task.TaskID)

	return &types.DownloadResult{
		ActionID:   actionID,
		TaskID:     task.TaskID,
		OutputPath: outputDir + "/" + actionID,
	}, nil
}

func (c *Client) emitClientEvent(ctx context.Context, evt sdkEvent.Event) {
	if c.isLocalEventType(evt.Type) {
		c.emitLocalEvent(ctx, evt)
	}
}

func (c *Client) logf(format string, args ...interface{}) {
	if c.logger == nil {
		return
	}
	sdklog.Infof(c.logger, format, args...)
}

func normalizeTaskType(raw string) string {
	if raw == "" {
		return ""
	}
	if val, ok := actiontypes.ActionType_value[raw]; ok {
		switch actiontypes.ActionType(val) {
		case actiontypes.ActionTypeCascade:
			return string(types.ActionTypeCascade)
		case actiontypes.ActionTypeSense:
			return string(types.ActionTypeSense)
		}
	}
	return raw
}
