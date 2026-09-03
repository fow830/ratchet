package codegen_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fow830/ratchet/pkg/codegen"
	"github.com/fow830/ratchet/pkg/tokens"
)

func TestSQLCConfigured(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "sqlc.yaml")
	_ = os.WriteFile(path, []byte("version: 2\n"), 0o644)
	cfg := tokens.DefaultConfig("m")
	cfg.Codegen.SQLCPath = path
	if !codegen.IsConfigured(cfg, codegen.ToolSQLC) {
		t.Fatal("sqlc should be configured")
	}
}

func TestOpenAPINotConfigured(t *testing.T) {
	cfg := tokens.DefaultConfig("m")
	if codegen.IsConfigured(cfg, codegen.ToolOpenAPI) {
		t.Fatal("openapi should not be configured by default")
	}
}
