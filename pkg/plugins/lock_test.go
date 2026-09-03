package plugins_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fow830/ratchet/pkg/plugins"
	"github.com/fow830/ratchet/pkg/tokens"
)

func TestPluginLockVerify(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "p.wasm")
	if err := os.WriteFile(path, []byte("wasm-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	refs := []tokens.PluginRef{{Path: "p.wasm", Name: "p"}}
	if err := plugins.LockPlugins(context.Background(), root, refs); err != nil {
		t.Fatal(err)
	}
	if err := plugins.VerifyLock(context.Background(), root, refs); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := plugins.VerifyLock(context.Background(), root, refs); err == nil {
		t.Fatal("expected drift")
	}
}
