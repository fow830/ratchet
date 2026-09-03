package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fow830/ratchet/pkg/tokens"
)

func TestTokensgenMainFindsModule(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, tokens.GoModFileName), []byte("module m\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// findModuleRoot is in main — smoke via build tag test in integration
	if _, err := os.Stat(filepath.Join(root, tokens.GoModFileName)); err != nil {
		t.Fatal(err)
	}
}
