package fitness_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fow830/ratchet/pkg/fitness"
	"github.com/fow830/ratchet/pkg/tokens"
)

func TestDetectCycles(t *testing.T) {
	root := t.TempDir()
	writeGoFile(t, root, "internal/a/a.go", `package a
import _ "example.com/c/internal/b"`)
	writeGoFile(t, root, "internal/b/b.go", `package b
import _ "example.com/c/internal/a"`)
	_ = os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/c\n\ngo 1.22\n"), 0o644)

	cycles, err := fitness.DetectCycles(context.Background(), root, "example.com/c")
	if err != nil {
		t.Fatal(err)
	}
	if len(cycles) == 0 {
		t.Fatal("expected cycle a↔b")
	}
}

func TestDetectCyclesNone(t *testing.T) {
	root := t.TempDir()
	writeGoFile(t, root, "internal/domain/d.go", `package domain`)
	writeGoFile(t, root, "internal/usecase/u.go", `package usecase
import _ "example.com/ok/internal/domain"`)
	_ = os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/ok\n\ngo 1.22\n"), 0o644)

	cycles, err := fitness.DetectCycles(context.Background(), root, "example.com/ok")
	if err != nil {
		t.Fatal(err)
	}
	if len(cycles) != 0 {
		t.Fatalf("unexpected cycles: %v", cycles)
	}
}

func TestExternalModulePolicy(t *testing.T) {
	root := t.TempDir()
	writeGoFile(t, root, "internal/domain/d.go", `package domain
import _ "github.com/evil/forbidden"`)
	_ = os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/x\n\ngo 1.22\n"), 0o644)
	cfg := tokens.DefaultConfig("example.com/x")
	cfg.AllowedExternal = []string{"golang.org/x/"}
	v, err := fitness.NewAnalyzer(cfg).CheckExternal(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) == 0 {
		t.Fatal("expected external violation")
	}
}

func TestTestFileImportRules(t *testing.T) {
	root := t.TempDir()
	writeGoFile(t, root, "internal/domain/d_test.go", `package domain_test
import _ "example.com/t/internal/delivery"`)
	_ = os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/t\n\ngo 1.22\n"), 0o644)
	cfg := tokens.DefaultConfig("example.com/t")
	cfg.TestForbiddenEdges = []tokens.ForbiddenEdge{{From: tokens.LayerDomain, To: tokens.LayerDelivery}}
	v, err := fitness.NewAnalyzer(cfg).CheckTestImports(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) == 0 {
		t.Fatal("expected test import violation")
	}
}

func writeGoFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
