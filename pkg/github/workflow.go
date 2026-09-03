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

// DefaultBranch is the protected / CI target branch.
const DefaultBranch = tokens.DefaultBranch

// Workflow / GitHub identity tokens.
const (
	WorkflowDir     = tokens.DirGitHub + "/workflows/"
	WorkflowRel     = WorkflowDir + tokens.ToolName + ".yml"
	GitHubHost      = "github.com"
	GitHubSSHPrefix = "git@" + GitHubHost + ":"
	GitHubHTTPSMark = GitHubHost + "/"
	GhBinary        = "gh"
)

// CI pin tokens (single place for workflow template + tests).
const (
	GoreleaserAction      = "goreleaser/goreleaser-action@v6"
	GoreleaserPin         = "~> v2.12"
	GoreleaserDist        = "goreleaser"
	CheckoutAction        = "actions/checkout@v4"
	SetupGoAction         = "actions/setup-go@v5"
	UploadArtifactAction  = "actions/upload-artifact@v4"
	UploadSARIFAction     = "github/codeql-action/upload-sarif@v3"
	CosignInstallerAction = "sigstore/cosign-installer@v3"
	CIRunnerOS            = "ubuntu-latest"
	CIScheduleCron        = "0 3 * * *"
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

// WorkflowYAML is the committed superproduction CI workflow template.
func WorkflowYAML() string {
	return fmt.Sprintf(`name: %s

on:
  push:
    branches: [%s]
    tags: ['v*.*.*']
  pull_request:
    branches: [%s]
  schedule:
    - cron: '%s'

jobs:
  check:
    name: %s
    runs-on: %s
    steps:
      - uses: %s

      - uses: %s
        with:
          go-version-file: %s
          cache: true

      - name: Install analysis tools
        run: |
          go install %s
          go install %s

      - name: Test
        run: go test ./... -count=1

      - name: Vet
        run: go vet ./...

      - name: Build %s
        run: go build -o %s ./%s

      - name: Architecture check (strict)
        run: ./%s check --profile=%s --format=%s

      - name: SARIF
        run: ./%s check --profile=%s --format=%s > %s.sarif

      - name: Validate SARIF
        run: ./%s validate-sarif %s.sarif

      - name: Upload SARIF
        uses: %s
        with:
          sarif_file: %s.sarif
        continue-on-error: true
        # Upload needs GitHub Advanced Security / security-events; generation+validate are hard gates.

      - name: CPU profile artifact
        run: go test ./%s -cpuprofile=%s -count=1

      - uses: %s
        with:
          name: pprof
          path: %s
          if-no-files-found: error

  nightly-paranoid:
    name: nightly-paranoid
    if: github.event_name == 'schedule'
    runs-on: %s
    steps:
      - uses: %s
      - uses: %s
        with:
          go-version-file: %s
      - run: |
          go install %s
          go install %s
      - run: go test ./... -race -count=1
      - run: go build -o %s ./%s
      - run: ./%s check --profile=%s --format=%s
      - run: ./%s observe --format=%s

  release:
    name: release
    needs: check
    if: startsWith(github.ref, 'refs/tags/v')
    runs-on: %s
    permissions:
      contents: write
      id-token: write
    steps:
      - uses: %s
        with:
          fetch-depth: 0

      - uses: %s
        with:
          go-version-file: %s

      - name: Test
        run: go test ./... -race -count=1

      - name: Vet
        run: go vet ./...

      - uses: %s

      - uses: %s
        with:
          distribution: %s
          version: '%s'
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          COSIGN_YES: "true"
`, StatusCheckName, DefaultBranch, DefaultBranch, CIScheduleCron,
		StatusCheckName, CIRunnerOS,
		CheckoutAction, SetupGoAction, tokens.GoModFileName,
		tokens.InstallStaticcheck, tokens.InstallGovulncheck,
		tokens.ToolName, tokens.BinaryRel, tokens.CmdRel,
		tokens.BinaryRel, tokens.ProfileStrict, report.FormatLLM,
		tokens.BinaryRel, tokens.ProfileStandard, report.FormatSARIF, tokens.ToolName,
		tokens.BinaryRel, tokens.ToolName,
		UploadSARIFAction, tokens.ToolName,
		tokens.FitnessPkgRel, tokens.CPUProfileFile,
		UploadArtifactAction, tokens.CPUProfileFile,
		CIRunnerOS, CheckoutAction, SetupGoAction, tokens.GoModFileName,
		tokens.InstallStaticcheck, tokens.InstallGovulncheck,
		tokens.BinaryRel, tokens.CmdRel,
		tokens.BinaryRel, tokens.ProfileParanoid, report.FormatLLM, tokens.BinaryRel, report.FormatJSON,
		CIRunnerOS, CheckoutAction, SetupGoAction, tokens.GoModFileName,
		CosignInstallerAction,
		GoreleaserAction, GoreleaserDist, GoreleaserPin)
}

// ProtectMain enables branch protection via gh api using the injectable runner.
func (c *Client) ProtectMain(owner, repo string, stdout, stderr io.Writer) error {
	r := c.runner()
	ghPath, err := r.LookPath(GhBinary)
	if err != nil {
		return fmt.Errorf("protect-main: %s not installed", GhBinary)
	}
	args, body := ProtectMainArgs(owner, repo)
	return r.Run(ghPath, args, bytes.NewBufferString(body), stdout, stderr)
}

// ProtectionBody is the JSON body for PUT .../branches/<DefaultBranch>/protection.
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
		fmt.Sprintf("repos/%s/%s/branches/%s/protection", owner, repo, DefaultBranch),
		"-H", "Accept: application/vnd.github+json",
		"--input", "-",
	}
	return args, body
}

// ProtectMainCommand returns a copy-pasteable shell command for branch protection.
func ProtectMainCommand(owner, repo string) string {
	_, body := ProtectMainArgs(owner, repo)
	return fmt.Sprintf(
		`printf '%%s' '%s' | %s api -X PUT repos/%s/%s/branches/%s/protection -H "Accept: application/vnd.github+json" --input -`,
		body, GhBinary, owner, repo, DefaultBranch,
	)
}

// ParseOwnerRepo extracts owner/repo from a github.com remote URL.
func ParseOwnerRepo(remote string) (owner, repo string, err error) {
	remote = strings.TrimSpace(remote)
	remote = strings.TrimSuffix(remote, tokens.GitDir)
	switch {
	case strings.HasPrefix(remote, GitHubSSHPrefix):
		parts := strings.SplitN(strings.TrimPrefix(remote, GitHubSSHPrefix), "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("invalid ssh remote: %s", remote)
		}
		return parts[0], parts[1], nil
	case strings.Contains(remote, GitHubHTTPSMark):
		idx := strings.Index(remote, GitHubHTTPSMark)
		rest := remote[idx+len(GitHubHTTPSMark):]
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
