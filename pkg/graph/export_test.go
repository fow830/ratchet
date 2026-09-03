package graph_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fow830/ratchet/pkg/graph"
	"github.com/fow830/ratchet/pkg/tokens"
)

func TestMermaidExport(t *testing.T) {
	root := t.TempDir()
	writeMod(t, root)
	writeGo(t, root, "internal/domain/a.go", `package domain`)
	writeGo(t, root, "internal/usecase/b.go", `package usecase
import _ "example.com/x/internal/domain"`)
	cfg := tokens.DefaultConfig("example.com/x")
	out, err := graph.ExportMermaid(context.Background(), root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(out, "domain") || !contains(out, "usecase") {
		t.Fatalf("graph: %s", out)
	}
}

func writeMod(t *testing.T, root string) {
	t.Helper()
	_ = os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/x\n\ngo 1.22\n"), 0o644)
}

func writeGo(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, rel)
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, []byte(body), 0o644)
}

func contains(s, sub string) bool { return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub)) }

func indexOf(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
