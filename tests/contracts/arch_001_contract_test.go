package contracts_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fow830/ratchet/pkg/fitness"
	"github.com/fow830/ratchet/pkg/tokens"
)

// ARCH-001: dogfood — ratchet self SSOT is valid and architecture is clean.
func TestContract_Architecture_Dogfood(t *testing.T) {
	root := moduleRoot(t)
	cfg, err := tokens.Load(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if err := tokens.Validate(cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Module != tokens.ModulePath {
		t.Fatalf("module=%q want %q", cfg.Module, tokens.ModulePath)
	}
	v, err := fitness.NewAnalyzer(cfg).AnalyzeAll(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 0 {
		t.Fatalf("architecture violations: %+v", v)
	}
	cycles, err := fitness.DetectCycles(context.Background(), root, cfg.Module)
	if err != nil {
		t.Fatal(err)
	}
	if len(cycles) != 0 {
		t.Fatalf("cycles: %v", cycles)
	}
	for _, must := range []string{tokens.ConfigFileName, tokens.LockFileName, tokens.CursorRules} {
		if _, err := os.Stat(filepath.Join(root, must)); err != nil {
			t.Fatalf("missing %s: %v", must, err)
		}
	}
}
