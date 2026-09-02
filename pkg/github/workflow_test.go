package github_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	githubci "github.com/fow830/ratchet/pkg/github"
)

func TestWriteWorkflow_CreatesRatchetYML(t *testing.T) {
	root := t.TempDir()
	path, err := githubci.WriteWorkflow(root)
	if err != nil {
		t.Fatalf("WriteWorkflow: %v", err)
	}
	wantPath := filepath.Join(root, ".github", "workflows", "ratchet.yml")
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
		"branches: [main]",
		"go test ./...",
		"ratchet check --format=llm",
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("workflow missing %q:\n%s", needle, body)
		}
	}
}

func TestProtectMainCommand_IsDeterministic(t *testing.T) {
	cmd := githubci.ProtectMainCommand("fow830", "ratchet")
	if !strings.Contains(cmd, "gh api") {
		t.Fatalf("expected gh api command, got %q", cmd)
	}
	if !strings.Contains(cmd, "repos/fow830/ratchet/branches/main/protection") {
		t.Fatalf("expected protection endpoint: %q", cmd)
	}
	if !strings.Contains(cmd, `"contexts":["ratchet"]`) {
		t.Fatalf("expected status check context: %q", cmd)
	}
}

func TestParseOwnerRepo(t *testing.T) {
	owner, repo, err := githubci.ParseOwnerRepo("https://github.com/fow830/ratchet.git")
	if err != nil {
		t.Fatal(err)
	}
	if owner != "fow830" || repo != "ratchet" {
		t.Fatalf("got %s/%s", owner, repo)
	}
}
