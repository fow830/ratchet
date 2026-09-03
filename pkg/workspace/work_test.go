package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fow830/ratchet/pkg/workspace"
)

func TestDiscoverGoWorkModules(t *testing.T) {
	root := t.TempDir()
	gowork := `go 1.22

use (
	./services/a
	./services/b
)
`
	_ = os.WriteFile(filepath.Join(root, "go.work"), []byte(gowork), 0o644)
	mods, err := workspace.ModulesFromGoWork(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 2 {
		t.Fatalf("mods=%v", mods)
	}
}
