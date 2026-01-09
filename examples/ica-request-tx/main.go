package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/LumeraProtocol/sdk-go/cascade"
	sdkcrypto "github.com/LumeraProtocol/sdk-go/pkg/crypto"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	controllertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
)

// This example builds an ICS-27 MsgSendTx that executes one or more
// Lumera MsgRequestAction messages over an Interchain Account.
// It performs no network calls; it only constructs and prints the tx bytes.
func main() {
	// Keyring-only; addresses are derived from the key
	keyringBackend := flag.String("keyring-backend", "os", "Keyring backend: os|file|test")
	keyringDir := flag.String("keyring-dir", "~/.lumera", "Keyring base directory (actual dir appends keyring-<backend> for file/test)")
	keyName := flag.String("key-name", "my-key", "Key name in the keyring")
	ownerHRP := flag.String("owner-hrp", "inj", "Bech32 HRP for controller chain owner address (e.g., inj)")
	icaAddress := flag.String("ica-address", "", "ICA address on Lumera (host chain)")

	// IBC params
	connectionID := flag.String("connection-id", "connection-0", "IBC connection ID on controller chain")
	relTimeout := flag.Uint64("relative-timeout", 600_000_000_000, "Relative timeout nanoseconds for MsgSendTx (e.g. 10 min)")

	// Input path
	path := flag.String("path", "", "Path to a single file or a directory containing files")
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

	params := sdkcrypto.KeyringParams{AppName: "lumera", Backend: *keyringBackend, Dir: *keyringDir}
	kr, err := sdkcrypto.NewKeyring(params)
	if err != nil {
		fmt.Printf("open keyring: %v\n", err)
		os.Exit(1)
	}

	lumeraAddress, err := sdkcrypto.AddressFromKey(kr, *keyName, "lumera")
	if err != nil {
		log.Fatalf("derive owner address: %v\n", err)
	}

	// Derive controller owner address from the same key with the given HRP
	ownerAddr, err := sdkcrypto.AddressFromKey(kr, *keyName, *ownerHRP)
	if err != nil {
		fmt.Printf("derive owner address: %v\n", err)
		os.Exit(1)
	}

	useICA := strings.TrimSpace(*icaAddress) != ""
	var appPubkey []byte
	if useICA {
		rec, err := kr.Key(*keyName)
		if err != nil {
			fmt.Printf("load key: %v\n", err)
			os.Exit(1)
		}
		pub, err := rec.GetPubKey()
		if err != nil {
			fmt.Printf("get pubkey: %v\n", err)
			os.Exit(1)
		}
		if pub == nil {
			fmt.Println("nil pubkey for key")
			os.Exit(1)
		}
		appPubkey = pub.Bytes()
	}

	// Build one MsgRequestAction per file
	var anys []*codectypes.Any
	for _, f := range files {
		var opts []cascade.UploadOption
		if useICA {
			opts = append(opts,
				cascade.WithICACreatorAddress(*icaAddress),
				cascade.WithAppPubkey(appPubkey),
			)
		}
		msg, _, err := cascade.CreateRequestActionMessage(context.Background(), lumeraAddress, f, opts...)
		if err != nil {
			fmt.Printf("create request message: %v\n", err)
			os.Exit(1)
		}

		// Pack to Any for ICA execution
		any, err := cascade.PackRequestAny(msg)
		if err != nil {
			fmt.Printf("pack request Any: %v\n", err)
			os.Exit(1)
		}
		anys = append(anys, any)
	}

	// Build packet and MsgSendTx
	packet, err := cascade.BuildICAPacketData(anys)
	if err != nil {
		fmt.Printf("build packet: %v\n", err)
		os.Exit(1)
	}

	msgSendTx, err := cascade.BuildMsgSendTx(ownerAddr, *connectionID, *relTimeout, packet)
	if err != nil {
		fmt.Printf("build MsgSendTx: %v\n", err)
		os.Exit(1)
	}

	// Convert to JSON
	jsonBytes, err := json.MarshalIndent(msgSendTx, "", "  ")
	if err != nil {
		fmt.Printf("marshal MsgSendTx to JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Built ICS-27 MsgSendTx (controller message)")
	fmt.Printf("Owner (controller): %s\n", msgSendTx.Owner)
	fmt.Printf("Connection: %s\n", msgSendTx.ConnectionId)
	fmt.Printf("RelativeTimeout: %d\n", msgSendTx.RelativeTimeout)
	fmt.Printf("Included messages: %d\n", len(anys))
	fmt.Println()
	fmt.Println("JSON(MsgSendTx):")
	fmt.Println(string(jsonBytes))

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
