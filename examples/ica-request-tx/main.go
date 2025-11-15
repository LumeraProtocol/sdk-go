package main

import (
    "context"
    "encoding/base64"
    "flag"
    "fmt"
    "io/fs"
    "os"
    "path/filepath"
    "strings"

    "github.com/LumeraProtocol/sdk-go/cascade"
    codectypes "github.com/cosmos/cosmos-sdk/codec/types"
    gogoproto "github.com/cosmos/gogoproto/proto"
    controllertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
)

// This example builds an ICS-27 MsgSendTx that executes one or more
// Lumera MsgRequestAction messages over an Interchain Account.
// It performs no network calls; it only constructs and prints the tx bytes.
func main() {
    // Keyring-related flags (accepted but not used for network calls in this offline example)
    keyringBackend := flag.String("keyring-backend", "os", "Keyring backend: os|file|test")
    keyringDir := flag.String("keyring-dir", "~/.lumera", "Keyring base directory (actual dir appends keyring-<backend> for file/test)")
    keyName := flag.String("key-name", "my-key", "Key name in the keyring")

    // Addresses and IBC params
    creator := flag.String("creator", "lumera1abc...", "Lumera ICA for RequestAction")
    owner := flag.String("owner", "inj1abc...", "Controller Address for MsgSendTx")
    connectionID := flag.String("connection-id", "connection-0", "IBC connection ID on controller chain")
    relTimeout := flag.Uint64("relative-timeout", 600_000_000_000, "Relative timeout nanoseconds for MsgSendTx (e.g. 10 min)")

    // Input path
    path := flag.String("path", "", "Path to a single file or a directory containing files")
    _ = keyringBackend
    _ = keyringDir
    _ = keyName
    flag.Parse()

    if strings.TrimSpace(*path) == "" {
        fmt.Println("--path is required")
        os.Exit(1)
    }

    files, err := collectFiles(*path)
    if err != nil {
        fmt.Printf("collect files: %v\n", err)
        os.Exit(1)
    }
    if len(files) == 0 {
        fmt.Println("no files found to build messages")
        os.Exit(1)
    }

    // Build one MsgRequestAction per file using SDK helper
    var anys []*codectypes.Any
    for _, f := range files {
        // Create the message using SDK package-level helper (no network calls)
        msg, _, err := cascade.CreateRequestActionMessage(context.Background(), *creator, f)
        if err != nil {
            fmt.Printf("create request message: %v\n", err)
            os.Exit(1)
        }

        // Pack to Any bytes and unmarshal back to Any struct
        anyBytes, err := cascade.PackRequestForICA(msg)
        if err != nil {
            fmt.Printf("pack request for ICA: %v\n", err)
            os.Exit(1)
        }
        var any codectypes.Any
        if err := gogoproto.Unmarshal(anyBytes, &any); err != nil {
            fmt.Printf("unmarshal Any: %v\n", err)
            os.Exit(1)
        }
        anys = append(anys, &any)
    }

    // Build packet and MsgSendTx
    packet, err := cascade.BuildICAPacketData(anys)
    if err != nil {
        fmt.Printf("build packet: %v\n", err)
        os.Exit(1)
    }
    msgSendTx, err := cascade.BuildMsgSendTx(*owner, *connectionID, *relTimeout, packet)
    if err != nil {
        fmt.Printf("build MsgSendTx: %v\n", err)
        os.Exit(1)
    }

    // Marshal final message to bytes and print a summary
    out, err := gogoproto.Marshal(msgSendTx)
    if err != nil {
        fmt.Printf("marshal MsgSendTx: %v\n", err)
        os.Exit(1)
    }

    fmt.Println("Built ICS-27 MsgSendTx (controller message)")
    fmt.Printf("Owner: %s\n", msgSendTx.Owner)
    fmt.Printf("Connection: %s\n", msgSendTx.ConnectionId)
    fmt.Printf("RelativeTimeout: %d\n", msgSendTx.RelativeTimeout)
    fmt.Printf("Included messages: %d\n", len(anys))
    fmt.Println()
    fmt.Println("Base64(gogo_proto(MsgSendTx)):")
    fmt.Println(base64.StdEncoding.EncodeToString(out))

    _ = controllertypes.MsgSendTx{} // ensure import is retained
}

func collectFiles(p string) ([]string, error) {
    st, err := os.Stat(p)
    if err != nil {
        return nil, err
    }
    if !st.IsDir() {
        // Single file
        return []string{p}, nil
    }
    var out []string
    dirEntries, err := os.ReadDir(p)
    if err != nil {
        return nil, err
    }
    for _, de := range dirEntries {
        if de.IsDir() {
            continue // non-recursive
        }
        // Ensure it's a regular file
        info, err := de.Info()
        if err != nil {
            continue
        }
        if (info.Mode() & fs.ModeType) == 0 {
            out = append(out, filepath.Join(p, de.Name()))
        }
    }
    return out, nil
}

// simpleMetadataForFile creates a tiny JSON with filename and size; this is
// sufficient for demonstrating message construction without external calls.
// simple utility retained for potential future debugging; not used by the example directly
func simpleMetadataForFile(path string) string {
    fi, err := os.Stat(path)
    if err != nil {
        return fmt.Sprintf(`{"file":"%s"}`, filepath.Base(path))
    }
    return fmt.Sprintf(`{"file":"%s","size":%d}`, filepath.Base(path), fi.Size())
}
