package cascade

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateRequestActionMessage(t *testing.T) {
	_, _, err := CreateRequestActionMessage(context.Background(), "", "", nil)
	require.Error(t, err)

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("data"), 0o600))

	_, _, err = CreateRequestActionMessage(context.Background(), "", filePath)
	require.Error(t, err)

	appPubkey := []byte{1, 2, 3}
	msg, meta, err := CreateRequestActionMessage(context.Background(), "creator", filePath,
		WithICACreatorAddress("ica-creator"),
		WithAppPubkey(appPubkey),
		WithPublic(true),
	)
	require.NoError(t, err)
	require.Equal(t, "ica-creator", msg.Creator)
	require.Equal(t, appPubkey, msg.AppPubkey)
	require.NotEmpty(t, msg.Metadata)
	require.Equal(t, msg.Metadata, string(meta))

	var decoded struct {
		File   string `json:"file"`
		Size   int64  `json:"size"`
		Public bool   `json:"public"`
	}
	require.NoError(t, json.Unmarshal(meta, &decoded))
	require.Equal(t, "file.txt", decoded.File)
	require.Equal(t, int64(4), decoded.Size)
	require.True(t, decoded.Public)
}
