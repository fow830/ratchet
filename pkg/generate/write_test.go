package generate_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fow830/ratchet/pkg/generate"
	"github.com/fow830/ratchet/pkg/tokens"
)

func TestWriteAll(t *testing.T) {
	root := t.TempDir()
	cfg := tokens.DefaultConfig("example.com/x")
	written, err := generate.WriteAll(context.Background(), root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(written) < 2 {
		t.Fatalf("written=%v", written)
	}
	if _, err := os.Stat(filepath.Join(root, tokens.EnvExampleRel)); err != nil {
		t.Fatal(err)
	}
}
