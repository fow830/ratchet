// Package gha generates CI workflows and branch-protection commands.
package gha

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fow830/ratchet/pkg/report"
	"github.com/fow830/ratchet/pkg/tokens"
)

// StatusCheckName is the GitHub Actions job name and required check context.
const StatusCheckName = tokens.ToolName

// WorkflowRel is the generated workflow path relative to repo root.
const WorkflowRel = ".github/workflows/" + tokens.ToolName + ".yml"

// CI pin tokens (single place for workflow template + tests).
const (
	GoreleaserAction = "goreleaser/goreleaser-action@v6"
	GoreleaserPin    = "~> v2.12"
	CheckoutAction   = "actions/checkout@v4"
	SetupGoAction    = "actions/setup-go@v5"
)

// FileSystem abstracts disk writes for workflow generation tests.
type FileSystem interface {
	MkdirAll(path string, perm fs.FileMode) error
	WriteFile(name string, data []byte, perm fs.FileMode) error
}

// OSFileSystem is the default real-disk implementation.
type OSFileSystem struct{}

func (OSFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}
func (OSFileSystem) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(name, data, perm)
}

// Runner abstracts external process execution (e.g. gh api).
type Runner interface {
	LookPath(file string) (string, error)
	Run(name string, args []string, stdin io.Reader, stdout, stderr io.Writer) error
}

// ExecRunner uses os/exec.
type ExecRunner struct{}

func (ExecRunner) LookPath(file string) (string, error) { return exec.LookPath(file) }

func (ExecRunner) Run(name string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	c := exec.Command(name, args...)
	c.Stdin = stdin
	c.Stdout = stdout
	c.Stderr = stderr
	return c.Run()
}

// Client bundles injectable FS and command runner.
type Client struct {
	FS     FileSystem
	Runner Runner
}

// NewClient returns a Client with OS defaults.
func NewClient() *Client {
	return &Client{FS: OSFileSystem{}, Runner: ExecRunner{}}
}

func (c *Client) fs() FileSystem {
	if c != nil && c.FS != nil {
		return c.FS
	}
	return OSFileSystem{}
}

func (c *Client) runner() Runner {
	if c != nil && c.Runner != nil {
		return c.Runner
	}
	return ExecRunner{}
}

// WriteWorkflow writes .github/workflows/ratchet.yml under root.
func WriteWorkflow(root string) (string, error) {
	return NewClient().WriteWorkflow(root)
}

// WriteWorkflow writes the CI workflow file.
func (c *Client) WriteWorkflow(root string) (string, error) {
	path := filepath.Join(root, filepath.FromSlash(WorkflowRel))
	if err := c.fs().MkdirAll(filepath.Dir(path), tokens.FileModeDir); err != nil {
		return "", fmt.Errorf("mkdir workflows: %w", err)
	}
	if err := c.fs().WriteFile(path, []byte(WorkflowYAML()), tokens.FileModeFile); err != nil {
		return "", fmt.Errorf("write workflow: %w", err)
	}
	return path, nil
}

// WorkflowYAML is the committed CI workflow template.
func WorkflowYAML() string {
	return fmt.Sprintf(`name: %s

on:
  push:
    branches: [main]
    tags: ['v*.*.*']
  pull_request:
    branches: [main]

jobs:
  check:
    name: %s
    runs-on: ubuntu-latest
    steps:
      - uses: %s

      - uses: %s
        with:
          go-version-file: go.mod

      - name: Test
        run: go test ./...

      - name: Vet
        run: go vet ./...

      - name: Build %s
        run: go build -o %s ./%s

      - name: Architecture check
        run: ./%s check --format=%s

  release:
    name: release
    needs: check
    if: startsWith(github.ref, 'refs/tags/v')
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: %s
        with:
          fetch-depth: 0

      - uses: %s
        with:
          go-version-file: go.mod

      - name: Test
        run: go test ./...

      - name: Vet
        run: go vet ./...

      - uses: %s
        with:
          distribution: goreleaser
          version: '%s'
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
`, StatusCheckName, StatusCheckName, CheckoutAction, SetupGoAction,
		tokens.ToolName, tokens.BinaryRel, tokens.CmdRel,
		tokens.BinaryRel, report.FormatLLM,
		CheckoutAction, SetupGoAction, GoreleaserAction, GoreleaserPin)
}

// ProtectMain enables branch protection via gh api using the injectable runner.
func (c *Client) ProtectMain(owner, repo string, stdout, stderr io.Writer) error {
	r := c.runner()
	ghPath, err := r.LookPath("gh")
	if err != nil {
		return fmt.Errorf("protect-main: gh not installed")
	}
	args, body := ProtectMainArgs(owner, repo)
	return r.Run(ghPath, args, bytes.NewBufferString(body), stdout, stderr)
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
