package crypto

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdkbech32 "github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/common"
)

// uLumeToALumeFactor is the scaling factor between ulume (6 decimals) and
// alume (18 decimals), defined by x/precisebank as 10^12.
const uLumeToALumeFactor = 1_000_000_000_000

// uLumeToALumeBig is the multiplier used by Wei / ULume / ULumeDecToWei.
var uLumeToALumeBig = new(big.Int).SetUint64(uLumeToALumeFactor)

// EVMToBech32 converts a 20-byte EVM address to the bech32 form used by the
// Cosmos bank/auth modules for the same account.
func EVMToBech32(addr common.Address, hrp string) (string, error) {
	if hrp == "" {
		return "", fmt.Errorf("hrp is required")
	}
	bech, err := sdkbech32.ConvertAndEncode(hrp, addr.Bytes())
	if err != nil {
		return "", fmt.Errorf("bech32 encode: %w", err)
	}
	return bech, nil
}

// Bech32ToEVM decodes a bech32 account address and reinterprets those 20 bytes
// as an EVM address. This is only a byte-level conversion: for legacy Cosmos
// secp256k1 accounts, the result is not the key's Ethereum address. Callers
// that care must validate against a keyring entry, on-chain account metadata,
// or an evmigration record.
func Bech32ToEVM(bech32Addr string) (common.Address, error) {
	if bech32Addr == "" {
		return common.Address{}, fmt.Errorf("bech32 address is required")
	}
	_, raw, err := sdkbech32.DecodeAndConvert(bech32Addr)
	if err != nil {
		return common.Address{}, fmt.Errorf("bech32 decode: %w", err)
	}
	if len(raw) != common.AddressLength {
		return common.Address{}, fmt.Errorf("bech32 address has %d bytes, want %d", len(raw), common.AddressLength)
	}
	return common.BytesToAddress(raw), nil
}

// Wei converts an integer ulume amount to its 18-decimal alume (wei-like) form
// using the 10^12 conversion factor defined by x/precisebank.
func Wei(ulume sdkmath.Int) *big.Int {
	if ulume.IsNil() {
		return new(big.Int)
	}
	return new(big.Int).Mul(ulume.BigInt(), uLumeToALumeBig)
}

// ULumeDecToWei converts a decimal ulume value (such as feemarket BaseFee or
// MinGasPrice in ulume/gas) into the integer alume/gas representation used by
// EIP-1559 fee fields. It errors if the conversion would lose precision
// (i.e. the value has more than 12 decimal places).
func ULumeDecToWei(ulume sdkmath.LegacyDec) (*big.Int, error) {
	if ulume.IsNil() {
		return nil, fmt.Errorf("nil decimal value")
	}
	if ulume.IsNegative() {
		return nil, fmt.Errorf("negative value %s", ulume)
	}
	scaled := ulume.MulInt64(uLumeToALumeFactor)
	if !scaled.TruncateDec().Equal(scaled) {
		return nil, fmt.Errorf("value %s has sub-alume precision", ulume)
	}
	return scaled.TruncateInt().BigInt(), nil
}

// ULume converts an alume amount to ulume, truncating any sub-ulume fractional
// component. The dropped fractional remainder is returned so callers can
// surface precision loss.
func ULume(alume *big.Int) (ulume sdkmath.Int, fractional *big.Int) {
	if alume == nil {
		return sdkmath.ZeroInt(), new(big.Int)
	}
	q, r := new(big.Int).QuoRem(alume, uLumeToALumeBig, new(big.Int))
	return sdkmath.NewIntFromBigInt(q), r
}
