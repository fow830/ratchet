package report_test

import (
	"strings"
	"testing"

	"github.com/fow830/ratchet/pkg/antidrift"
	"github.com/fow830/ratchet/pkg/fitness"
	"github.com/fow830/ratchet/pkg/report"
)

func TestFormatLLM_LayerIsolation(t *testing.T) {
	v := fitness.Violation{
		File:          "internal/domain/doc.go",
		Line:          2,
		ImportPath:    "example.com/app/internal/usecase",
		ImporterPkg:   "example.com/app/internal/domain",
		ImporterLayer: "domain",
		ImportedLayer: "usecase",
	}
	out := report.FormatLLMViolation(v)
	want := []string{
		"RULE_VIOLATION: LayerIsolation",
		"FILE: internal/domain/doc.go:2",
		"DETAILS:",
		"ACTION_REQUIRED:",
	}
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Fatalf("missing %q in:\n%s", w, out)
		}
	}
	if strings.Contains(out, "please") || strings.Contains(out, "😊") {
		t.Fatal("LLM output must stay dry")
	}
}

func TestFormatLLM_AntiDriftChanged(t *testing.T) {
	diff := antidrift.Diff{
		Changed: []antidrift.ChangedFile{{
			Path:     "ratchet.json",
			Expected: "aaa",
			Actual:   "bbb",
		}},
	}
	out := report.FormatLLMDiff(diff)
	if !strings.Contains(out, "RULE_VIOLATION: AntiDrift") {
		t.Fatalf("missing AntiDrift rule:\n%s", out)
	}
	if !strings.Contains(out, "FILE: ratchet.json") {
		t.Fatalf("missing file:\n%s", out)
	}
	if !strings.Contains(out, "ACTION_REQUIRED:") {
		t.Fatalf("missing action:\n%s", out)
	}
}

func TestFormatHuman_Default(t *testing.T) {
	v := fitness.Violation{
		File:          "a.go",
		Line:          1,
		ImporterPkg:   "m/domain",
		ImporterLayer: "domain",
		ImportPath:    "m/usecase",
		ImportedLayer: "usecase",
	}
	out := report.FormatHumanViolation(v)
	if strings.Contains(out, "RULE_VIOLATION:") {
		t.Fatalf("human format must not use LLM schema:\n%s", out)
	}
}
