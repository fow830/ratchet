package breaking_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/fow830/ratchet/pkg/breaking"
	"github.com/fow830/ratchet/pkg/tokens"
)

func TestDiffAgainstGitHEAD(t *testing.T) {
	root := t.TempDir()
	run(t, root, "git", "init")
	run(t, root, "git", "config", "user.email", "t@t.t")
	run(t, root, "git", "config", "user.name", "t")
	cfg := tokens.DefaultConfig("m")
	data, _ := json.MarshalIndent(cfg, "", "  ")
	path := filepath.Join(root, tokens.ConfigFileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, root, "git", "add", tokens.ConfigFileName)
	run(t, root, "git", "commit", "-m", "init")

	cfg.AllowedEdges[tokens.LayerDelivery] = []string{tokens.LayerDomain}
	newData, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(path, newData, 0o644); err != nil {
		t.Fatal(err)
	}

	changes, err := breaking.DiffAgainstGit(context.Background(), root, path)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) == 0 {
		t.Fatal("expected breaking changes vs HEAD")
	}
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}
