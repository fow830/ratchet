# ratchet skill

Use this skill when changing architecture, contracts, or agent rules in module github.com/fow830/ratchet.

## RED→GREEN
1. Add/adjust `tests/contracts/*_contract_test.go` that fails.
2. Implement until `go test ./tests/contracts/...` passes.
3. Run `ratchet check --format=llm` — one RULE_VIOLATION = one COMMAND.

## Commands
- `ratchet init --preset=clean|vitek|hex --with-contracts`
- `ratchet check --profile=standard|strict|paranoid`
- `ratchet new-contract ARCH-001 --title="..."`
- `ratchet gen` / `ratchet lock` / `ratchet doctor`

## Rules
- Keep Pure Go SSOT.
- Never introduce Go .so plugins (use wazero).
- Enforce layer edges from tokens.Config.
