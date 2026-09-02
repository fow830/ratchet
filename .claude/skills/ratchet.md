# ratchet skill

Use this skill when changing architecture, contracts, or agent rules in module github.com/fow830/ratchet.

## Commands
- `ratchet init` — bootstrap .cursorrules and ratchet.go
- `ratchet check` — AST fitness + anti-drift verify
- `ratchet gen` — regenerate agent skill rules and lock contracts

## Rules
- Keep Pure Go SSOT.
- Never introduce Go .so plugins.
- Enforce layer edges from tokens.Config.
