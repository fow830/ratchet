package gha_test

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gha "github.com/fow830/ratchet/pkg/github"
	"github.com/fow830/ratchet/pkg/report"
	"github.com/fow830/ratchet/pkg/tokens"
)

func TestWriteWorkflow_CreatesRatchetYML(t *testing.T) {
	root := t.TempDir()
	path, err := gha.WriteWorkflow(root)
	if err != nil {
		t.Fatalf("WriteWorkflow: %v", err)
	}
	wantPath := filepath.Join(root, filepath.FromSlash(gha.WorkflowRel))
	if path != wantPath {
		t.Fatalf("path = %q, want %q", path, wantPath)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	body := string(data)
	for _, needle := range []string{
		"push:",
		"pull_request:",
		"branches: [" + gha.DefaultBranch + "]",
		"tags: ['v*.*.*']",
		"go test ./...",
		"go vet ./...",
		gha.GoreleaserAction,
		gha.GoreleaserPin,
		tokens.ToolName + " check --profile=" + tokens.ProfileStrict + " --format=" + report.FormatLLM,
		tokens.BinaryRel,
		gha.CosignInstallerAction,
		gha.UploadArtifactAction,
		gha.UploadSARIFAction,
		gha.CIRunnerOS,
		gha.CIScheduleCron,
		tokens.InstallStaticcheck,
		tokens.InstallGovulncheck,
		tokens.FitnessPkgRel,
		"go-version-file: " + tokens.GoModFileName,
		"observe --format=" + report.FormatJSON,
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("workflow missing %q:\n%s", needle, body)
		}
	}
}

func TestWriteWorkflow_UsesMockFS(t *testing.T) {
	mem := &memFS{dirs: map[string]bool{}, files: map[string][]byte{}}
	client := &gha.Client{FS: mem}
	path, err := client.WriteWorkflow("/repo")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, gha.WorkflowRel) {
		t.Fatalf("path=%s", path)
	}
	if _, ok := mem.files[path]; !ok {
		t.Fatalf("file not written: %v", mem.files)
	}
}

func TestProtectMain_UsesMockRunner(t *testing.T) {
	r := &mockRunner{path: "/bin/gh"}
	client := &gha.Client{Runner: r}
	var stdout, stderr bytes.Buffer
	if err := client.ProtectMain("acme", "widgets", &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if r.lastName != "/bin/gh" {
		t.Fatalf("name=%s", r.lastName)
	}
	joined := strings.Join(r.lastArgs, " ")
	if !strings.Contains(joined, "repos/acme/widgets/branches/"+gha.DefaultBranch+"/protection") {
		t.Fatalf("args=%v", r.lastArgs)
	}
}

func TestProtectMainCommand_IsDeterministic(t *testing.T) {
	cmd := gha.ProtectMainCommand("acme", "widgets")
	if !strings.Contains(cmd, gha.GhBinary+" api") {
		t.Fatalf("expected %s api command, got %q", gha.GhBinary, cmd)
	}
	if !strings.Contains(cmd, "repos/acme/widgets/branches/"+gha.DefaultBranch+"/protection") {
		t.Fatalf("expected protection endpoint: %q", cmd)
	}
}

func TestIsProtectionUnavailable(t *testing.T) {
	if !gha.IsProtectionUnavailable(`Upgrade to GitHub Pro or make this repository public`) {
		t.Fatal("expected true")
	}
	if gha.IsProtectionUnavailable("ok") {
		t.Fatal("expected false")
	}
}

func TestWorkflowYAML_MatchesCommittedFile(t *testing.T) {
	root, err := repoRoot(t)
	if err != nil {
		t.Fatal(err)
	}
	committed, err := os.ReadFile(filepath.Join(root, gha.WorkflowRel))
	if err != nil {
		t.Fatal(err)
	}
	want := gha.WorkflowYAML()
	got := string(committed)
	if got != want && got != want+"\n" && strings.TrimSuffix(got, "\n") != strings.TrimSuffix(want, "\n") {
		t.Fatalf("committed %s drifts from WorkflowYAML()\n--- want ---\n%s\n--- got ---\n%s", gha.WorkflowRel, want, got)
	}
}

func repoRoot(t *testing.T) (string, error) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", wd)
		}
		dir = parent
	}
}

func TestParseOwnerRepo(t *testing.T) {
	cases := []struct {
		remote    string
		wantOwner string
		wantRepo  string
	}{
		{"https://github.com/acme/widgets.git", "acme", "widgets"},
		{"git@github.com:acme/widgets.git", "acme", "widgets"},
	}
	for _, tc := range cases {
		owner, repo, err := gha.ParseOwnerRepo(tc.remote)
		if err != nil {
			t.Fatalf("%s: %v", tc.remote, err)
		}
		if owner != tc.wantOwner || repo != tc.wantRepo {
			t.Fatalf("%s: got %s/%s", tc.remote, owner, repo)
		}
	}
}

type memFS struct {
	dirs  map[string]bool
	files map[string][]byte
}

func (m *memFS) MkdirAll(path string, _ fs.FileMode) error {
	m.dirs[path] = true
	return nil
}

func (m *memFS) WriteFile(name string, data []byte, _ fs.FileMode) error {
	m.files[name] = append([]byte(nil), data...)
	return nil
}

type mockRunner struct {
	path     string
	lastName string
	lastArgs []string
}

func (m *mockRunner) LookPath(file string) (string, error) {
	if m.path == "" {
		return "", os.ErrNotExist
	}
	return m.path, nil
}

func (m *mockRunner) Run(name string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	m.lastName = name
	m.lastArgs = append([]string(nil), args...)
	if stdin != nil {
		_, _ = io.Copy(io.Discard, stdin)
	}
	return nil
}
