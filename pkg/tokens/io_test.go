package tokens_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fow830/ratchet/pkg/tokens"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	cfg := tokens.DefaultConfig("example.com/app")
	if err := tokens.Save(ctx, root, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := tokens.Load(ctx, root)
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
	mod, err := tokens.ModuleFromGoMod(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if mod != "example.com/x" {
		t.Fatalf("got %q", mod)
	}
}

func TestDecode_InvalidJSON(t *testing.T) {
	_, err := tokens.Decode(context.Background(), strings.NewReader("{not-json"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDecode_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := tokens.Decode(ctx, strings.NewReader(`{"module":"x"}`))
	if err == nil {
		t.Fatal("expected canceled error")
	}
}

func TestLoadFile_StreamLargeConfig(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, tokens.ConfigFileName)
	var b bytes.Buffer
	b.WriteString(`{"module":"example.com/big","layers":{`)
	for i := 0; i < 200; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		_, _ = b.WriteString(`"/layer`)
		_, _ = b.WriteString(strings.Repeat("x", 32))
		_, _ = b.WriteString(`":"domain"`)
	}
	b.WriteString(`},"allowed_edges":{"domain":[]},"contract_files":[".cursorrules"]}`)
	if err := os.WriteFile(path, b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cfg, err := tokens.Load(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Module != "example.com/big" {
		t.Fatalf("got module %q", cfg.Module)
	}
}
