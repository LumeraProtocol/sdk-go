package cascade

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
	"github.com/LumeraProtocol/sdk-go/blockchain"
	sdkEvent "github.com/LumeraProtocol/sdk-go/cascade/event"
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

// Upload uploads a file using Cascade
func (c *Client) Upload(ctx context.Context, creator string, bc *blockchain.Client, filePath string, opts ...UploadOption) (*types.CascadeResult, error) {
	// Apply options
	options := &UploadOptions{
		Public: false,
	}
	for _, opt := range opts {
		opt(options)
	}

	// Register Action in blockchain
	// Create metadata
	meta, price, expiration, err := c.snClient.BuildCascadeMetadataFromFile(ctx, filePath, options.Public)
	if err != nil {
		return nil, fmt.Errorf("failed to build metadata: %w", err)
	}
	metaBytes, err := json.Marshal(&meta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Request Action transaction
	c.emitClientEvent(ctx, sdkEvent.Event{
		Type:      sdkEvent.SDKActionRegistrationRequested,
		TaskType:  "CASCADE",
		Timestamp: time.Now(),
		Data: sdkEvent.EventData{
			sdkEvent.KeyEventType:  actiontypes.ActionTypeCascade.String(),
			sdkEvent.KeyMessage:    options.FileName,
			sdkEvent.KeyPrice:      price,
			sdkEvent.KeyExpiration: expiration,
			sdkEvent.KeyFilePath:   filePath,
		},
	})
	ar, err := bc.RequestActionTx(ctx, creator, actiontypes.ActionTypeCascade, string(metaBytes), price, expiration, options.FileName)
	if err != nil {
		c.logf("request action tx failed: creator=%s file=%s err=%v", creator, filePath, err)
		return nil, fmt.Errorf("request action tx: %w", err)
	}
	actionID := ar.ActionID
	regHeight := ar.Height
	c.emitClientEvent(ctx, sdkEvent.Event{
		Type:      sdkEvent.SDKActionRegistrationConfirmed,
		ActionID:  actionID,
		TaskType:  "CASCADE",
		Timestamp: time.Now(),
		Data: sdkEvent.EventData{
			sdkEvent.KeyActionID:    actionID,
			sdkEvent.KeyTxHash:      ar.TxHash,
			sdkEvent.KeyBlockHeight: regHeight,
		},
	})

	// Upload file to SN for processing
	// Create a file signature
	signature, err := c.snClient.GenerateStartCascadeSignatureFromFile(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create signature: %w", err)
	}

	// Start cascade upload via SuperNode SDK
	taskID, err := c.snClient.StartCascade(ctx, filePath, actionID, signature)
	if err != nil {
		return nil, fmt.Errorf("failed to start cascade: %w", err)
	}
	c.emitClientEvent(ctx, sdkEvent.Event{
		Type:      sdkEvent.SDKCascadeTaskStarted,
		ActionID:  actionID,
		TaskType:  "CASCADE",
		Timestamp: time.Now(),
		Data: sdkEvent.EventData{
			sdkEvent.KeyTaskID:   taskID,
			sdkEvent.KeyActionID: actionID,
			sdkEvent.KeyFilePath: filePath,
		},
	})

	// Wait for task completion
	task, err := c.tasks.Wait(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("cascade failed: %w", err)
	}

	return &types.CascadeResult{
		ActionResult: types.ActionResult{
			ActionID: actionID,
			TxHash:   ar.TxHash,
			Height:   regHeight,
		},
		TaskID:   task.TaskID,
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
	taskID, err := c.snClient.DownloadCascade(ctx, actionID, outputDir, signature)
	if err != nil {
		return nil, fmt.Errorf("failed to start download: %w", err)
	}

	// Wait for completion
	task, err := c.tasks.Wait(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}

	return &types.DownloadResult{
		ActionID:   actionID,
		TaskID:     task.TaskID,
		OutputPath: outputDir + "/" + actionID,
	}, nil
}

func (c *Client) logf(format string, args ...interface{}) {
	if c.logger != nil {
		c.logger.Printf(format, args...)
	}
}

func (c *Client) emitClientEvent(ctx context.Context, evt sdkEvent.Event) {
	if c.isLocalEventType(evt.Type) {
		c.emitLocalEvent(ctx, evt)
	}
}
