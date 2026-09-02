// Package report formats ratchet check results for humans and LLMs.
package report

import (
	"fmt"
	"strings"

	"github.com/fow830/ratchet/pkg/antidrift"
	"github.com/fow830/ratchet/pkg/fitness"
)

const (
	FormatHuman = "human"
	FormatLLM   = "llm"
)

// FormatHumanViolation returns a single-line human-readable violation.
func FormatHumanViolation(v fitness.Violation) string {
	return v.String()
}

// FormatLLMViolation returns a dry, structured block for LLM agents.
func FormatLLMViolation(v fitness.Violation) string {
	file := v.File
	if v.Line > 0 {
		file = fmt.Sprintf("%s:%d", v.File, v.Line)
	}
	details := fmt.Sprintf(
		"package %q (layer %q) imports %q (layer %q); edge is forbidden",
		v.ImporterPkg, v.ImporterLayer, v.ImportPath, v.ImportedLayer,
	)
	action := fmt.Sprintf(
		"Remove import %q from %s, or move the dependency so layer %q only depends on allowed layers.",
		v.ImportPath, v.File, v.ImporterLayer,
	)
	return strings.Join([]string{
		"RULE_VIOLATION: LayerIsolation",
		"FILE: " + file,
		"DETAILS: " + details,
		"ACTION_REQUIRED: " + action,
	}, "\n")
}

// FormatHumanDiff returns the antidrift human report.
func FormatHumanDiff(d antidrift.Diff) string {
	return d.String()
}

// FormatLLMDiff returns one LLM block per drifted/missing contract file.
func FormatLLMDiff(d antidrift.Diff) string {
	var blocks []string
	for _, c := range d.Changed {
		blocks = append(blocks, strings.Join([]string{
			"RULE_VIOLATION: AntiDrift",
			"FILE: " + c.Path,
			fmt.Sprintf("DETAILS: contract hash mismatch expected=%s actual=%s", c.Expected, c.Actual),
			"ACTION_REQUIRED: Restore generated content or run `ratchet gen` then commit the updated ratchet.lock.",
		}, "\n"))
	}
	for _, m := range d.Missing {
		blocks = append(blocks, strings.Join([]string{
			"RULE_VIOLATION: AntiDrift",
			"FILE: " + m,
			"DETAILS: locked contract file is missing on disk",
			"ACTION_REQUIRED: Restore the file or run `ratchet gen` to regenerate contracts and lock.",
		}, "\n"))
	}
	for _, e := range d.Extra {
		blocks = append(blocks, strings.Join([]string{
			"RULE_VIOLATION: AntiDrift",
			"FILE: " + e,
			"DETAILS: unexpected unlocked contract file present",
			"ACTION_REQUIRED: Remove the file or add it to ContractFiles and run `ratchet gen`.",
		}, "\n"))
	}
	return strings.Join(blocks, "\n\n")
}
