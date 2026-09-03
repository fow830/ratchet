package report_test

import (
	"strings"
	"testing"

	"github.com/fow830/ratchet/pkg/fitness"
	"github.com/fow830/ratchet/pkg/gates"
	"github.com/fow830/ratchet/pkg/report"
)

func TestExplainResult(t *testing.T) {
	er := report.ExtendedResult{
		Result: report.Result{
			OK: false,
			Violations: []fitness.Violation{{
				File: "x.go", ImporterLayer: "domain", ImportedLayer: "delivery", ImportPath: "p",
			}},
		},
		Failures: []gates.Failure{{Gate: "cycles", Message: "a->b->a"}},
		Profile:  "standard",
	}
	out := report.ExplainResult(er)
	if !strings.Contains(out, "arch:") || !strings.Contains(out, "cycles") {
		t.Fatalf("explain: %s", out)
	}
}
