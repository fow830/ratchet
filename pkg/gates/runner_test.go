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

func TestRunMinimalProfile(t *testing.T) {
	root := t.TempDir()
	writeRatchetRepo(t, root)
	res, err := gates.Run(context.Background(), gates.Options{
		Root:    root,
		Config:  tokens.DefaultConfig("example.com/r"),
		Profile: tokens.ProfileMinimal,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("result: %+v failures: %d", res, len(res.Failures))
	}
}

func writeRatchetRepo(t *testing.T, root string) {
	t.Helper()
	_ = os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/r\n\ngo 1.22\n"), 0o644)
	cfg := tokens.DefaultConfig("example.com/r")
	if err := tokens.Save(context.Background(), root, cfg); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(root, tokens.CursorRules), []byte("# rules\n"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "README.md"), []byte("# ok\n"), 0o644)
	if err := antidrift.New(root).Lock(context.Background(), cfg.ContractFiles); err != nil {
		t.Fatal(err)
	}
}
