package hooks_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fow830/ratchet/pkg/hooks"
)

func TestInstall_WritesPreCommit(t *testing.T) {
	root := t.TempDir()
	gitDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	path, err := hooks.Install(root)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	want := filepath.Join(root, ".git", "hooks", "pre-commit")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "ratchet check --format=llm") {
		t.Fatalf("hook missing check invocation:\n%s", body)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("pre-commit must be executable, mode=%v", info.Mode())
	}
}

func TestInstall_RequiresGitDir(t *testing.T) {
	root := t.TempDir()
	if _, err := hooks.Install(root); err == nil {
		t.Fatal("expected error when .git is missing")
	}
}
