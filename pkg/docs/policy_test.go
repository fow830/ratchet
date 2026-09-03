package docs_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fow830/ratchet/pkg/docs"
	"github.com/fow830/ratchet/pkg/tokens"
)

func TestDocsPolicyViolation(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "RANDOM.md"), []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := tokens.DefaultConfig("m")
	cfg.AllowedProseDocs = []string{"README.md"}
	violations, err := docs.Check(context.Background(), root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 {
		t.Fatalf("got %#v", violations)
	}
}

func TestDocsPolicyAllowed(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := tokens.DefaultConfig("m")
	violations, err := docs.Check(context.Background(), root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("got %#v", violations)
	}
}
