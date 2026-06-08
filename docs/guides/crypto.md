# Crypto Helpers (`pkg/crypto`)

The `pkg/crypto` package provides keyring creation, key import, and address derivation. A single keyring supports both Cosmos (`secp256k1`) and EVM (`eth_secp256k1`) key types.

## Key types

`KeyType` selects the cryptographic algorithm and BIP44 derivation path:

| KeyType         | Algorithm        | BIP44 Coin Type | HD Path              |
| --------------- | ---------------- | --------------- | -------------------- |
| `KeyTypeCosmos` | `secp256k1`      | 118             | `m/44'/118'/0'/0/0`  |
| `KeyTypeEVM`    | `eth_secp256k1`  | 60              | `m/44'/60'/0'/0/0`   |

## Creating a keyring

`NewKeyring` creates a keyring that accepts both key types. The algorithm is selected when importing or creating keys, not at keyring creation time.

```go
import sdkcrypto "github.com/LumeraProtocol/sdk-go/pkg/crypto"

kr, err := sdkcrypto.NewKeyring(sdkcrypto.DefaultKeyringParams())
```

## Importing keys from a mnemonic

Use `LoadKeyring` to create a test keyring and import a key in one step:

```go
kr, pubBytes, addr, err := sdkcrypto.LoadKeyring("alice", "mnemonic.txt", sdkcrypto.KeyTypeCosmos)
```

Use `ImportKey` to add a key to an existing keyring:

```go
pubBytes, addr, err := sdkcrypto.ImportKey(kr, "bob", "mnemonic.txt", "lumera", sdkcrypto.KeyTypeCosmos)
```

`ImportKey` is idempotent: re-importing the same name with the same mnemonic and key type returns the existing key. It errors if the name already exists with a different key type **or** a different mnemonic, so a typo cannot silently return the wrong account.

## Mixing key types in the same keyring

When controller and host chains use different cryptographic key types, import keys under separate names:

```go
kr, _ := sdkcrypto.NewKeyring(sdkcrypto.DefaultKeyringParams())

// Controller chain: standard Cosmos key (secp256k1, coin type 118)
sdkcrypto.ImportKey(kr, "controller-key", "mnemonic.txt", "lumera", sdkcrypto.KeyTypeCosmos)

// Host chain: EVM-compatible key (eth_secp256k1, coin type 60)
sdkcrypto.ImportKey(kr, "host-key", "mnemonic.txt", "inj", sdkcrypto.KeyTypeEVM)
```

The ICA controller supports this via the `HostKeyName` config field (see [ica.md](ica.md)).

## Deriving addresses

`AddressFromKey` derives a bech32 address for any HRP without mutating global SDK config:

```go
addr, err := sdkcrypto.AddressFromKey(kr, "alice", "lumera")
```

For EVM keys (`KeyTypeEVM`), `EVMAddressFromKey` returns the 0x-prefixed EIP-55 hex address. The key must be `eth_secp256k1`; passing a Cosmos secp256k1 key returns an error.

```go
hexAddr, err := sdkcrypto.EVMAddressFromKey(kr, "host-key")
// e.g. 0xAbC1234...
```

`EVMToBech32` and `Bech32ToEVM` are byte-level conversions for 20-byte account identifiers. They round-trip EVM-derived accounts, but a legacy Cosmos `secp256k1` bech32 address does not prove or recover the key's Ethereum address; use `EVMAddressFromKey` when the keyring entry is available.

## Ethereum-format signing primitives

When wrapping a raw `go-ethereum` transaction yourself, the helpers in `pkg/crypto` produce an EIP-155 signed tx using the keyring:

```go
signed, err := sdkcrypto.SignEthereumTx(kr, "host-key", big.NewInt(1414), tx)
msg, err := sdkcrypto.WrapAsMsgEthereumTx(signed) // for broadcast as a cosmos MsgEthereumTx
from, err := sdkcrypto.RecoverSender(signed)      // sanity-check recovery
```

`Wei`, `ULume`, `ULumeDecToWei` bridge the 6 ↔ 18 decimal boundary defined by `x/precisebank` (factor `10^12`) — e.g. converting `feemarket` `BaseFee` (ulume decimal) to a wei-like integer suitable for an EIP-1559 fee field.
