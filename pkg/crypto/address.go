package crypto

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdkbech32 "github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/evm/crypto/ethsecp256k1"
	"github.com/ethereum/go-ethereum/common"
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

// EVMAddressFromKey derives the EIP-55 0x-prefixed hex address for the key
// stored in the keyring under keyName. The key must use the eth_secp256k1
// signing algorithm; for cosmos secp256k1 keys an error is returned.
func EVMAddressFromKey(kr keyring.Keyring, keyName string) (string, error) {
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
	if pub.Type() != ethsecp256k1.KeyType {
		return "", fmt.Errorf("key %s has algorithm %s; EVM address requires %s",
			keyName, pub.Type(), ethsecp256k1.KeyType)
	}
	return common.BytesToAddress(pub.Address()).Hex(), nil
}
