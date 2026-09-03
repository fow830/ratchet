// Package hooks installs local git soft-friction hooks.
package hooks

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"github.com/fow830/ratchet/pkg/report"
	"github.com/fow830/ratchet/pkg/tokens"
)

const (
	preCommitRel = tokens.PreCommitRel
	commitMsgRel = tokens.CommitMsgRel
)

// FileSystem abstracts disk IO for install tests.
type FileSystem interface {
	Stat(name string) (fs.FileInfo, error)
	MkdirAll(path string, perm fs.FileMode) error
	WriteFile(name string, data []byte, perm fs.FileMode) error
}

// OSFileSystem is the default real-disk implementation.
type OSFileSystem struct{}

func (OSFileSystem) Stat(name string) (fs.FileInfo, error) { return os.Stat(name) }
func (OSFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}
func (OSFileSystem) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(name, data, perm)
}

// Installer installs git hooks using an injectable filesystem.
type Installer struct {
	FS FileSystem
}

// NewInstaller returns an Installer with OS filesystem defaults.
func NewInstaller() *Installer {
	return &Installer{FS: OSFileSystem{}}
}

func (i *Installer) fs() FileSystem {
	if i.FS != nil {
		return i.FS
	}
	return OSFileSystem{}
}

// InstallOptions configures hook behavior.
type InstallOptions struct {
	LRTVerify    bool
	ContractsDir string // relative contracts path for LRT grep; empty = default
}

// Install writes an executable pre-commit hook under root/.git/hooks.
func Install(root string) (string, error) {
	return NewInstaller().Install(root, InstallOptions{})
}

// Install writes pre-commit and optionally commit-msg hooks.
func (i *Installer) Install(root string, opts InstallOptions) (string, error) {
	fsys := i.fs()
	gitDir := filepath.Join(root, tokens.GitDir)
	info, err := fsys.Stat(gitDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("hooks: %s is not a git repository (%s missing)", root, tokens.GitDir)
		}
		return "", fmt.Errorf("hooks: stat %s: %w", tokens.GitDir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("hooks: %s/%s is not a directory", root, tokens.GitDir)
	}

	path := filepath.Join(root, filepath.FromSlash(preCommitRel))
	if err := fsys.MkdirAll(filepath.Dir(path), tokens.FileModeDir); err != nil {
		return "", fmt.Errorf("hooks: mkdir: %w", err)
	}
	if err := fsys.WriteFile(path, []byte(preCommitScript()), tokens.FileModeExec); err != nil {
		return "", fmt.Errorf("hooks: write pre-commit: %w", err)
	}
	if opts.LRTVerify {
		cm := filepath.Join(root, filepath.FromSlash(commitMsgRel))
		cdir := opts.ContractsDir
		if cdir == "" {
			cdir = tokens.ContractsDirDefault
		}
		if err := fsys.WriteFile(cm, []byte(commitMsgScript(cdir)), tokens.FileModeExec); err != nil {
			return "", fmt.Errorf("hooks: write commit-msg: %w", err)
		}
	}
	return path, nil
}

func preCommitScript() string {
	format := report.FormatLLM
	bin := tokens.BinaryRel
	tool := tokens.ToolName
	return fmt.Sprintf(`#!/bin/sh
# Installed by %[1]s init-hooks (soft friction; CI is the hard constraint).
set -e
if command -v %[1]s >/dev/null 2>&1; then
  %[1]s check --format=%[2]s
elif [ -x ./%[3]s ]; then
  ./%[3]s check --format=%[2]s
elif [ -x ./%[1]s ]; then
  ./%[1]s check --format=%[2]s
else
  echo "RULE_VIOLATION: HookSetup"
  echo "FILE: %[5]s"
  echo "DETAILS: %[1]s binary not found in PATH, ./%[3]s, or ./%[1]s"
  echo "ACTION_REQUIRED: Install %[1]s or run: go build -o %[3]s ./%[4]s"
  exit 1
fi
`, tool, format, bin, tokens.CmdRel, tokens.PreCommitRel)
}

func commitMsgScript(contractsDir string) string {
	pat := fmt.Sprintf(`(^|/)(%s|%s|%s/|%s$)`,
		regexp.QuoteMeta(tokens.ConfigFileName),
		regexp.QuoteMeta(tokens.LockFileName),
		regexp.QuoteMeta(contractsDir),
		regexp.QuoteMeta(tokens.CursorRules),
	)
	return fmt.Sprintf(`#!/bin/sh
# LRT-VERIFY: require marker when staged contract files change.
set -e
MSG_FILE="$1"
if git diff --cached --name-only | grep -Eq '%s'; then
  if ! grep -q 'LRT-VERIFY' "$MSG_FILE"; then
    echo "RULE_VIOLATION: LRTVerify"
    echo "FILE: commit-msg"
    echo "DETAILS: contract files staged but commit message lacks LRT-VERIFY"
    echo "ACTION_REQUIRED: Add a line containing LRT-VERIFY to the commit message"
    exit 1
  fi
fi
`, pat)
}
