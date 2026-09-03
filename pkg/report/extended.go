package report

import (
	"github.com/fow830/ratchet/pkg/antidrift"
	"github.com/fow830/ratchet/pkg/docs"
	"github.com/fow830/ratchet/pkg/fitness"
	"github.com/fow830/ratchet/pkg/gates"
)

// ExtendedResult is the full superproduction check payload.
type ExtendedResult struct {
	Result
	Gates          gates.Result     `json:"gates"`
	DocsViolations []docs.Violation `json:"docs_violations"`
	Failures       []gates.Failure  `json:"failures"`
	Profile        string           `json:"profile"`
}

// FromGateResult builds ExtendedResult from gate runner output.
func FromGateResult(gr gates.Result) ExtendedResult {
	er := ExtendedResult{
		Result: Result{
			OK:         gr.OK,
			Violations: gr.Violations,
			Drift:      gr.Drift,
		},
		Gates:          gr,
		DocsViolations: gr.DocsViolations,
		Failures:       gr.Failures,
		Profile:        gr.Profile,
	}
	er.NormalizeExtended()
	return er
}

// NormalizeExtended fills nil slices on extended result.
func (r *ExtendedResult) NormalizeExtended() {
	r.Normalize()
	if r.DocsViolations == nil {
		r.DocsViolations = []docs.Violation{}
	}
	if r.Failures == nil {
		r.Failures = []gates.Failure{}
	}
	if r.Violations == nil {
		r.Violations = []fitness.Violation{}
	}
	if r.Drift.Changed == nil {
		r.Drift = antidrift.Diff{Changed: []antidrift.ChangedFile{}, Missing: []string{}, Extra: []string{}}
	}
}
