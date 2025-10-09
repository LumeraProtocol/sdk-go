package cascade

import (
	"context"
	"encoding/json"
	"fmt"

	txv1beta1 "cosmossdk.io/api/cosmos/tx/v1beta1"
	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
	"github.com/LumeraProtocol/sdk-go/blockchain"
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
func (c *Client) Upload(ctx context.Context, bc *blockchain.Client, filePath string, opts ...UploadOption) (*types.CascadeResult, error) {
	// Apply options
	options := &UploadOptions{
		Public: false,
	}
	for _, opt := range opts {
		opt(options)
	}

	// Register Action in blockchain
	// Crearte metadata
	meta, price, expiration, err := c.snClient.BuildCascadeMetadataFromFile(ctx, filePath, options.Public)
	if err != nil {
		return nil, fmt.Errorf("failed to build metadata: %w", err)
	}
	metaBytes, err := json.Marshal(&meta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Build RequestAction Message
	msg := blockchain.NewMsgRequestAction(
		"",
		actiontypes.ActionTypeCascade,
		string(metaBytes),
		price,
		expiration,
	)

	// Build Transaction
	txBytes, err := bc.BuildAndSignTx(ctx, msg, options.FileName)
	if err != nil {
		return nil, fmt.Errorf("failed to build transaction: %w", err)
	}

	//Broadcast Transaction
	txHash, err := bc.Broadcast(ctx, txBytes, txv1beta1.BroadcastMode_BROADCAST_MODE_SYNC)
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	// Fetch Action ID from the transaction result
	txResult, err := bc.GetTx(ctx, txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tx result: %w", err)
	}
	if txResult == nil || len(txResult.Logs) == 0 || len(txResult.Logs[0].Events) == 0 {
		return nil, fmt.Errorf("invalid tx result")
	}
	var actionID string
	for _, event := range txResult.Logs[0].Events {
		if event.Type == "action_requested" {
			for _, attr := range event.Attributes {
				if attr.Key == "action_id" {
					actionID = attr.Value
					break
				}
			}
		}
	}
	if actionID == "" {
		return nil, fmt.Errorf("action_id not found in tx events")
	}

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

	// Wait for task completion
	task, err := c.tasks.Wait(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("cascade failed: %w", err)
	}

	return &types.CascadeResult{
		ActionID: actionID,
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
		ActionID: actionID,
		TaskID:   task.TaskID,
	}, nil
}
