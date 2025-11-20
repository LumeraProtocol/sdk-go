# Lumera SDK Agent Guide

## Architecture & Config
- `client.Client` (`client/client.go`) fans out to both `blockchain.Client` and `cascade.Client`; add new modules by extending `client.New` so options/config stay in lockstep.
- `client/config.go` hydrates timeouts, 50 MB gRPC limits, retry counts, and `WaitTx` knobs; use `DefaultConfig()` in samples/tests to avoid zero durations.
- `blockchain/client.go` enforces `ensureLumeraBech32Prefixes` and selects TLS using `shouldUseTLS`; follow this heuristic instead of forcing creds per call.
- `cascade/client.go` wraps the SuperNode SDK via `snconfig.NewConfig`; new Cascade entry points must reuse the same keyring/key name to share signatures.

## Transactions & Waiting
- `blockchain/tx.go` centralizes the tx lifecycle: build with `NewDefaultTxConfig`, resolve account/sequence, simulate gas (adds 30 % buffer), set min `ulume` fees (0.025 gas price), sign with `internal/crypto.SignTxWithKeyring`.
- Always broadcast through `Client.Broadcast` + `WaitForTxInclusion`; the waiter (`internal/wait-tx`) prefers CometBFT websockets via `WaitTxConfig.RPCEndpoint` and falls back to gRPC polling.
- `blockchain.Client.ExtractEventAttribute` is how examples retrieve IDs; keep event parsing there so higher layers stay clean.

## Queries & Module Patterns
- `blockchain/action.go` + `blockchain/query.go` expose `WithActionType(Str|Enum)` and `WithPagination` helpers; they normalize enum casing and wire `query.PageRequest`.
- `types.ActionFromProto` decodes Cascade/Sense metadata to typed structs; never re-unmarshal metadata blobs outside `types`.
- `blockchain/supernode.go` converts proto responses with `types.SuperNodeFromProto` which picks the highest-height IP/state—mirror this when surfacing new fields.
- Message constructors (e.g., `NewMsgRequestAction`, `NewMsgRegisterSupernode`) live alongside their modules; reuse them so string/enum handling stays consistent.

## Cascade Flows
- `cascade/cascade.go` uploads: build metadata via SuperNode SDK, call `blockchain.Client.RequestActionTx`, then start storage with `snClient.StartCascade`; downloads reuse the same client/key for signatures.
- `cascade.TaskManager` polls `snsdk.Client.GetTask` every second until `COMPLETED`/`FAILED`; extend this manager for new task types instead of bespoke loops.

## Keyring & Examples
- `internal/crypto.NewKeyring`, `GetKey`, and `NewDefaultTxConfig` wrap Cosmos keyring and proto registration; all examples (`examples/*`) show the expected bootstrapping/flag pattern.
- `ensureLumeraBech32Prefixes` seals global Bech32 config once; avoid calling Cosmos prefix setters elsewhere to prevent panics.

## Developer Workflows
- `make sdk` (same as `make build`) compiles all packages; `make examples` or `make example-<name>` drops binaries into `build/`.
- `make test` runs `go test -race -coverprofile=coverage.out ./...` and emits `coverage.html`; `make lint` requires `golangci-lint`.
- Stick to Go 1.25.1 (per `go.mod`) and respect the `replace` pins for CometBFT/Cosmos; update both blockchain + SuperNode deps together when bumping versions.

## Conventions
- Errors wrap context with `fmt.Errorf("context: %w", err)` so callers can unwrap lower layers.
- All public methods accept `context.Context`; propagate caller deadlines rather than creating `context.Background()` internally.
- Honor `MaxRecvMsgSize`/`MaxSendMsgSize` from config when introducing new gRPC connections; large Cascade payloads depend on these defaults.
- Update `README.md` examples whenever `client.Config` signatures change so onboarding stays accurate.
