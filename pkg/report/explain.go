package report

import (
	"fmt"
	"strings"

	"github.com/fow830/ratchet/pkg/gates"
	"github.com/fow830/ratchet/pkg/tokens"
)

// ExplainResult returns human+agent guidance for a failed check.
func ExplainResult(r ExtendedResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "profile=%s ok=%v\n", r.Profile, r.OK)
	for _, v := range r.Violations {
		fmt.Fprintf(&b, "- arch: %s\n  fix: remove forbidden import or adjust allowed_edges in %s\n",
			v.String(), tokens.ConfigFileName)
	}
	if !r.Drift.OK() {
		fmt.Fprintf(&b, "- drift: %s\n  fix: run `%s gen` or restore contract files\n",
			strings.TrimSpace(FormatPlainDiff(r.Drift)), tokens.ToolName)
	}
	for _, d := range r.DocsViolations {
		fmt.Fprintf(&b, "- docs: %s\n  fix: delete or add to allowed_prose_docs\n", d.String())
	}
	for _, f := range r.Failures {
		fmt.Fprintf(&b, "- gate[%s]: %s\n  fix: %s\n", f.Gate, f.Message, gateFix(f))
	}
	if b.Len() == 0 {
		return "nothing to explain: last check was ok"
	}
	return b.String()
}

func gateFix(f gates.Failure) string {
	switch f.Gate {
	case "cycles":
		return "break the import cycle between listed packages"
	case "coverage":
		return "raise test coverage or lower quality.coverage_min_pct"
	case "mutation":
		return "kill more mutants or lower quality.mutation_min_pct"
	case "bench":
		return "fix regression or refresh with ratchet bench-lock"
	case "plugin_lock", "wasm":
		return "re-lock plugins or restore plugin wasm bytes"
	default:
		return fmt.Sprintf("fix %s gate or adjust profile/gates in %s", f.Gate, tokens.ConfigFileName)
	}
}

// FormatLLMViolationV2 is one-violation-one-fix with explicit shell command.
func FormatLLMViolationV2(v interface{ String() string }, file, details, action, command string) string {
	return strings.Join([]string{
		"RULE_VIOLATION: " + RuleLayerIsolation,
		"FILE: " + file,
		"DETAILS: " + details,
		"ACTION_REQUIRED: " + action,
		"COMMAND: " + command,
	}, "\n")
}
