// sdk-go/pkg/crypto/ethsecp256k1/ethsecp256k1.go
package ethsecp256k1

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	PrivKeySize = 32
	PubKeySize  = 33 // compressed
	KeyType     = "eth_secp256k1"

	// Proto type URLs - must match Injective's registration
	PubKeyName  = "injective.crypto.v1beta1.ethsecp256k1.PubKey"
	PrivKeyName = "injective.crypto.v1beta1.ethsecp256k1.PrivKey"
)

var (
	_ cryptotypes.PrivKey = &PrivKey{}
	_ cryptotypes.PubKey  = &PubKey{}
)

// PrivKey defines a secp256k1 private key using Ethereum's Keccak256 hashing
type PrivKey struct {
	Key []byte `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
}

func (privKey *PrivKey) Bytes() []byte {
	if privKey == nil {
		return nil
	}
	return privKey.Key
}

func (privKey *PrivKey) PubKey() cryptotypes.PubKey {
	ecdsaPrivKey, err := crypto.ToECDSA(privKey.Key)
	if err != nil {
		return nil
	}
	return &PubKey{Key: crypto.CompressPubkey(&ecdsaPrivKey.PublicKey)}
}

func (privKey *PrivKey) Equals(other cryptotypes.LedgerPrivKey) bool {
	return privKey.Type() == other.Type() && subtle.ConstantTimeCompare(privKey.Bytes(), other.Bytes()) == 1
}

func (privKey *PrivKey) Type() string {
	return KeyType
}

func (privKey *PrivKey) Sign(digestBz []byte) ([]byte, error) {
	if len(digestBz) != crypto.DigestLength {
		// Use SHA256 like standard Cosmos, NOT Keccak256
		hash := sha256.Sum256(digestBz)
		digestBz = hash[:]
	}
	key, err := crypto.ToECDSA(privKey.Key)
	if err != nil {
		return nil, err
	}

	sig, err := crypto.Sign(digestBz, key)
	if err != nil {
		return nil, err
	}

	// Remove recovery ID (65 → 64 bytes) to match Cosmos format
	if len(sig) == 65 {
		sig = sig[:64]
	}

	return sig, nil
}

// Proto methods
func (privKey *PrivKey) Reset()         { *privKey = PrivKey{} }
func (privKey *PrivKey) String() string { return fmt.Sprintf("eth_secp256k1{%X}", privKey.Key) }
func (privKey *PrivKey) ProtoMessage()  {}

// XXX_MessageName returns the proto message name for proper type URL registration
func (*PrivKey) XXX_MessageName() string { return PrivKeyName }

// PubKey defines a secp256k1 public key using Ethereum's format
type PubKey struct {
	Key []byte `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
}

func (pubKey *PubKey) Address() cryptotypes.Address {
	if len(pubKey.Key) != PubKeySize {
		return nil
	}
	uncompressed, err := crypto.DecompressPubkey(pubKey.Key)
	if err != nil {
		return nil
	}
	return crypto.PubkeyToAddress(*uncompressed).Bytes()
}

func (pubKey *PubKey) Bytes() []byte {
	if pubKey == nil {
		return nil
	}
	return pubKey.Key
}

func (pubKey *PubKey) Equals(other cryptotypes.PubKey) bool {
	return pubKey.Type() == other.Type() && bytes.Equal(pubKey.Bytes(), other.Bytes())
}

func (pubKey *PubKey) Type() string {
	return KeyType
}

func (pubKey *PubKey) VerifySignature(msg, sig []byte) bool {
	if len(sig) == crypto.SignatureLength {
		sig = sig[:crypto.SignatureLength-1] // remove recovery ID
	}
	hash := crypto.Keccak256(msg)
	return crypto.VerifySignature(pubKey.Key, hash, sig)
}

// Proto methods
func (pubKey *PubKey) Reset()         { *pubKey = PubKey{} }
func (pubKey *PubKey) String() string { return fmt.Sprintf("eth_secp256k1{%X}", pubKey.Key) }
func (pubKey *PubKey) ProtoMessage()  {}

// XXX_MessageName returns the proto message name for proper type URL registration
func (*PubKey) XXX_MessageName() string { return PubKeyName }

// RegisterInterfaces registers the ethsecp256k1 types with Injective's type URLs
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*cryptotypes.PubKey)(nil), &PubKey{})
	registry.RegisterImplementations((*cryptotypes.PrivKey)(nil), &PrivKey{})
}
