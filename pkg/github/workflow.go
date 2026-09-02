// Package github generates CI workflows and branch-protection commands.
package github

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fow830/ratchet/pkg/report"
)

// StatusCheckName is the GitHub Actions job name and required check context.
const StatusCheckName = "ratchet"

// WorkflowRel is the generated workflow path relative to repo root.
const WorkflowRel = ".github/workflows/ratchet.yml"

// WriteWorkflow writes .github/workflows/ratchet.yml under root.
func WriteWorkflow(root string) (string, error) {
	path := filepath.Join(root, filepath.FromSlash(WorkflowRel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("mkdir workflows: %w", err)
	}
	if err := os.WriteFile(path, []byte(workflowYAML()), 0o644); err != nil {
		return "", fmt.Errorf("write workflow: %w", err)
	}
	return path, nil
}

func workflowYAML() string {
	return fmt.Sprintf(`name: %s

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  check:
    name: %s
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Test
        run: go test ./...

      - name: Build ratchet
        run: go build -o bin/ratchet ./cmd/ratchet

      - name: Architecture check
        run: ./bin/ratchet check --format=%s
`, StatusCheckName, StatusCheckName, report.FormatLLM)
}

// ProtectionBody is the JSON body for PUT .../branches/main/protection.
func ProtectionBody() string {
	return fmt.Sprintf(
		`{"required_status_checks":{"strict":true,"contexts":[%q]},"enforce_admins":true,"required_pull_request_reviews":null,"restrictions":null}`,
		StatusCheckName,
	)
}

// ProtectMainArgs returns gh argv and JSON body for PUT branch protection.
func ProtectMainArgs(owner, repo string) (args []string, body string) {
	body = ProtectionBody()
	args = []string{
		"api",
		"-X", "PUT",
		fmt.Sprintf("repos/%s/%s/branches/main/protection", owner, repo),
		"-H", "Accept: application/vnd.github+json",
		"--input", "-",
	}
	return args, body
}

// ProtectMainCommand returns a copy-pasteable shell command for branch protection.
func ProtectMainCommand(owner, repo string) string {
	_, body := ProtectMainArgs(owner, repo)
	return fmt.Sprintf(
		`printf '%%s' '%s' | gh api -X PUT repos/%s/%s/branches/main/protection -H "Accept: application/vnd.github+json" --input -`,
		body, owner, repo,
	)
}

// ParseOwnerRepo extracts owner/repo from a github.com remote URL.
func ParseOwnerRepo(remote string) (owner, repo string, err error) {
	remote = strings.TrimSpace(remote)
	remote = strings.TrimSuffix(remote, ".git")
	switch {
	case strings.HasPrefix(remote, "git@github.com:"):
		parts := strings.SplitN(strings.TrimPrefix(remote, "git@github.com:"), "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("invalid ssh remote: %s", remote)
		}
		return parts[0], parts[1], nil
	case strings.Contains(remote, "github.com/"):
		idx := strings.Index(remote, "github.com/")
		rest := remote[idx+len("github.com/"):]
		parts := strings.SplitN(rest, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("invalid https remote: %s", remote)
		}
		return parts[0], parts[1], nil
	default:
		return "", "", fmt.Errorf("unsupported remote: %s", remote)
	}
}

// IsProtectionUnavailable reports GitHub plan/visibility errors (403 Pro / public required).
func IsProtectionUnavailable(stderr string) bool {
	s := strings.ToLower(stderr)
	return strings.Contains(s, "upgrade to github pro") ||
		strings.Contains(s, "make this repository public")
}
