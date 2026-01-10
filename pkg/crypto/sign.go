package crypto

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
)

// SignTxWithKeyring signs the provided TxBuilder using the given keyring identity.
// The caller must supply chainID, account number and sequence.
// Set overwrite=false for the first signature; true for subsequent signers on the same tx.
func SignTxWithKeyring(
	ctx context.Context,
	txCfg client.TxConfig,
	kr keyring.Keyring,
	keyName string,
	builder client.TxBuilder,
	chainID string,
	accountNumber uint64,
	sequence uint64,
	overwrite bool,
) error {
	factory := tx.Factory{}.
		WithChainID(chainID).
		WithTxConfig(txCfg).
		WithAccountNumber(accountNumber).
		WithSequence(sequence).
		WithKeybase(kr)

	return tx.Sign(ctx, factory, keyName, builder, overwrite)
}