package doctor_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fow830/ratchet/pkg/antidrift"
	gha "github.com/fow830/ratchet/pkg/github"
	"github.com/fow830/ratchet/pkg/doctor"
	"github.com/fow830/ratchet/pkg/tokens"
)

func TestDoctorHealthy(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "go.mod"), []byte("module m\n\ngo 1.22\n"), 0o644)
	cfg := tokens.DefaultConfig("m")
	_ = tokens.Save(context.Background(), root, cfg)
	_ = os.WriteFile(filepath.Join(root, tokens.CursorRules), []byte("x"), 0o644)
	if err := antidrift.New(root).Lock(context.Background(), cfg.ContractFiles); err != nil {
		t.Fatal(err)
	}
	if _, err := gha.WriteWorkflow(root); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, tokens.GitDir), 0o755); err != nil {
		t.Fatal(err)
	}
	report, err := doctor.Run(context.Background(), root, cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	if !report.Healthy() {
		t.Fatalf("report: %+v", report.Checks)
	}
}
