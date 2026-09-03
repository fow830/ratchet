package fitness_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fow830/ratchet/pkg/fitness"
	"github.com/fow830/ratchet/pkg/tokens"
)

func TestForbiddenEdgeViolation(t *testing.T) {
	root := t.TempDir()
	writeModule(t, root, "example.com/app")
	writeGo(t, root, "internal/domain/x.go", `package domain
import _ "example.com/app/internal/delivery"`)
	cfg := tokens.PresetConfig(tokens.PresetClean, "example.com/app")
	cfg.ForbiddenEdges = []tokens.ForbiddenEdge{{From: tokens.LayerDomain, To: tokens.LayerDelivery}}
	violations, err := fitness.NewAnalyzer(cfg).AnalyzeAll(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) == 0 {
		t.Fatal("expected forbidden edge violation")
	}
}

func TestLayerPathsMissing(t *testing.T) {
	root := t.TempDir()
	writeModule(t, root, "m")
	cfg := tokens.DefaultConfig("m")
	cfg.LayerPaths = []string{"internal/domain"}
	violations, err := fitness.NewAnalyzer(cfg).CheckLayerPaths(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 {
		t.Fatalf("violations: %#v", violations)
	}
}

func writeModule(t *testing.T, root, module string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module "+module+"\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeGo(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
