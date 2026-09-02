package tokens_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fow830/ratchet/pkg/tokens"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	root := t.TempDir()
	cfg := tokens.DefaultConfig("example.com/app")
	if err := tokens.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := tokens.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Module != cfg.Module {
		t.Fatalf("module %q != %q", got.Module, cfg.Module)
	}
	if len(got.ContractFiles) != 2 {
		t.Fatalf("contract files: %#v", got.ContractFiles)
	}
}

func TestModuleFromGoMod(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/x\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mod, err := tokens.ModuleFromGoMod(root)
	if err != nil {
		t.Fatal(err)
	}
	if mod != "example.com/x" {
		t.Fatalf("got %q", mod)
	}
}
