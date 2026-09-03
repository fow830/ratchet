package gates_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fow830/ratchet/pkg/antidrift"
	"github.com/fow830/ratchet/pkg/gates"
	"github.com/fow830/ratchet/pkg/tokens"
)

func TestRunWorkspace(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "services", "a")
	_ = os.MkdirAll(a, 0o755)
	_ = os.WriteFile(filepath.Join(a, "go.mod"), []byte("module example.com/a\n\ngo 1.22\n"), 0o644)
	cfg := tokens.DefaultConfig("example.com/a")
	_ = tokens.Save(context.Background(), a, cfg)
	_ = os.WriteFile(filepath.Join(a, tokens.CursorRules), []byte("# r\n"), 0o644)
	_ = antidrift.New(a).Lock(context.Background(), cfg.ContractFiles)
	gowork := "go 1.22\n\nuse (\n\t./services/a\n)\n"
	_ = os.WriteFile(filepath.Join(root, "go.work"), []byte(gowork), 0o644)

	res, err := gates.RunWorkspace(context.Background(), gates.WorkspaceOptions{
		Root:    root,
		Profile: tokens.ProfileMinimal,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || !res[0].OK {
		t.Fatalf("res=%+v", res)
	}
}
