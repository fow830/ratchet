package gha_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fow830/ratchet/pkg/github"
)

func TestWriteWorkflow_CreatesRatchetYML(t *testing.T) {
	root := t.TempDir()
	path, err := gha.WriteWorkflow(root)
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
	cmd := gha.ProtectMainCommand("acme", "widgets")
	if !strings.Contains(cmd, "gh api") {
		t.Fatalf("expected gh api command, got %q", cmd)
	}
	if !strings.Contains(cmd, "repos/acme/widgets/branches/main/protection") {
		t.Fatalf("expected protection endpoint: %q", cmd)
	}
	if !strings.Contains(cmd, gha.StatusCheckName) {
		t.Fatalf("expected status check context: %q", cmd)
	}
	body := gha.ProtectionBody()
	if !strings.Contains(body, `"contexts":["`+gha.StatusCheckName+`"]`) {
		t.Fatalf("protection body missing context: %s", body)
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
