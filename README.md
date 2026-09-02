# ratchet

Deterministic AI-native software engineering framework for Go.

**Mission:** Zero Architectural Regression (Anti-Drift) — rigid architectural contracts, pure Go SSOT, and AST-level fitness functions so agents and humans cannot degrade the codebase.

## Hard Constraints Policy

True architectural lock-in is guaranteed at the **CI/CD boundary** via **Exit Code 1** and **server-side GitHub Branch Protection**. Local hooks and read-only attributes are soft friction only — useful for immediate developer feedback, bypassable from a shell.

| Layer | Mechanism | Hard? |
|-------|-----------|-------|
| Local hook | `ratchet init-hooks` → `.git/hooks/pre-commit` | No (bypassable) |
| CI | `.github/workflows/ratchet.yml` → `ratchet check --format=llm` | Yes (exit 1) |
| Server | `ratchet init-ci --protect-main` required status check | Yes (GitHub enforces) |

## Guardrails

- Pure Go SSOT only (no CUE / TypeSpec / TypeDB / Rego)
- No Go `.so` plugins (use wazero if isolation is needed)
- Validation via `go/ast`, property-based tests, testcontainers — not formal-verifier fantasy
- CLI: Cobra

## Install / build

```bash
go build -o bin/ratchet ./cmd/ratchet
```

## Commands

| Command | Purpose |
|---------|---------|
| `ratchet init` | Bootstrap `.cursorrules`, `ratchet.go` / `ratchet.json`, Claude skill, lock file |
| `ratchet check` | AST layer fitness + anti-drift verify (`--format=human` default, `--format=llm` for agents) |
| `ratchet gen` | Regenerate agent rules and re-lock contracts |
| `ratchet init-ci` | Write `.github/workflows/ratchet.yml`; `--protect-main` enables required checks via `gh` |
| `ratchet init-hooks` | Install local `pre-commit` soft friction |

## Layout

```
cmd/ratchet/          CLI
pkg/fitness/          AST architecture linter
pkg/antidrift/        SHA-256 contract lock / verify
pkg/report/           human + LLM error formatters
pkg/github/           CI workflow + branch protection helpers
pkg/hooks/            local git hook installer
pkg/skills/           .cursorrules + Claude skill generator
pkg/tokens/           Pure Go SSOT config structs
```

## License

Private — github.com/fow830/ratchet
