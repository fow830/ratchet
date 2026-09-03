package workspace_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fow830/ratchet/pkg/tokens"
	"github.com/fow830/ratchet/pkg/workspace"
)

func TestDiscoverSingleModule(t *testing.T) {
	root := t.TempDir()
	writeMod(t, root, "example.com/a")
	cfg := tokens.DefaultConfig("example.com/a")
	if err := tokens.Save(context.Background(), root, cfg); err != nil {
		t.Fatal(err)
	}
	mods, err := workspace.Discover(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0].Module != "example.com/a" {
		t.Fatalf("mods=%+v", mods)
	}
}

func TestDiscoverGoWork(t *testing.T) {
	root := t.TempDir()
	writeMod(t, root, "example.com/root")
	a := filepath.Join(root, "services", "a")
	b := filepath.Join(root, "services", "b")
	writeMod(t, a, "example.com/a")
	writeMod(t, b, "example.com/b")
	_ = tokens.Save(context.Background(), a, tokens.DefaultConfig("example.com/a"))
	// services/b has go.mod but no ratchet.json — skipped
	gowork := "go 1.22\n\nuse (\n\t./services/a\n\t./services/b\n)\n"
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte(gowork), 0o644); err != nil {
		t.Fatal(err)
	}
	mods, err := workspace.Discover(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 {
		t.Fatalf("mods=%+v", mods)
	}
}

func writeMod(t *testing.T, root, mod string) {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module "+mod+"\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}
