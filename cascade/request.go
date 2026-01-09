package cascade

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
	"github.com/LumeraProtocol/sdk-go/blockchain"
)

// CreateRequestActionMessage is a package-level helper that builds a MsgRequestAction
// for Cascade using only local file information (no network calls).
// It returns the constructed message and the metadata bytes used inside it.
//
// This function mirrors (*Client).CreateRequestActionMessage but avoids depending on
// SuperNode SDK; it prepares minimal metadata from the provided file path so it can
// be used in offline ICA examples/tests.
func CreateRequestActionMessage(_ context.Context, creator string, filePath string, opts ...UploadOption) (*actiontypes.MsgRequestAction, []byte, error) { //nolint:revive
	if filePath == "" {
		return nil, nil, fmt.Errorf("filePath is required")
	}

	// Minimal metadata based on local file properties
	type meta struct {
		File   string `json:"file"`
		Size   int64  `json:"size"`
		Public bool   `json:"public"`
	}

	fi, err := os.Stat(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("stat file: %w", err)
	}

	// Apply options (default: not public)
	options := &UploadOptions{Public: false}
	for _, opt := range opts {
		opt(options)
	}

	effectiveCreator := creator
	if options.ICACreatorAddress != "" {
		effectiveCreator = options.ICACreatorAddress
	}
	if effectiveCreator == "" {
		return nil, nil, fmt.Errorf("creator is required")
	}

	m := meta{File: filepath.Base(filePath), Size: fi.Size(), Public: options.Public}
	metaBytes, err := json.Marshal(&m)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal metadata: %w", err)
	}

	// Construct the action message (no price/expiration in offline example)
	fileSizeKbs := (fi.Size() + 1023) / 1024
	msg := blockchain.NewMsgRequestAction(effectiveCreator, actiontypes.ActionTypeCascade, string(metaBytes), "", "0", fileSizeKbs)
	if len(options.AppPubkey) > 0 {
		msg.AppPubkey = options.AppPubkey
	}
	return msg, metaBytes, nil
}
