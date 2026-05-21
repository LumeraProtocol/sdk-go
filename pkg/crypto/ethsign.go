package crypto

import (
	"fmt"
	"math/big"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/evm/crypto/ethsecp256k1"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// SignEthereumTx signs a go-ethereum *types.Transaction with the named keyring
// entry (which must be an eth_secp256k1 key) using the latest replay-protected
// signer for chainID. Returns the signed transaction.
func SignEthereumTx(
	kr keyring.Keyring,
	keyName string,
	chainID *big.Int,
	tx *ethtypes.Transaction,
) (*ethtypes.Transaction, error) {
	if kr == nil {
		return nil, fmt.Errorf("keyring is required")
	}
	if keyName == "" {
		return nil, fmt.Errorf("key name is required")
	}
	if chainID == nil || chainID.Sign() <= 0 {
		return nil, fmt.Errorf("chainID must be positive")
	}
	if tx == nil {
		return nil, fmt.Errorf("tx is required")
	}

	rec, err := kr.Key(keyName)
	if err != nil {
		return nil, fmt.Errorf("load key %q: %w", keyName, err)
	}
	pub, err := rec.GetPubKey()
	if err != nil {
		return nil, fmt.Errorf("get pubkey: %w", err)
	}
	if pub == nil || pub.Type() != ethsecp256k1.KeyType {
		return nil, fmt.Errorf("key %q has algorithm %q; EVM signing requires %q",
			keyName, typeOf(pub), ethsecp256k1.KeyType)
	}

	signer := ethtypes.LatestSignerForChainID(chainID)
	hash := signer.Hash(tx).Bytes()

	sig, _, err := kr.Sign(keyName, hash, signingtypes.SignMode_SIGN_MODE_DIRECT)
	if err != nil {
		return nil, fmt.Errorf("keyring sign: %w", err)
	}
	if len(sig) != 65 {
		return nil, fmt.Errorf("expected 65-byte recoverable signature, got %d", len(sig))
	}

	signed, err := tx.WithSignature(signer, sig)
	if err != nil {
		return nil, fmt.Errorf("apply signature: %w", err)
	}
	return signed, nil
}

// WrapAsMsgEthereumTx packages a signed go-ethereum tx as a cosmos
// MsgEthereumTx ready for MsgEthereumTx.BuildTx and broadcast.
func WrapAsMsgEthereumTx(signed *ethtypes.Transaction) (*evmtypes.MsgEthereumTx, error) {
	if signed == nil {
		return nil, fmt.Errorf("signed tx is required")
	}
	chainID := signed.ChainId()
	if chainID == nil || chainID.Sign() <= 0 {
		return nil, fmt.Errorf("signed tx has no chain id; refusing to wrap a non-EIP-155 tx")
	}

	msg := &evmtypes.MsgEthereumTx{}
	signer := ethtypes.LatestSignerForChainID(chainID)
	if err := msg.FromSignedEthereumTx(signed, signer); err != nil {
		return nil, fmt.Errorf("populate MsgEthereumTx: %w", err)
	}
	return msg, nil
}

// RecoverSender returns the EVM address recovered from the signature of a
// signed go-ethereum tx. Useful for verifying that the keyring key produced
// the expected sender.
func RecoverSender(signed *ethtypes.Transaction) (common.Address, error) {
	if signed == nil {
		return common.Address{}, fmt.Errorf("signed tx is required")
	}
	chainID := signed.ChainId()
	if chainID == nil || chainID.Sign() <= 0 {
		return common.Address{}, fmt.Errorf("signed tx has no chain id")
	}
	signer := ethtypes.LatestSignerForChainID(chainID)
	return signer.Sender(signed)
}

func typeOf(pub interface{ Type() string }) string {
	if pub == nil {
		return "<nil>"
	}
	return pub.Type()
}
