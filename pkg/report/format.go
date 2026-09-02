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

	RuleLayerIsolation = "LayerIsolation"
	RuleAntiDrift      = "AntiDrift"
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
	return llmBlock(
		RuleLayerIsolation,
		file,
		fmt.Sprintf(
			"package %q (layer %q) imports %q (layer %q); edge is forbidden",
			v.ImporterPkg, v.ImporterLayer, v.ImportPath, v.ImportedLayer,
		),
		fmt.Sprintf(
			"Remove import %q from %s, or move the dependency so layer %q only depends on allowed layers.",
			v.ImportPath, v.File, v.ImporterLayer,
		),
	)
}

// FormatHumanDiff returns the antidrift human report.
func FormatHumanDiff(d antidrift.Diff) string {
	return d.String()
}

// FormatLLMDiff returns one LLM block per drifted/missing/extra contract file.
func FormatLLMDiff(d antidrift.Diff) string {
	var blocks []string
	for _, c := range d.Changed {
		blocks = append(blocks, llmBlock(
			RuleAntiDrift,
			c.Path,
			fmt.Sprintf("contract hash mismatch expected=%s actual=%s", c.Expected, c.Actual),
			"Restore generated content or run `ratchet gen` then commit the updated ratchet.lock.",
		))
	}
	for _, m := range d.Missing {
		blocks = append(blocks, llmBlock(
			RuleAntiDrift,
			m,
			"locked contract file is missing on disk",
			"Restore the file or run `ratchet gen` to regenerate contracts and lock.",
		))
	}
	for _, e := range d.Extra {
		blocks = append(blocks, llmBlock(
			RuleAntiDrift,
			e,
			"declared contract file exists on disk but is absent from ratchet.lock",
			"Run `ratchet gen` to lock it, or remove it from ContractFiles in ratchet.json.",
		))
	}
	return strings.Join(blocks, "\n\n")
}

func llmBlock(rule, file, details, action string) string {
	return strings.Join([]string{
		"RULE_VIOLATION: " + rule,
		"FILE: " + file,
		"DETAILS: " + details,
		"ACTION_REQUIRED: " + action,
	}, "\n")
}
