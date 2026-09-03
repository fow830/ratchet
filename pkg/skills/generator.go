package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fow830/ratchet/pkg/report"
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
	if err := os.WriteFile(filepath.Join(root, tokens.CursorRules), []byte(g.CursorRules()), tokens.FileModeFile); err != nil {
		return fmt.Errorf("write %s: %w", tokens.CursorRules, err)
	}

	skillPath := filepath.Join(root, filepath.FromSlash(tokens.ClaudeSkillRel))
	if err := os.MkdirAll(filepath.Dir(skillPath), tokens.FileModeDir); err != nil {
		return fmt.Errorf("mkdir skills: %w", err)
	}
	if err := os.WriteFile(skillPath, []byte(g.ClaudeSkill()), tokens.FileModeFile); err != nil {
		return fmt.Errorf("write %s: %w", tokens.ClaudeSkillRel, err)
	}
	if g.cfg.Preset != "" && g.cfg.Preset != tokens.PresetClean {
		presetSkill := filepath.Join(root, filepath.FromSlash(tokens.PresetSkillRel(g.cfg.Preset)))
		if err := os.WriteFile(presetSkill, []byte(g.PresetSkill()), tokens.FileModeFile); err != nil {
			return fmt.Errorf("write preset skill: %w", err)
		}
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

	tool := tokens.ToolName
	profile := g.cfg.Profile
	if profile == "" {
		profile = tokens.ProfileStandard
	}
	return fmt.Sprintf(`# %s — Zero Architectural Regression (Anti-Drift)

Module: %s
Preset: %s
Profile: %s

## Absolute Guardrails
1. Pure Go SSOT in %s (+ optional Go codegen: sqlc/buf/oapi via profiles).
2. NO Go .so plugins — use WebAssembly (wazero) rule packs.
3. No academic formal-verifier translators (TLA+, Lean4) in core.
4. Validate with go/ast, PBT, testcontainers, race/fuzz/mutation gates.

## Layer Edges
%s

## Agent Protocol (RED→GREEN)
1. Write or extend a failing contract in %s/ first (RED).
2. Implement the minimal fix (GREEN).
3. Run `+"`"+`%s check --format=%s`+"`"+` and fix every RULE_VIOLATION.
4. Run `+"`"+`%s gen`+"`"+` / `+"`"+`%s lock`+"`"+` before committing contract/SSOT changes.
5. When contracts change, include LRT-VERIFY in the commit message.
`, tool, escapeMarkdown(g.cfg.Module), g.cfg.Preset, profile, tokens.ConfigFileName, strings.Join(edges, "\n"),
		tokens.ContractsDirDefault, tool, report.FormatLLM, tool, tool)
}

// ClaudeSkill returns the deterministic Claude skill markdown body.
func (g *Generator) ClaudeSkill() string {
	mod := escapeMarkdown(g.cfg.Module)
	tool := tokens.ToolName
	profiles := tokens.ProfileStandard + "|" + tokens.ProfileStrict + "|" + tokens.ProfileParanoid
	presets := tokens.PresetClean + "|" + tokens.PresetVitek + "|" + tokens.PresetHex
	return fmt.Sprintf(`# %s skill

Use this skill when changing architecture, contracts, or agent rules in module %s.

## RED→GREEN
1. Add/adjust `+"`"+`%s/*%s`+"`"+` that fails.
2. Implement until `+"`"+`go test ./%s/...`+"`"+` passes.
3. Run `+"`"+`%s check --format=%s`+"`"+` — one RULE_VIOLATION = one COMMAND.

## Commands
- `+"`"+`%s init --preset=%s --with-contracts`+"`"+`
- `+"`"+`%s check --profile=%s`+"`"+`
- `+"`"+`%s new-contract ARCH-001 --title="..."`+"`"+`
- `+"`"+`%s gen`+"`"+` / `+"`"+`%s lock`+"`"+` / `+"`"+`%s doctor`+"`"+`

## Rules
- Keep Pure Go SSOT.
- Never introduce Go .so plugins (use wazero).
- Enforce layer edges from tokens.Config.
`, tool, mod,
		tokens.ContractsDirDefault, tokens.ContractTestSuffix,
		tokens.ContractsDirDefault,
		tool, report.FormatLLM,
		tool, presets,
		tool, profiles,
		tool, tool, tool, tool)
}

// PresetSkill returns a preset-specific skill body.
func (g *Generator) PresetSkill() string {
	return fmt.Sprintf("# %s preset: %s\n\nEnforce layer edges and contracts for preset %q.\n",
		tokens.ToolName, g.cfg.Preset, g.cfg.Preset)
}

func escapeMarkdown(s string) string {
	s = strings.ReplaceAll(s, "`", "'")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
