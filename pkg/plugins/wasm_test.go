package plugins_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fow830/ratchet/pkg/plugins"
)

// Minimal wasm: (func (export "run") (result i32) i32.const 0)
var wasmOK = []byte{
	0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0x01, 0x05, 0x01, 0x60, 0x00, 0x01, 0x7f,
	0x03, 0x02, 0x01, 0x00, 0x07, 0x07, 0x01, 0x03, 0x72, 0x75, 0x6e, 0x00, 0x00, 0x0a, 0x06, 0x01,
	0x04, 0x00, 0x41, 0x00, 0x0b,
}

// Minimal wasm: run returns 1 (violation).
var wasmFail = []byte{
	0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0x01, 0x05, 0x01, 0x60, 0x00, 0x01, 0x7f,
	0x03, 0x02, 0x01, 0x00, 0x07, 0x07, 0x01, 0x03, 0x72, 0x75, 0x6e, 0x00, 0x00, 0x0a, 0x06, 0x01,
	0x04, 0x00, 0x41, 0x01, 0x0b,
}

func TestRunPluginOK(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "ok.wasm")
	if err := os.WriteFile(path, wasmOK, 0o644); err != nil {
		t.Fatal(err)
	}
	eng, err := plugins.NewEngine(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close(context.Background())
	res, err := eng.RunFile(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if res.Code != 0 {
		t.Fatalf("code=%d", res.Code)
	}
}

func TestRunPluginViolation(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "fail.wasm")
	if err := os.WriteFile(path, wasmFail, 0o644); err != nil {
		t.Fatal(err)
	}
	eng, err := plugins.NewEngine(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close(context.Background())
	res, err := eng.RunFile(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if res.Code == 0 {
		t.Fatal("expected non-zero violation code")
	}
}

func TestRunAllPlugins(t *testing.T) {
	root := t.TempDir()
	okPath := filepath.Join(root, "ok.wasm")
	if err := os.WriteFile(okPath, wasmOK, 0o644); err != nil {
		t.Fatal(err)
	}
	eng, err := plugins.NewEngine(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close(context.Background())
	if err := plugins.RunAll(context.Background(), eng, []string{okPath}); err != nil {
		t.Fatal(err)
	}
}
