package cascade

import (
	"context"
	"fmt"
	"strings"
	"time"

	sdkEvent "github.com/LumeraProtocol/sdk-go/cascade/event"
	sdkcrypto "github.com/LumeraProtocol/sdk-go/pkg/crypto"
	"github.com/LumeraProtocol/sdk-go/types"
)

// DownloadOptions configures cascade download
type DownloadOptions struct {
	SignerAddr string
}

// DownloadOption is a functional option for Download
type DownloadOption func(*DownloadOptions)

// WithDownloadSignerAddress overrides the signer address used for download signatures.
func WithDownloadSignerAddress(addr string) DownloadOption {
	return func(opts *DownloadOptions) {
		opts.SignerAddr = addr
	}
}

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
		Type:     sdkEvent.SDKGoDownloadStarted,
		ActionID: actionID,
		TaskType: taskType,
		Data: sdkEvent.EventData{
			sdkEvent.KeyMessage: "Cascade download task started",
		},
		Timestamp: time.Now(),
	})

	// Create download signature
	signerAddr := strings.TrimSpace(options.SignerAddr)
	if signerAddr == "" {
		signerAddr = c.config.Address
	}
	if strings.TrimSpace(c.config.ICAOwnerKeyName) != "" && strings.TrimSpace(c.config.ICAOwnerHRP) != "" {
		if options.SignerAddr == "" {
			icaAddr, err := sdkcrypto.AddressFromKey(c.keyring, c.config.ICAOwnerKeyName, c.config.ICAOwnerHRP)
			if err == nil && icaAddr != "" {
				signerAddr = icaAddr
			} else if err != nil {
				c.logf("cascade: failed to derive ICA owner address, using default signer: %v", err)
			}
		}
	}
	signature, err := c.snClient.GenerateDownloadSignature(ctx, actionID, signerAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create signature: %w", err)
	}

	c.logf("cascade: download signature generated, action=%s", actionID)
	c.emitClientEvent(ctx, sdkEvent.Event{
		Type:     sdkEvent.SDKGoDownloadSignatureGenerated,
		ActionID: actionID,
		TaskType: taskType,
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
		Type:     sdkEvent.SDKGoDownloadCompleted,
		ActionID: actionID,
		TaskType: taskType,
		TaskID:   taskID,
		Data: sdkEvent.EventData{
			sdkEvent.KeyMessage: "Cascade download task completed",
		},
		Timestamp: time.Now(),
	})

	return result, nil
}
