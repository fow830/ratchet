package antidrift_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fow830/ratchet/pkg/antidrift"
	"github.com/fow830/ratchet/pkg/tokens"
)

func TestVerifyRenderLock(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "out.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := tokens.DefaultConfig("m")
	cfg.ContractFiles = []string{"out.txt"}
	cfg.ContractLocks = []tokens.ContractLock{{
		Path:          "out.txt",
		Mode:          tokens.LockModeRender,
		RenderPackage: "example.com/render",
		RenderFunc:    "RenderOut",
	}}
	eng := antidrift.New(root)
	eng.Renderers = map[string]antidrift.RenderFunc{
		"example.com/render.RenderOut": func() ([]byte, error) { return []byte("hello"), nil },
	}
	if err := eng.LockAll(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	diff, err := eng.VerifyAll(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !diff.OK() {
		t.Fatalf("diff: %s", diff)
	}
}

func TestVerifyRenderDrift(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "out.txt")
	if err := os.WriteFile(path, []byte("drift"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := tokens.DefaultConfig("m")
	cfg.ContractFiles = []string{"out.txt"}
	cfg.ContractLocks = []tokens.ContractLock{{
		Path:          "out.txt",
		Mode:          tokens.LockModeRender,
		RenderPackage: "example.com/render",
		RenderFunc:    "RenderOut",
	}}
	eng := antidrift.New(root)
	eng.Renderers = map[string]antidrift.RenderFunc{
		"example.com/render.RenderOut": func() ([]byte, error) { return []byte("expected"), nil },
	}
	if err := eng.LockAll(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	diff, err := eng.VerifyAll(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if diff.OK() {
		t.Fatal("expected render drift")
	}
}
