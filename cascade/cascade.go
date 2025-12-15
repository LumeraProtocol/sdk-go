package cascade

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
	"path/filepath"

	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
	"github.com/LumeraProtocol/sdk-go/blockchain"
	sdkEvent "github.com/LumeraProtocol/sdk-go/cascade/event"
	sdklog "github.com/LumeraProtocol/sdk-go/pkg/log"
	"github.com/LumeraProtocol/sdk-go/types"
)

// UploadOptions configures cascade upload
type UploadOptions struct {
	ID     string // optional custom ID for the upload
	Public bool   // whether the uploaded file will be accessible publicly
}

// UploadOption is a functional option for Upload
type UploadOption func(*UploadOptions)

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
// CreateRequestActionMessage builds Cascade metadata and constructs a MsgRequestAction without broadcasting it.
// Returns the built Cosmos message and the serialized metadata bytes used in the message.
func (c *Client) CreateRequestActionMessage(ctx context.Context, creator string, filePath string, options *UploadOptions) (*actiontypes.MsgRequestAction, []byte, error) {
	var isPublic bool
	if options != nil {
		isPublic = options.Public
	}
	// Build metadata with SuperNode SDK
	meta, price, expiration, err := c.snClient.BuildCascadeMetadataFromFile(ctx, filePath, isPublic)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build metadata: %w", err)
	}
	metaBytes, err := json.Marshal(&meta)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	c.logf("cascade: built metadata for %s (public=%t price=%s expires=%s)", filePath, isPublic, price, expiration)

	// Construct the action message
	fi, err := os.Stat(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("stat file: %w", err)
	}
	fileSizeKbs := int64(0)
	if fi != nil && fi.Size() > 0 {
		fileSizeKbs = (fi.Size() + 1023) / 1024
	}
	msg := blockchain.NewMsgRequestAction(creator, actiontypes.ActionTypeCascade, string(metaBytes), price, expiration, fileSizeKbs)
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
		TaskID: id,
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
			sdkEvent.KeyMessage:    "Action registration confirmed",
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
	// emit upload started event
	c.emitClientEvent(ctx, sdkEvent.Event{
		Type:      sdkEvent.SDKGoUploadStarted,
		ActionID:  actionID,
		TaskID:    taskID,
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

// Upload provides a one-shot convenience helper that performs:
//   1) CreateRequestActionMessage
//   2) SendRequestActionMessage
//   3) UploadToSupernode
func (c *Client) Upload(ctx context.Context, creator string, bc *blockchain.Client, filePath string, opts ...UploadOption) (*types.CascadeResult, error) {

	// Apply upload options
	options := &UploadOptions{Public: false}
	for _, opt := range opts {
		opt(options)
	}

	// Build message
	msg, _, err := c.CreateRequestActionMessage(ctx, creator, filePath, options)
	if err != nil {
		return nil, err
	}

	// extract filename from path
	fileName := filepath.Base(filePath)

	// Broadcast
	ar, err := c.SendRequestActionMessage(ctx, bc, msg, fileName, options)
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

	taskType := string(types.ActionTypeCascade)
	c.logf("cascade: starting download, action=%s dest=%s", actionID, outputDir)
	c.emitClientEvent(ctx, sdkEvent.Event{
		Type:      sdkEvent.SDKGoDownloadStarted,
		ActionID:  actionID,
		TaskType:  taskType,
		Data: sdkEvent.EventData{
			sdkEvent.KeyMessage: "Cascade download task started",
		},
		Timestamp: time.Now(),
	})

	// Create download signature
	signature, err := c.snClient.GenerateDownloadSignature(ctx, actionID, c.config.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to create signature: %w", err)
	}

	c.logf("cascade: download signature generated, action=%s", actionID)
	c.emitClientEvent(ctx, sdkEvent.Event{
		Type:      sdkEvent.SDKGoDownloadSignatureGenerated,
		ActionID:  actionID,
		TaskType:  taskType,
		Data: sdkEvent.EventData{
			sdkEvent.KeyMessage: "Cascade download signature generated",
		},
		Timestamp: time.Now(),
	})

	// Start download via SuperNode SDK
	taskID, err := c.snClient.DownloadCascade(ctx, actionID, outputDir, signature)
	if err != nil {
		return nil, fmt.Errorf("failed to start download: %w", err)
	}

	// Wait for completion
	task, err := c.tasks.Wait(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}

	result := &types.DownloadResult{
		ActionID:   actionID,
		TaskID:     task.TaskID,
		OutputPath: outputDir + "/" + actionID,
	}

	c.logf("cascade: download completed action_id=%s task_id=%s", actionID, taskID)
	c.emitClientEvent(ctx, sdkEvent.Event{
		Type:      sdkEvent.SDKGoDownloadCompleted,
		ActionID:  actionID,
		TaskType:  taskType,
		TaskID:    taskID,
		Data: sdkEvent.EventData{
			sdkEvent.KeyMessage: "Cascade download task completed",
		},
		Timestamp: time.Now(),
	})

	return result, nil
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
