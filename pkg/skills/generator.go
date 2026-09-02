// Package skills generates agent rule files from pure-Go SSOT config.
package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fow830/ratchet/pkg/tokens"
)

// Generator writes .cursorrules and Claude skill stubs.
type Generator struct {
	cfg tokens.Config
}

// NewGenerator constructs a Generator from SSOT config.
func NewGenerator(cfg tokens.Config) *Generator {
	return &Generator{cfg: cfg}
}

// Generate writes agent skill artifacts under root.
func (g *Generator) Generate(root string) error {
	if err := os.WriteFile(filepath.Join(root, tokens.CursorRules), []byte(g.CursorRules()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", tokens.CursorRules, err)
	}

	skillPath := filepath.Join(root, filepath.FromSlash(tokens.ClaudeSkillRel))
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		return fmt.Errorf("mkdir skills: %w", err)
	}
	if err := os.WriteFile(skillPath, []byte(g.ClaudeSkill()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", tokens.ClaudeSkillRel, err)
	}
	return nil
}

// CursorRules returns the deterministic .cursorrules body.
func (g *Generator) CursorRules() string {
	froms := make([]string, 0, len(g.cfg.AllowedEdges))
	for from := range g.cfg.AllowedEdges {
		froms = append(froms, from)
	}
	sort.Strings(froms)

	var edges []string
	for _, from := range froms {
		tos := append([]string(nil), g.cfg.AllowedEdges[from]...)
		sort.Strings(tos)
		if len(tos) == 0 {
			edges = append(edges, fmt.Sprintf("- %s → (none; leaf)", from))
			continue
		}
		edges = append(edges, fmt.Sprintf("- %s → %s", from, strings.Join(tos, ", ")))
	}

	return fmt.Sprintf(`# ratchet — Zero Architectural Regression (Anti-Drift)

Module: %s

## Absolute Guardrails
1. Pure Go SSOT Only — no external DSLs (CUE, TypeSpec, TypeDB, Rego/OPA).
2. NO Go .so plugins — use WebAssembly (wazero) if isolation is required.
3. No academic formal-verifier translators (TLA+, Lean4). Validate with go/ast, PBT, testcontainers.
4. Standard Go tooling only: go/ast, go/parser, go/token, golang.org/x/tools, cobra.

## Layer Edges
%s

## Agent Protocol
- Prefer tests first; keep gofmt/go vet clean.
- Run `+"`ratchet check`"+` before claiming architecture is green.
- Do not manually edit locked contract files without regenerating and re-locking.
`, escapeMarkdown(g.cfg.Module), strings.Join(edges, "\n"))
}

// ClaudeSkill returns the deterministic Claude skill markdown body.
func (g *Generator) ClaudeSkill() string {
	mod := escapeMarkdown(g.cfg.Module)
	return fmt.Sprintf(`# ratchet skill

Use this skill when changing architecture, contracts, or agent rules in module %s.

## Commands
- `+"`ratchet init`"+` — bootstrap .cursorrules and ratchet.json
- `+"`ratchet check`"+` — AST fitness + anti-drift verify
- `+"`ratchet gen`"+` — regenerate agent skill rules and lock contracts

## Rules
- Keep Pure Go SSOT.
- Never introduce Go .so plugins.
- Enforce layer edges from tokens.Config.
`, mod)
}

// escapeMarkdown escapes backticks and angle brackets in untrusted context fragments.
func escapeMarkdown(s string) string {
	s = strings.ReplaceAll(s, "`", "'")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
