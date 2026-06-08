# Changelog

All notable changes to this project are documented in this file.

## [v1.2.0] - 2026-05-27

### Added

- Added EVM migration support with query helpers, migration message constructors, and transaction helpers for claiming legacy accounts and migrating validators.
- Added EVM address and signing utilities, including Ethereum address derivation, Bech32/EVM address conversion, wei/ulume conversion helpers, Ethereum transaction signing, sender recovery, and `MsgEthereumTx` wrapping.
- Registered cosmos/evm interfaces in the default transaction config.
- Added read-only clients for cosmos/evm modules: EVM, FeeMarket, and PreciseBank.
- Added Ethereum-format transaction support through `EVMClient`, including nonce resolution, gas estimation, EIP-1559 fee cap resolution, raw Ethereum transaction broadcast, contract deployment, and read-only contract calls.
- Added ERC20 module support with token-pair queries, coin/ERC20 conversion transaction helpers, and ABI helpers for balance, supply, allowance, and metadata reads.
- Added Lumera precompile support with embedded Action, Supernode, and Wasm ABIs plus generic precompile call/send wrappers.
- Added EVM-focused examples for balance queries, transfers, ERC20 conversion, and Action precompile usage.
- Added a Cosmos EVM API design document and split developer documentation into focused guides.

### Changed

- Refactored the base transaction build/sign pipeline to support explicit transaction build options, manual signer metadata, simulation-based gas estimation, fee overrides, and multi-message validation.
- Added configurable EVM options to the client configuration, including EVM chain ID, EVM native/extended denoms, and EVM gas caps.
- Exposed chain-economics overrides on the top-level client config (`AccountHRP`, `FeeDenom`, `GasPrice`) via `WithAccountHRP`, `WithFeeDenom`, and `WithGasPrice`, forwarding them into the blockchain config (previously settable only at the blockchain layer).
- Replaced the local ethsecp256k1 implementation with the cosmos/evm implementation and updated keyring handling for EVM keys.
- Updated dependency pins for the EVM stack, including Lumera, Cosmos SDK, cosmos/evm, the forked go-ethereum replacement, and SuperNode SDK `v2.5.2`.
- Kept local Lumera/SuperNode development `replace` directives commented out for release safety.

### Fixed

- Guarded Cosmos transaction building from accidentally routing `MsgEthereumTx` through the regular Cosmos signing pipeline.
- Fixed EVM denom resolution so transaction helpers can derive missing EVM denoms from chain params while still failing clearly when required values are unavailable.
- Tightened EVM nonce, gas, and fee cap validation, including negative-value checks and large gas fee overflow coverage.
- Hardened EVM API response handling for nil math values and malformed EVM transaction response data.
- `ImportKey` now verifies that an existing key was derived from the supplied mnemonic, instead of silently returning the stored key when a different mnemonic is imported under the same name.
- ERC20 read helpers (`Erc20Balance`/`Erc20TotalSupply`/`Erc20Allowance`/`Erc20Metadata`) return a clear error when the call target returns no data (no contract code or reverted), rather than a cryptic ABI unmarshal error.
- Gas estimation now surfaces simulation failures instead of silently falling back to a fixed gas limit that could under-gas the transaction; callers can bypass via `GasLimit` or `SkipSimulation`.
- `DeployContract` returns the zero address and an error when the constructor reverts, instead of an address where no contract was deployed.
- `buildEthereumTxBytes` rejects an empty native denom (previously panicked in `sdk.NewCoin`), `FeeMarket.BlockGas` rejects negative gas, and `GasUsed` is guarded against negative-to-`uint64` wraparound.
- The tx-wait helper surfaces the subscriber error when the poller also fails, preserving the more diagnostic root cause.
- Fixed linter issues in EVM examples, ERC20 helpers, and EVM transaction tests.

### Tests

- Added unit coverage for EVM address conversion, Ethereum signing, transaction wrapping, EVM query wrappers, Ethereum transaction byte construction, ERC20 helpers, precompile wrappers, EVMigration queries/messages, and transaction build pipeline validation.
- Added regression coverage for large `uint64` gas fee calculation.

## [v1.1.2] - 2026-05-15

### Changed

- Bumped the SuperNode SDK dependency to `v2.5.0-rc`.
- Refreshed module sums for the SuperNode dependency update.

## [v1.1.1] - 2026-03-26

### Changed

- Bumped the SuperNode SDK dependency to `v2.4.72`.
- Refreshed module sums for the SuperNode dependency update.

## [v1.1.0] - 2026-03-24

### Changed

- Updated Lumera and SuperNode SDK dependencies.
- Fixed compatibility issues related to the Cosmos SDK update.
- Updated release workflow permissions.
- Adjusted Cascade upload and SuperNode logging behavior for the dependency updates.

### Fixed

- Fixed linter issues.

## [v1.0.9] - 2026-02-16

### Changed

- Unified the `KeyType` API for multi-chain keyring usage.
- Added ICA `HostKeyName` support.
- Updated keyring docs and examples to use the unified key-type flow.
- Updated ICA controller types and helper code for host-key-name aware requests.

### Tests

- Updated crypto/keyring tests for the unified multi-chain keyring behavior.

## [v1.0.8] - 2026-01-30

### Changed

- Updated Cosmos SDK to `v0.53.5`.
- Updated Lumera dependency to `v1.10.0`.
- Updated Go setup/release workflow configuration for the dependency bump.
- Refreshed repository guidance for the newer dependency stack.

## [v1.0.7] - 2026-01-30

### Added

- Added multi-chain keyring support.
- Added Injective key support.
- Added ethsecp256k1 test coverage.

### Fixed

- Fixed verify logic in the ICA request verification example.
- Fixed verify behavior for modified injected keyring flows.

### Changed

- Bumped the SuperNode SDK dependency.
- Updated ICA request and verification examples for the expanded keyring support.
- Updated build tooling and module sums for the new crypto/keyring dependencies.

## [v1.0.6] - 2026-01-15

### Added

- Added ICA controller support.
- Added standalone ICA package helpers for acknowledgements, packing, controller operations, and request helpers.
- Added ICA examples for request, approval, and multi-account flows.

### Changed

- Refactored the blockchain client base.
- Split lower-level blockchain client configuration and transaction building into the `blockchain/base` package.
- Updated high-level client options and configuration wiring around the refactored base client.
- Updated Cascade upload/download/request flows to use the refactored client and ICA helpers.
- Expanded API and developer documentation for the refactored client and ICA flows.

### Tests

- Added unit tests for ICA acknowledgement helpers, packet packing, controller helpers, Cascade request handling, and base transaction behavior.

## [v1.0.5] - 2026-01-10

### Added

- Added ICA flow support for Cascade actions.
- Added ICA acknowledgement helpers.
- Added ICA packet packing helpers and tests.
- Added ICA request/approval examples and multi-account examples.

### Changed

- Updated Cascade upload/download flows to support ICA-backed action workflows.
- Updated action approval and claim-token examples.
- Expanded README, API, and developer guide documentation for ICA and Cascade flows.
- Moved crypto/keyring helper code under `pkg/crypto`.

### Tests

- Added tests for ICA acknowledgement handling, packet packing, and Cascade request behavior.

## [v1.0.4] - 2025-12-28

### Changed

- Updated dependencies to their latest compatible versions at the time of release.
- Updated Action and Cascade request structures for file-size metadata in kilobytes.
- Refreshed API and developer documentation for the Action/Cascade changes.
- Updated README guidance for the dependency and request changes.

## [v1.0.3] - 2025-12-10

### Fixed

- Handled transaction indexing lag in `WaitForTxInclusion`.
- Added retry behavior when a transaction is observed but not yet available through the transaction query service.
- Updated transaction waiting tests for delayed indexing behavior.

### Changed

- Updated CI workflow configuration for lint, release, and test jobs.

## [v1.0.2] - 2025-12-03

### Added

- Added SDK event changes from the event-change branch.
- Added Cascade event type support.
- Added event handling updates in Cascade client/upload flows.

### Changed

- Updated SuperNode SDK dependency to `v2.4.10`.
- Updated Cascade upload example for the event changes.
- Refreshed module sums for the SuperNode and event dependency updates.

### Fixed

- Fixed linter warnings and formatting issues.

## [v1.0.1] - 2025-11-26

### Changed

- Added `go.sum` verification to the Makefile flow.
- Updated Lumera dependency to `v1.8.5`.
- Updated SuperNode SDK dependency to `v2.4.9`.
- Refreshed module sums for the Lumera and SuperNode dependency updates.

## [v1.0.0] - 2025-11-20

### Added

- Initial SDK release.
- Added address helper support.
- Added ICA helpers and examples.
- Added Action approval support.
- Added Cascade upload support with request creation, request sending, and SuperNode upload stages.
- Added Cascade action status checks after file upload.
- Added blockchain message constructors and transaction handling for Action/Cascade flows.
- Added README and developer/API documentation for the first SDK surface.

### Changed

- Split upload flow into request creation, request sending, and SuperNode upload steps.
- Added action status checks after Cascade uploads.
- Refactored message constructors and transaction handling for Cascade flows.
- Updated Lumera dependency to `v1.8.4`.
- Updated SuperNode SDK dependency to `v2.4.2`.
