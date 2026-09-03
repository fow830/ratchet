package report_test

import (
	"testing"

	"github.com/fow830/ratchet/pkg/antidrift"
	"github.com/fow830/ratchet/pkg/fitness"
	"github.com/fow830/ratchet/pkg/report"
)

func TestValidateSARIF(t *testing.T) {
	data, err := report.MarshalSARIF(report.Result{
		OK:         true,
		Violations: []fitness.Violation{},
		Drift:      antidrift.Diff{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := report.ValidateSARIF(data); err != nil {
		t.Fatal(err)
	}
	if err := report.ValidateSARIF([]byte(`{}`)); err == nil {
		t.Fatal("expected invalid")
	}
}
