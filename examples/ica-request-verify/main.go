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
	"sort"
	"strings"

	"encoding/asn1"
	"encoding/base64"
	"math/big"

	"github.com/LumeraProtocol/sdk-go/cascade"
	"github.com/LumeraProtocol/sdk-go/constants"
	"github.com/LumeraProtocol/sdk-go/ica"
	sdkcrypto "github.com/LumeraProtocol/sdk-go/pkg/crypto"
	"github.com/LumeraProtocol/sdk-go/pkg/crypto/ethsecp256k1"
	sdktypes "github.com/LumeraProtocol/sdk-go/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
)

// This example builds an ICS-27 MsgSendTx that executes one or more
// Lumera MsgRequestAction messages over an Interchain Account.
// It builds Cascade metadata via the sdk-go Cascade client (chain gRPC)
// but does not broadcast any transactions.
func main() {
	// Keyring-only; addresses are derived from the key
	keyringBackend := flag.String("keyring-backend", "os", "Keyring backend: os|file|test")
	keyringDir := flag.String("keyring-dir", "~/.lumera", "Keyring base directory (actual dir appends keyring-<backend> for file/test)")
	keyName := flag.String("key-name", "my-key", "Key name in the keyring")
	ownerHRP := flag.String("owner-hrp", "inj", "Bech32 HRP for controller chain owner address (e.g., inj)")
	icaAddress := flag.String("ica-address", "", "ICA address on Lumera (host chain)")
	grpcAddr := flag.String("grpc-addr", "", "Lumera gRPC address (host:port)")
	chainID := flag.String("chain-id", "", "Lumera chain ID")
	keyringType := flag.String("keyring-type", "lumera", "Keyring type: lumera|injective")

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
	if strings.TrimSpace(*grpcAddr) == "" {
		fmt.Println("--grpc-addr is required")
		os.Exit(1)
	}
	if strings.TrimSpace(*chainID) == "" {
		fmt.Println("--chain-id is required")
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

	// Choose app name based on keyring type
	appName := "lumera"
	if *keyringType == "injective" {
		appName = "injectived"
	}
	fmt.Printf("Using keyring app name: %s\n", appName)

	kr, err := sdkcrypto.NewMultiChainKeyring(appName, *keyringBackend, *keyringDir)
	if err != nil {
		fmt.Printf("open keyring: %v\n", err)
		os.Exit(1)
	}

	lumeraAddress, err := sdkcrypto.AddressFromKey(kr, *keyName, constants.LumeraAccountHRP)
	if err != nil {
		log.Fatalf("derive owner address: %v\n", err)
	}
	fmt.Printf("Derived Lumera address for key '%s': %s\n", *keyName, lumeraAddress)

	// Derive controller owner address from the same key with the given HRP
	ownerAddr, err := sdkcrypto.AddressFromKey(kr, *keyName, *ownerHRP)
	if err != nil {
		fmt.Printf("derive owner address: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Derived controller owner address for key '%s' with HRP '%s': %s\n", *keyName, *ownerHRP, ownerAddr)

	ctx := context.Background()
	cascadeClient, err := cascade.New(ctx, cascade.Config{
		ChainID:         *chainID,
		GRPCAddr:        *grpcAddr,
		Address:         lumeraAddress,
		KeyName:         *keyName,
		ICAOwnerKeyName: *keyName,
		ICAOwnerHRP:     *ownerHRP,
	}, kr)
	if err != nil {
		fmt.Printf("create cascade client: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = cascadeClient.Close()
	}()

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
		fmt.Printf("Using ICA with address %s and app pubkey %X\n", *icaAddress, appPubkey)

		///////////////////////////////////////////////////////
		//test ethsecp256k1
		addr, err := rec.GetAddress()
		if err != nil {
			fmt.Printf("get address: %v\n", err)
			os.Exit(1)
		}
		st := "Test string"
		b := []byte(st)
		sig, _, err := kr.SignByAddress(addr, b, signing.SignMode_SIGN_MODE_DIRECT)
		if err != nil {
			fmt.Printf("sign: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Signature: %X\n", sig)

		appPk := ethsecp256k1.PubKey{Key: appPubkey}
		pubKey := &appPk
		valid := pubKey.VerifySignature(b, sig)
		fmt.Printf("PubKey type - %s, Key Name - %s\n", appPk.Type(), rec.Name)
		fmt.Printf("Verify Signature: %v\n", valid)
		///////////////////////////////////////////////////////
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
		uploadOpts := &cascade.UploadOptions{}
		for _, opt := range opts {
			opt(uploadOpts)
		}
		msgSendTx, _, err := cascadeClient.CreateRequestActionMessage(ctx, lumeraAddress, f, uploadOpts)
		if err != nil {
			fmt.Printf("create request message: %v\n", err)
			os.Exit(1)
		}
		jsonBytes, err := json.MarshalIndent(msgSendTx, "", "  ")
		if err != nil {
			fmt.Printf("marshal MsgSendTx to JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("JSON(MsgSendTx):")
		fmt.Println(string(jsonBytes))

		// --- Signature Verification ---
		fmt.Printf("Verifying signature for %s...\n", f)

		// Assume secp256k1-formatted pubkey bytes for app-level signatures.
		appPk := secp256k1.PubKey{Key: appPubkey}
		pubKey := &appPk
		fmt.Printf("Using app pubkey: %s\n", pubKey.String())

		// 1. Unmarshal Metadata
		var meta sdktypes.CascadeMetadata
		// MsgRequestAction.Metadata is a JSON string of CascadeMetadata
		if err := json.Unmarshal([]byte(msgSendTx.Metadata), &meta); err != nil {
			fmt.Printf("failed to unmarshal metadata: %v\n", err)
			os.Exit(1)
		}

		// 2. Parse Signatures field: "Base64(rq_ids).Base64(signature)"
		parts := strings.Split(meta.Signatures, ".")
		if len(parts) != 2 {
			fmt.Printf("invalid signatures format: expected 'data.signature', got '%s'\n", meta.Signatures)
			os.Exit(1)
		}
		dataB64 := parts[0]
		sigB64 := parts[1]
		fmt.Printf("  Data (Base64): %s\n", dataB64)
		fmt.Printf("  Signature (Base64): %s\n", sigB64)

		// 3. Decode the base64 signature
		sigRaw, err := base64.StdEncoding.DecodeString(sigB64)
		if err != nil {
			fmt.Printf("failed to decode signature base64: %v\n", err)
			os.Exit(1)
		}

		// 4. Coerce to r||s format
		sigRS := CoerceToRS64(sigRaw)

		// 5. Verify the signature
		valid := pubKey.VerifySignature([]byte(dataB64), sigRS)
		if !valid {
			fmt.Printf("Signature verification FAILED for %s\n", f)

			// 6. ADR-36 (Keplr/browser)
			signBytes, err := MakeADR36AminoSignBytes(lumeraAddress, dataB64)
			if err == nil && pubKey.VerifySignature(signBytes, sigRS) {
				fmt.Printf("ADR-36 Signature Verified Successfully for %s\n", f)
			} else {
				fmt.Printf("ADR-36 signature verification FAILED for %s\n", f)
				os.Exit(1)
			}
		} else {
			fmt.Printf("Signature Verified Successfully for %s\n", f)
		}
		// ------------------------------

		// Pack to Any for ICA execution
		any, err := ica.PackRequestAny(msgSendTx)
		if err != nil {
			fmt.Printf("pack request Any: %v\n", err)
			os.Exit(1)
		}
		anys = append(anys, any)
	}

	// Build packet and MsgSendTx
	packet, err := ica.BuildICAPacketData(anys)
	if err != nil {
		fmt.Printf("build packet: %v\n", err)
		os.Exit(1)
	}

	msgSendTx, err := ica.BuildMsgSendTx(ownerAddr, *connectionID, *relTimeout, packet)
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

// CoerceToRS64 converts an ECDSA signature to 64-byte [R||S] format.
// If it's already 64 bytes, it returns it as is.
// If it's ASN.1, it parses and converts.
func CoerceToRS64(sig []byte) []byte {
	if len(sig) == 64 {
		return sig
	}
	// Try parsing as ASN.1
	var ecdsaSig struct {
		R, S *big.Int
	}
	if _, err := asn1.Unmarshal(sig, &ecdsaSig); err != nil {
		// Not ASN.1 or invalid, return original
		return sig
	}

	// Normalize to 32 bytes each
	rBytes := ecdsaSig.R.Bytes()
	sBytes := ecdsaSig.S.Bytes()

	r32 := make([]byte, 32)
	s32 := make([]byte, 32)

	// Copy into the end of the buffer (big endian)
	if len(rBytes) > 32 {
		// Should not happen for secp256k1 unless padded with zero?
		// ecdsaSig.R.Bytes() strips leading zeros, but if it was somehow larger...
		// Just take last 32? No, error out or take strict?
		// For robustness we just copy what fits or all if small
		copy(r32[:], rBytes[len(rBytes)-32:])
	} else {
		copy(r32[32-len(rBytes):], rBytes)
	}

	if len(sBytes) > 32 {
		copy(s32[:], sBytes[len(sBytes)-32:])
	} else {
		copy(s32[32-len(sBytes):], sBytes)
	}

	return append(r32, s32...)
}

// MakeADR36AminoSignBytes returns the exact JSON bytes Keplr signs.
// signerBech32: bech32 address; dataB64: base64 STRING that was given to Keplr signArbitrary().
func MakeADR36AminoSignBytes(signerBech32, dataB64 string) ([]byte, error) {
	doc := map[string]any{
		"account_number": "0",
		"chain_id":       "",
		"fee": map[string]any{
			"amount": []any{},
			"gas":    "0",
		},
		"memo": "",
		"msgs": []any{
			map[string]any{
				"type": "sign/MsgSignData",
				"value": map[string]any{
					"data":   dataB64,      // IMPORTANT: base64 STRING (do not decode)
					"signer": signerBech32, // bech32 account address
				},
			},
		},
		"sequence": "0",
	}

	canon := sortObjectByKey(doc)
	bz, err := json.Marshal(canon)
	if err != nil {
		return nil, fmt.Errorf("marshal adr36 doc: %w", err)
	}
	return bz, nil
}

func sortObjectByKey(v any) any {
	switch x := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make(map[string]any, len(x))
		for _, k := range keys {
			out[k] = sortObjectByKey(x[k])
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = sortObjectByKey(x[i])
		}
		return out
	default:
		return v
	}
}
