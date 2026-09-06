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
| `ratchet graph` / `diff-lock` / `analyze` / `fuzz-init` / `observe` | Mermaid graph, config-diff, escape regex, fuzz seed file, PATH presence check |
| `ratchet init-ci` / `init-hooks --lrt-verify` / `init-example` | CI (incl. cosign), hooks, reference service |
| `ratchet completion bash\|zsh\|fish` | Shell completion |
| `go run ./cmd/tokensgen` | Generate env/compose/dockerfile/sqlc stubs |

HTTP smoke: `pkg/smoke`. Tool presence (`exec.LookPath`): `pkg/observe` — not eBPF/Cilium telemetry.

### Exit codes

| Code | Meaning |
|------|---------|
| 0 | OK |
| 1 | Architecture / contract / gate failure |
| 2 | System / parse / flag error |

## Layout

```
cmd/ratchet/          CLI
cmd/tokensgen/        SSOT stub codegen
schema/               JSON Schema for ratchet.json
examples/service/     reference vitek service
tests/contracts/      dogfood contracts
```

### Core (used by `ratchet check` / setup)

| Package | Role |
|---------|------|
| `pkg/tokens` | SSOT: config, presets, profiles, validate, migrate, IO |
| `pkg/fitness` | AST layer edges, cycles, external + test imports |
| `pkg/antidrift` | SHA + render lock |
| `pkg/gates` | Profile orchestrator |
| `pkg/plugins` | wazero WASM rules + plugin lock |
| `pkg/benchlock` | `ratchet.bench` baseline |
| `pkg/hooks` | pre-commit / commit-msg installer (soft friction; CI is hard) |
| `pkg/github` | branch protection API (`init-ci --protect-main`) |
| `pkg/report` | human / llm / json / SARIF |
| `pkg/skills` | `.cursorrules` + `.claude/skills` generator |
| `pkg/doctor` | health-check (go.mod, config, lock, schema, git) |
| `pkg/docs` | prose allowlist |
| `pkg/contracts` | scaffold + httpassert |

### Utilities (narrow, working)

| Package | Role |
|---------|------|
| `pkg/graph` | Mermaid layer graph (reuses `fitness.LayerOf`) |
| `pkg/smoke` | HTTP GET probe (status / body / timeout) |
| `pkg/analyze` | regex filter on `go build -gcflags=all=-m` “escapes to heap” |
| `pkg/breaking` | `ratchet.json` layer/edge config-diff (+ git helper); not buf-style API breaking |
| `pkg/generate` | tokensgen renderers |
| `pkg/workspace` | `go.work` multi-module discovery |

### Scaffolding (stubs / presence only — not full drift gates)

| Package | Honest scope |
|---------|----------------|
| `pkg/codegen` | detects sqlc/buf/openapi/cue config file presence (`os.Stat`); does not run or diff them |
| `pkg/fuzzinit` | creates a fuzz corpus dir + `seed` stub file; does not run `go test -fuzz` |
| `pkg/observe` | `exec.LookPath` for go/pprof/cilium/hubble; presence check, not a runtime probe |

## License

Public repo — [github.com/fow830/ratchet](https://github.com/fow830/ratchet) (no `LICENSE` file yet).
