package crypto

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cosmoscrypto "github.com/cosmos/cosmos-sdk/crypto/types"

	injethsecp256k1 "github.com/LumeraProtocol/sdk-go/pkg/crypto/ethsecp256k1"
)

func NewMultiChainKeyring(appName, backend, dir string) (keyring.Keyring, error) {
	// Expand ~ in path
	if strings.HasPrefix(dir, "~/") {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, dir[2:])
	}

	registry := codectypes.NewInterfaceRegistry()

	// Standard Cosmos
	registry.RegisterInterface("cosmos.crypto.PubKey", (*cosmoscrypto.PubKey)(nil))
	registry.RegisterInterface("cosmos.crypto.PrivKey", (*cosmoscrypto.PrivKey)(nil))
	registry.RegisterImplementations((*cosmoscrypto.PubKey)(nil),
		&secp256k1.PubKey{},
	)
	registry.RegisterImplementations((*cosmoscrypto.PrivKey)(nil),
		&secp256k1.PrivKey{},
	)

	// Injective
	registry.RegisterImplementations((*cosmoscrypto.PubKey)(nil),
		&injethsecp256k1.PubKey{},
	)
	registry.RegisterImplementations((*cosmoscrypto.PrivKey)(nil),
		&injethsecp256k1.PrivKey{},
	)

	cdc := codec.NewProtoCodec(registry)
	// For file backend, provide stdin for password input
	var userInput *bufio.Reader
	if backend == keyring.BackendFile {
		userInput = bufio.NewReader(os.Stdin)
	}

	return keyring.New(appName, backend, dir, userInput, cdc)
}
