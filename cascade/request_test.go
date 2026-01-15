package cascade

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
	pb "github.com/LumeraProtocol/supernode/v2/gen/supernode"
	"github.com/LumeraProtocol/supernode/v2/sdk/event"
	"github.com/LumeraProtocol/supernode/v2/sdk/task"
	"github.com/stretchr/testify/require"
)

type fakeSNClient struct {
	meta       actiontypes.CascadeMetadata
	price      string
	expiration string
	errOnEmpty error

	lastFile   string
	lastPublic bool
	lastSigner string
	calls      int
}

func (f *fakeSNClient) StartCascade(_ context.Context, _ string, _ string, _ string) (string, error) {
	return "", nil
}

func (f *fakeSNClient) DeleteTask(_ context.Context, _ string) error {
	return nil
}

func (f *fakeSNClient) GetTask(_ context.Context, _ string) (*task.TaskEntry, bool) {
	return nil, false
}

func (f *fakeSNClient) SubscribeToEvents(_ context.Context, _ event.EventType, _ event.Handler) error {
	return nil
}

func (f *fakeSNClient) SubscribeToAllEvents(_ context.Context, _ event.Handler) error {
	return nil
}

func (f *fakeSNClient) GetSupernodeStatus(_ context.Context, _ string) (*pb.StatusResponse, error) {
	return nil, nil
}

func (f *fakeSNClient) DownloadCascade(_ context.Context, _ string, _ string, _ string) (string, error) {
	return "", nil
}

func (f *fakeSNClient) BuildCascadeMetadataFromFile(_ context.Context, filePath string, public bool, signerAddr string) (actiontypes.CascadeMetadata, string, string, error) {
	f.calls++
	f.lastFile = filePath
	f.lastPublic = public
	f.lastSigner = signerAddr
	if filePath == "" && f.errOnEmpty != nil {
		return actiontypes.CascadeMetadata{}, "", "", f.errOnEmpty
	}
	return f.meta, f.price, f.expiration, nil
}

func (f *fakeSNClient) GenerateStartCascadeSignatureFromFile(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (f *fakeSNClient) GenerateDownloadSignature(_ context.Context, _ string, _ string) (string, error) {
	return "", nil
}

func TestCreateRequestActionMessage(t *testing.T) {
	snClient := &fakeSNClient{
		meta: actiontypes.CascadeMetadata{
			DataHash:   "data-hash",
			FileName:   "file.txt",
			RqIdsIc:    1,
			RqIdsMax:   2,
			RqIdsIds:   []string{"rq-1"},
			Signatures: "sig",
			Public:     true,
		},
		price:      "10ulume",
		expiration: "12345",
		errOnEmpty: errors.New("file path is required"),
	}
	cascadeClient := &Client{snClient: snClient}

	_, _, err := cascadeClient.CreateRequestActionMessage(context.Background(), "", "", nil)
	require.Error(t, err)
	require.Zero(t, snClient.calls)

	_, _, err = cascadeClient.CreateRequestActionMessage(context.Background(), "creator", "", nil)
	require.Error(t, err)
	require.Equal(t, 1, snClient.calls)

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("data"), 0o600))

	_, _, err = cascadeClient.CreateRequestActionMessage(context.Background(), "", filePath, nil)
	require.Error(t, err)
	require.Equal(t, 1, snClient.calls)

	appPubkey := []byte{1, 2, 3}
	options := &UploadOptions{}
	WithICACreatorAddress("ica-creator")(options)
	WithAppPubkey(appPubkey)(options)
	WithPublic(true)(options)
	msg, meta, err := cascadeClient.CreateRequestActionMessage(context.Background(), "creator", filePath, options)
	require.NoError(t, err)
	require.Equal(t, "ica-creator", msg.Creator)
	require.Equal(t, appPubkey, msg.AppPubkey)
	require.Equal(t, snClient.price, msg.Price)
	require.Equal(t, snClient.expiration, msg.ExpirationTime)
	require.Equal(t, "1", msg.FileSizeKbs)

	expectedMeta, err := json.Marshal(snClient.meta)
	require.NoError(t, err)
	require.Equal(t, string(expectedMeta), msg.Metadata)
	require.Equal(t, msg.Metadata, string(meta))
	require.Equal(t, filePath, snClient.lastFile)
	require.True(t, snClient.lastPublic)
	require.Equal(t, "ica-creator", snClient.lastSigner)
}
