package crypto

import (
	"math/big"
	"strings"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/LumeraProtocol/sdk-go/constants"
	sdkbech32 "github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestEVMToBech32_RoundTrip(t *testing.T) {
	original := common.HexToAddress("0xAbC0123456789abcdef0123456789aBcDef012345")
	bech, err := EVMToBech32(original, constants.LumeraAccountHRP)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(bech, constants.LumeraAccountHRP+"1"), "bech32: %s", bech)

	decoded, err := Bech32ToEVM(bech)
	require.NoError(t, err)
	require.Equal(t, original, decoded)
}

func TestEVMToBech32_RequiresHRP(t *testing.T) {
	_, err := EVMToBech32(common.Address{}, "")
	require.Error(t, err)
}

func TestBech32ToEVM_RejectsEmpty(t *testing.T) {
	_, err := Bech32ToEVM("")
	require.Error(t, err)
}

func TestBech32ToEVM_RejectsMalformed(t *testing.T) {
	_, err := Bech32ToEVM("not-a-bech32-string")
	require.Error(t, err)
}

func TestBech32ToEVM_RejectsWrongLength(t *testing.T) {
	// Encode a 32-byte payload (e.g. a consensus pubkey), which is not a
	// valid EVM account address.
	thirtyTwo := make([]byte, 32)
	for i := range thirtyTwo {
		thirtyTwo[i] = byte(i + 1)
	}
	bech, err := sdkbech32.ConvertAndEncode(constants.LumeraAccountHRP, thirtyTwo)
	require.NoError(t, err)

	_, err = Bech32ToEVM(bech)
	require.Error(t, err)
	require.Contains(t, err.Error(), "want 20")
}

func TestWeiAndULume_RoundTrip(t *testing.T) {
	ulume := sdkmath.NewInt(123_456_789)
	wei := Wei(ulume)
	require.Equal(t, new(big.Int).Mul(big.NewInt(123_456_789), big.NewInt(1_000_000_000_000)), wei)

	back, fractional := ULume(wei)
	require.True(t, ulume.Equal(back), "got %s want %s", back, ulume)
	require.Equal(t, int64(0), fractional.Int64())
}

func TestWei_NilInt(t *testing.T) {
	require.Equal(t, big.NewInt(0), Wei(sdkmath.Int{}))
}

func TestULume_FractionalRemainder(t *testing.T) {
	// 1 ulume + 1 = 10^12 + 1 alume.
	alume := new(big.Int).Add(big.NewInt(1_000_000_000_000), big.NewInt(1))
	ulume, frac := ULume(alume)
	require.True(t, ulume.Equal(sdkmath.NewInt(1)))
	require.Equal(t, int64(1), frac.Int64())
}

func TestULume_NilInput(t *testing.T) {
	ulume, frac := ULume(nil)
	require.True(t, ulume.IsZero())
	require.Equal(t, int64(0), frac.Int64())
}

func TestULumeDecToWei(t *testing.T) {
	// 0.0025 ulume/gas = 0.0025 * 10^12 alume/gas = 2,500,000,000 alume/gas
	dec := sdkmath.LegacyMustNewDecFromStr("0.0025")
	got, err := ULumeDecToWei(dec)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(2_500_000_000), got)

	// MinGasPrice 0.0005 -> 500,000,000.
	got2, err := ULumeDecToWei(sdkmath.LegacyMustNewDecFromStr("0.0005"))
	require.NoError(t, err)
	require.Equal(t, big.NewInt(500_000_000), got2)
}

func TestULumeDecToWei_Negative(t *testing.T) {
	_, err := ULumeDecToWei(sdkmath.LegacyMustNewDecFromStr("-1"))
	require.Error(t, err)
}

func TestULumeDecToWei_SubAlumePrecision(t *testing.T) {
	// 18 decimal places exceeds the 12-decimal conversion precision.
	dec := sdkmath.LegacyNewDecWithPrec(1, 18)
	_, err := ULumeDecToWei(dec)
	require.Error(t, err)
}

func TestULumeDecToWei_Nil(t *testing.T) {
	_, err := ULumeDecToWei(sdkmath.LegacyDec{})
	require.Error(t, err)
}
