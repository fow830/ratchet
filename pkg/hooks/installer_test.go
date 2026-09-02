package hooks_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fow830/ratchet/pkg/hooks"
	"github.com/fow830/ratchet/pkg/report"
	"github.com/fow830/ratchet/pkg/tokens"
)

func TestInstall_WritesPreCommit(t *testing.T) {
	root := t.TempDir()
	gitDir := filepath.Join(root, tokens.GitDir, "hooks")
	if err := os.MkdirAll(gitDir, tokens.FileModeDir); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	path, err := hooks.Install(root)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	want := filepath.Join(root, filepath.FromSlash(tokens.PreCommitRel))
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, tokens.ToolName+" check --format="+report.FormatLLM) {
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

func TestInstall_UsesMockFS(t *testing.T) {
	mem := &memFS{
		info: map[string]fs.FileInfo{
			"/repo/.git": fakeDir{},
		},
		files: map[string][]byte{},
	}
	inst := &hooks.Installer{FS: mem}
	path, err := inst.Install("/repo")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/repo/.git/hooks/pre-commit" {
		t.Fatalf("path=%s", path)
	}
	if _, ok := mem.files[path]; !ok {
		t.Fatal("hook not written")
	}
}

type memFS struct {
	info  map[string]fs.FileInfo
	files map[string][]byte
}

func (m *memFS) Stat(name string) (fs.FileInfo, error) {
	info, ok := m.info[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return info, nil
}

func (m *memFS) MkdirAll(path string, _ fs.FileMode) error {
	return nil
}

func (m *memFS) WriteFile(name string, data []byte, _ fs.FileMode) error {
	m.files[name] = append([]byte(nil), data...)
	return nil
}

type fakeDir struct{}

func (fakeDir) Name() string       { return tokens.GitDir }
func (fakeDir) Size() int64        { return 0 }
func (fakeDir) Mode() fs.FileMode  { return fs.ModeDir | 0o755 }
func (fakeDir) ModTime() time.Time { return time.Time{} }
func (fakeDir) IsDir() bool        { return true }
func (fakeDir) Sys() any           { return nil }
