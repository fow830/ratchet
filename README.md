# ratchet

Deterministic AI-native anti-drift framework for Go.

**Mission:** Zero Architectural Regression — rigid architectural contracts, pure Go SSOT, AST fitness, contract tests, and profile-based quality gates so agents and humans cannot quietly degrade the codebase.

## Hard Constraints

| Layer | Mechanism | Hard? |
|-------|-----------|-------|
| Local hook | `ratchet init-hooks` (+ `--lrt-verify` commit-msg) | Soft |
| CI | `.github/workflows/ratchet.yml` → `check --profile=strict` | Hard (exit 1) |
| Nightly | `check --profile=paranoid` | Hard |
| Server | `init-ci --protect-main` | Hard (needs GitHub Pro on private personal repos) |

## Profiles

`minimal` → `standard` → `service` / `api` → `strict` → `paranoid`

## Commands

| Command | Purpose |
|---------|---------|
| `ratchet init --preset=clean\|vitek\|hex --with-contracts` | Bootstrap SSOT + optional contracts |
| `ratchet check [--profile=…] [--workspace]` | All enabled gates |
| `ratchet explain` | Human fix guidance for failures |
| `ratchet gen` / `lock` / `bench-lock` | Regenerate rules / SHA lock / bench baseline |
| `ratchet new-contract ID` | Scaffold `tests/contracts` |
| `ratchet doctor` / `validate-config` / `migrate-config` | Setup & schema |
| `ratchet gen-tokens` / `plugin-lock` / `smoke --url=` | Tokens stubs, WASM plugin lock, HTTP smoke |
| `ratchet graph` / `diff-lock` / `analyze` / `fuzz-init` / `observe` | Graph, breaking diff, escape hints, fuzz seed, runtime tool probe |
| `ratchet init-ci` / `init-hooks --lrt-verify` / `init-example` | CI (incl. cosign), hooks, reference service |
| `ratchet completion bash\|zsh\|fish` | Shell completion |
| `go run ./cmd/tokensgen` | Generate env/compose/dockerfile/sqlc stubs |

HTTP smoke helpers: `pkg/smoke`. Runtime probes: `pkg/observe` (go/pprof/cilium/hubble).

### Exit codes

| Code | Meaning |
|------|---------|
| 0 | OK |
| 1 | Architecture / contract / gate failure |
| 2 | System / parse / flag error |

## Layout

```
cmd/ratchet/          CLI
cmd/tokensgen/        SSOT codegen
pkg/fitness/          AST + cycles + external + test imports
pkg/antidrift/        SHA + render lock
pkg/gates/            Profile orchestrator
pkg/plugins/          wazero WASM rules + plugin lock
pkg/benchlock/        ratchet.bench baseline
pkg/contracts/        scaffold + httpassert
pkg/docs/             prose allowlist
pkg/generate/         tokensgen renderers
pkg/workspace/        go.work multi-module
schema/               JSON Schema for ratchet.json
examples/service/     reference vitek service
tests/contracts/      dogfood contracts
```

## License

Private — github.com/fow830/ratchet
