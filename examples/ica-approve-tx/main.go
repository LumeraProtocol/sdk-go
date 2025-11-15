package main

import (
    "context"
    "encoding/base64"
    "flag"
    "fmt"
    "os"
    "strings"

    "github.com/LumeraProtocol/sdk-go/cascade"
    codectypes "github.com/cosmos/cosmos-sdk/codec/types"
    gogoproto "github.com/cosmos/gogoproto/proto"
)

// This example builds an ICS-27 MsgSendTx that executes one or more
// Lumera MsgApproveAction messages over an Interchain Account.
// It performs no network calls; it only constructs and prints the tx bytes.
func main() {
    // Keyring-related flags (accepted but not used by this offline example)
    keyringBackend := flag.String("keyring-backend", "os", "Keyring backend: os|file|test")
    keyringDir := flag.String("keyring-dir", "~/.lumera", "Keyring base directory (actual dir appends keyring-<backend> for file/test)")
    keyName := flag.String("key-name", "my-key", "Key name in the keyring")

    owner := flag.String("owner", "inj1abc...", "Controller Address for MsgSendTx")
    connectionID := flag.String("connection-id", "connection-0", "IBC connection ID on controller chain")
    relTimeout := flag.Uint64("relative-timeout", 600_000_000_000, "Relative timeout nanoseconds for MsgSendTx (e.g. 10 min)")
    creator := flag.String("creator", "lumera1abc...", "Lumera ICA for ApproveAction")
    actionIDs := flag.String("action-ids", "", "Comma-separated list of action IDs to approve")
    _ = keyringBackend
    _ = keyringDir
    _ = keyName
    flag.Parse()

    ids := splitList(*actionIDs)
    if len(ids) == 0 {
        fmt.Println("--action-ids is required (comma-separated)")
        os.Exit(1)
    }

    var anys []*codectypes.Any
    for _, id := range ids {
        // Create the message using SDK package-level helper (no network calls)
        msg, err := cascade.CreateApproveActionMessage(context.Background(), id, cascade.WithApproveCreator(*creator))
        if err != nil {
            fmt.Printf("create approve message: %v\n", err)
            os.Exit(1)
        }

        anyBytes, err := cascade.PackApproveForICA(msg)
        if err != nil {
            fmt.Printf("pack approve for ICA: %v\n", err)
            os.Exit(1)
        }
        var any codectypes.Any
        if err := gogoproto.Unmarshal(anyBytes, &any); err != nil {
            fmt.Printf("unmarshal Any: %v\n", err)
            os.Exit(1)
        }
        anys = append(anys, &any)
    }

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

    out, err := gogoproto.Marshal(msgSendTx)
    if err != nil {
        fmt.Printf("marshal MsgSendTx: %v\n", err)
        os.Exit(1)
    }

    fmt.Println("Built ICS-27 MsgSendTx (approve actions)")
    fmt.Printf("Owner: %s\n", msgSendTx.Owner)
    fmt.Printf("Connection: %s\n", msgSendTx.ConnectionId)
    fmt.Printf("RelativeTimeout: %d\n", msgSendTx.RelativeTimeout)
    fmt.Printf("Included approvals: %d\n", len(anys))
    fmt.Println()
    fmt.Println("Base64(gogo_proto(MsgSendTx)):")
    fmt.Println(base64.StdEncoding.EncodeToString(out))
}

func splitList(s string) []string {
    s = strings.TrimSpace(s)
    if s == "" {
        return nil
    }
    parts := strings.Split(s, ",")
    out := make([]string, 0, len(parts))
    for _, p := range parts {
        p = strings.TrimSpace(p)
        if p != "" {
            out = append(out, p)
        }
    }
    return out
}
