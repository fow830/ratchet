package report_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/fow830/ratchet/pkg/antidrift"
	"github.com/fow830/ratchet/pkg/fitness"
	"github.com/fow830/ratchet/pkg/report"
	"github.com/fow830/ratchet/pkg/tokens"
)

func sampleResult() report.Result {
	return report.Result{
		OK: false,
		Violations: []fitness.Violation{{
			File:          "internal/domain/doc.go",
			Line:          2,
			ImportPath:    "example.com/app/internal/usecase",
			ImporterPkg:   "example.com/app/internal/domain",
			ImporterLayer: "domain",
			ImportedLayer: "usecase",
		}},
		Drift: antidrift.Diff{
			Changed: []antidrift.ChangedFile{{Path: tokens.ConfigFileName, Expected: "aaa", Actual: "bbb"}},
			Missing: []string{},
			Extra:   []string{"extra.txt"},
		},
	}
}

func TestNormalizeFormat(t *testing.T) {
	tests := []struct {
		in, want string
		err      bool
	}{
		{"text", report.FormatText, false},
		{"human", report.FormatText, false},
		{"json", report.FormatJSON, false},
		{"sarif", report.FormatSARIF, false},
		{"llm", report.FormatLLM, false},
		{"xml", "", true},
	}
	for _, tc := range tests {
		got, err := report.NormalizeFormat(tc.in)
		if tc.err {
			if err == nil {
				t.Fatalf("%q: expected error", tc.in)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Fatalf("%q: got (%q,%v) want %q", tc.in, got, err, tc.want)
		}
	}
}

func TestFormatJSON_Marshal(t *testing.T) {
	data, err := report.MarshalJSON(sampleResult())
	if err != nil {
		t.Fatal(err)
	}
	var decoded report.Result
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, data)
	}
	if decoded.OK || len(decoded.Violations) != 1 || len(decoded.Drift.Extra) != 1 {
		t.Fatalf("unexpected payload: %+v", decoded)
	}
	if !strings.Contains(string(data), `"import_path"`) {
		t.Fatalf("expected snake_case json tags:\n%s", data)
	}
}

func TestFormatSARIF_Marshal(t *testing.T) {
	data, err := report.MarshalSARIF(sampleResult())
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, data)
	}
	if doc["version"] != "2.1.0" {
		t.Fatalf("version=%v", doc["version"])
	}
	runs, ok := doc["runs"].([]any)
	if !ok || len(runs) != 1 {
		t.Fatalf("runs=%v", doc["runs"])
	}
	run := runs[0].(map[string]any)
	results := run["results"].([]any)
	if len(results) < 2 {
		t.Fatalf("expected results, got %d", len(results))
	}
	body := string(data)
	if !strings.Contains(body, report.RuleLayerIsolation) || !strings.Contains(body, report.RuleAntiDrift) {
		t.Fatalf("missing rules:\n%s", body)
	}
}

func TestFormatText_UsesANSI(t *testing.T) {
	v := sampleResult().Violations[0]
	out := report.FormatTextViolation(v)
	if !strings.Contains(out, "\033[") {
		t.Fatalf("expected ANSI codes: %q", out)
	}
	diff := report.FormatTextDiff(sampleResult().Drift)
	if !strings.Contains(diff, "\033[") {
		t.Fatalf("expected ANSI in diff: %q", diff)
	}
}

func TestFormatLLM_LayerIsolation(t *testing.T) {
	v := sampleResult().Violations[0]
	out := report.FormatLLMViolation(v)
	for _, w := range []string{
		"RULE_VIOLATION: " + report.RuleLayerIsolation,
		"FILE: internal/domain/doc.go:2",
		"DETAILS:",
		"ACTION_REQUIRED:",
	} {
		if !strings.Contains(out, w) {
			t.Fatalf("missing %q in:\n%s", w, out)
		}
	}
}

func TestFormatLLM_AntiDriftExtra(t *testing.T) {
	out := report.FormatLLMDiff(antidrift.Diff{Extra: []string{"extra.txt"}})
	if !strings.Contains(out, "FILE: extra.txt") {
		t.Fatalf("missing file:\n%s", out)
	}
	if strings.Contains(out, "add it to ContractFiles") {
		t.Fatalf("stale Extra action text:\n%s", out)
	}
}

func TestFormatPlainDiff(t *testing.T) {
	out := report.FormatPlainDiff(sampleResult().Drift)
	if strings.Contains(out, "\033[") {
		t.Fatalf("plain diff must not use ANSI: %q", out)
	}
}
