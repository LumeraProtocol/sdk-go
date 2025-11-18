package crypto

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdkbech32 "github.com/cosmos/cosmos-sdk/types/bech32"
)

// AddressFromKey derives an account address for the given HRP from the
// public key stored in the keyring under keyName, without mutating the
// global bech32 prefix configuration.
func AddressFromKey(kr keyring.Keyring, keyName, hrp string) (string, error) {
	if kr == nil {
		return "", fmt.Errorf("keyring is required")
	}
	if keyName == "" {
		return "", fmt.Errorf("key name is required")
	}
	rec, err := kr.Key(keyName)
	if err != nil {
		return "", fmt.Errorf("key %s not found: %w", keyName, err)
	}
	pub, err := rec.GetPubKey()
	if err != nil {
		return "", fmt.Errorf("get pubkey: %w", err)
	}
	if pub == nil {
		return "", fmt.Errorf("nil pubkey for key %s", keyName)
	}
	addrBz := pub.Address()
	bech, err := sdkbech32.ConvertAndEncode(hrp, addrBz)
	if err != nil {
		return "", fmt.Errorf("bech32 encode: %w", err)
	}
	return bech, nil
}
