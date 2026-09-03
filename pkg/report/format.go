// Package report formats ratchet check results for humans, LLMs, JSON, and SARIF.
package report

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fow830/ratchet/pkg/antidrift"
	"github.com/fow830/ratchet/pkg/docs"
	"github.com/fow830/ratchet/pkg/fitness"
	"github.com/fow830/ratchet/pkg/gates"
	"github.com/fow830/ratchet/pkg/tokens"
)

const (
	FormatText  = "text"
	FormatHuman = "human" // alias of text
	FormatLLM   = "llm"
	FormatJSON  = "json"
	FormatSARIF = "sarif"

	RuleLayerIsolation = "LayerIsolation"
	RuleAntiDrift      = "AntiDrift"
	RuleDocsPolicy     = "DocsPolicy"
	RuleGateFailure    = "GateFailure"
)

// SupportedFormats is the canonical help string for --format.
const SupportedFormats = FormatText + "|" + FormatJSON + "|" + FormatSARIF + "|" + FormatLLM

const (
	ansiReset  = "\033[0m"
	ansiRed    = "\033[31m"
	ansiYellow = "\033[33m"
	ansiGreen  = "\033[32m"
	ansiBold   = "\033[1m"
)

// Result is the machine-readable check payload.
type Result struct {
	OK         bool                `json:"ok"`
	Violations []fitness.Violation `json:"violations"`
	Drift      antidrift.Diff      `json:"drift"`
}

// Normalize fills nil slices so JSON/SARIF emit [] instead of null.
func (r *Result) Normalize() {
	if r.Violations == nil {
		r.Violations = []fitness.Violation{}
	}
	if r.Drift.Changed == nil {
		r.Drift.Changed = []antidrift.ChangedFile{}
	}
	if r.Drift.Missing == nil {
		r.Drift.Missing = []string{}
	}
	if r.Drift.Extra == nil {
		r.Drift.Extra = []string{}
	}
}

// NormalizeFormat maps aliases to canonical format names.
func NormalizeFormat(f string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(f)) {
	case FormatText, FormatHuman:
		return FormatText, nil
	case FormatLLM:
		return FormatLLM, nil
	case FormatJSON:
		return FormatJSON, nil
	case FormatSARIF:
		return FormatSARIF, nil
	default:
		return "", fmt.Errorf("unsupported --format %q (want %s)", f, SupportedFormats)
	}
}

// FormatTextViolation returns ANSI-colored terminal output for a violation.
func FormatTextViolation(v fitness.Violation) string {
	return fmt.Sprintf("%s%serror%s %s%s%s", ansiBold, ansiRed, ansiReset, ansiYellow, v.String(), ansiReset)
}

// FormatTextDiff returns ANSI-colored antidrift report.
func FormatTextDiff(d antidrift.Diff) string {
	if d.OK() {
		return ansiGreen + "antidrift: ok" + ansiReset
	}
	var b strings.Builder
	b.WriteString(ansiBold + ansiRed + "antidrift: drift detected" + ansiReset + "\n")
	for _, c := range d.Changed {
		fmt.Fprintf(&b, "  %schanged%s %s\n    expected %s\n    actual   %s\n", ansiYellow, ansiReset, c.Path, c.Expected, c.Actual)
	}
	for _, m := range d.Missing {
		fmt.Fprintf(&b, "  %smissing%s %s\n", ansiYellow, ansiReset, m)
	}
	for _, e := range d.Extra {
		fmt.Fprintf(&b, "  %sextra%s   %s\n", ansiYellow, ansiReset, e)
	}
	return b.String()
}

// FormatPlainDiff returns the antidrift report without ANSI codes.
func FormatPlainDiff(d antidrift.Diff) string {
	return d.String()
}

// FormatLLMViolation returns a dry, structured block for LLM agents (v2: includes COMMAND).
func FormatLLMViolation(v fitness.Violation) string {
	file := v.File
	if v.Line > 0 {
		file = fmt.Sprintf("%s:%d", v.File, v.Line)
	}
	return strings.Join([]string{
		"RULE_VIOLATION: " + RuleLayerIsolation,
		"FILE: " + file,
		"DETAILS: " + fmt.Sprintf(
			"package %q (layer %q) imports %q (layer %q); edge is forbidden",
			v.ImporterPkg, v.ImporterLayer, v.ImportPath, v.ImportedLayer,
		),
		"ACTION_REQUIRED: " + fmt.Sprintf(
			"Remove import %q from %s, or move the dependency so layer %q only depends on allowed layers.",
			v.ImportPath, v.File, v.ImporterLayer,
		),
		"COMMAND: " + tokens.ToolName + " check --format=" + FormatLLM,
	}, "\n")
}

// FormatLLMDiff returns one LLM block per drifted/missing/extra contract file.
func FormatLLMDiff(d antidrift.Diff) string {
	var blocks []string
	for _, c := range d.Changed {
		blocks = append(blocks, llmBlockCmd(
			RuleAntiDrift,
			c.Path,
			fmt.Sprintf("contract hash mismatch expected=%s actual=%s", c.Expected, c.Actual),
			fmt.Sprintf(
				"Restore generated content or run `%s gen` then commit the updated %s.",
				tokens.ToolName, tokens.LockFileName,
			),
			tokens.ToolName+" gen",
		))
	}
	for _, m := range d.Missing {
		blocks = append(blocks, llmBlockCmd(
			RuleAntiDrift,
			m,
			"locked contract file is missing on disk",
			fmt.Sprintf("Restore the file or run `%s gen` to regenerate contracts and lock.", tokens.ToolName),
			tokens.ToolName+" gen",
		))
	}
	for _, e := range d.Extra {
		blocks = append(blocks, llmBlockCmd(
			RuleAntiDrift,
			e,
			fmt.Sprintf("declared contract file exists on disk but is absent from %s", tokens.LockFileName),
			fmt.Sprintf(
				"Run `%s gen` to lock it, or remove it from ContractFiles in %s.",
				tokens.ToolName, tokens.ConfigFileName,
			),
			tokens.ToolName+" lock",
		))
	}
	return strings.Join(blocks, "\n\n")
}

// FormatLLMDocs returns an LLM block for docs policy violations.
func FormatLLMDocs(v docs.Violation) string {
	return llmBlock(
		RuleDocsPolicy,
		v.Path,
		v.String(),
		fmt.Sprintf("Remove %s or add it to allowed_prose_docs in %s.", v.Path, tokens.ConfigFileName),
	)
}

// FormatLLMFailure returns an LLM block for gate failures.
func FormatLLMFailure(f gates.Failure) string {
	return llmBlock(
		RuleGateFailure,
		f.Gate,
		f.Message,
		fmt.Sprintf("Fix gate %q failure or adjust profile/gates in %s.", f.Gate, tokens.ConfigFileName),
	)
}

// MarshalJSON marshals Result as indented JSON.
func MarshalJSON(r Result) ([]byte, error) {
	r.Normalize()
	return json.MarshalIndent(r, "", "  ")
}

func llmBlock(rule, file, details, action string) string {
	return llmBlockCmd(rule, file, details, action, tokens.ToolName+" check --format="+FormatLLM)
}

func llmBlockCmd(rule, file, details, action, command string) string {
	return strings.Join([]string{
		"RULE_VIOLATION: " + rule,
		"FILE: " + file,
		"DETAILS: " + details,
		"ACTION_REQUIRED: " + action,
		"COMMAND: " + command,
	}, "\n")
}

func toFileURI(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	if strings.HasPrefix(path, "file:") {
		return path
	}
	return path
}
