# Lumera Go SDK – Developer Guide

This is the entry point for engineers building on Lumera. The guide is split by topic — each link below is self-contained.

## Topics

- **[Getting Started](guides/getting-started.md)** — prerequisites, install, configuration reference, creating a client, multi-signer factory, running examples, troubleshooting.
- **[Crypto Helpers](guides/crypto.md)** — `pkg/crypto`: key types (Cosmos / EVM), keyring creation, mnemonic import, address derivation (bech32 + 0x), Ethereum-format signing primitives, decimal bridging (`Wei` / `ULume`).
- **[Actions and SuperNodes](guides/actions.md)** — query / tx helpers for `x/action` and `x/supernode`.
- **[Cascade Storage](guides/cascade.md)** — file upload / download, event subscriptions, stepwise control over the SuperNode + on-chain flow.
- **[Interchain Accounts](guides/ica.md)** — `ica.Controller`, packet-building helpers, and when to use it vs. `interchaintest`.
- **[EVM Integration](guides/evm.md)** — `x/vm`, `x/erc20`, `x/feemarket`, `x/precisebank`, custom precompiles (`0x0901`/`0x0902`/`0x0903`), and `x/evmigration`.

## Reference

- [API.md](API.md) — exhaustive list of exported types, functions, and options.
- [docs/design/](design/) — design records that motivate the larger API surfaces.

## Examples

All examples live in `examples/`. The most useful entry points:

- `examples/query-actions` — read-only Action module queries.
- `examples/cascade-upload` / `cascade-download` — Cascade storage end-to-end.
- `examples/multi-account` — `client.NewFactory` with multiple signers.
- `examples/ica-request-tx` — build an ICA packet for a controller-chain broadcast.
- `examples/evm-transfer` — sign and broadcast an EIP-1559 transaction.
- `examples/evm-balance` — read EVM balance, nonce, base fee.
- `examples/erc20-convert` — `ConvertCoinToERC20` and `ConvertERC20ToCoin`.
- `examples/precompile-action` — invoke a Lumera precompile method.

Run `make examples` to compile them all.
