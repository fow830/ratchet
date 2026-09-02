// Package hooks installs local git soft-friction hooks.
package hooks

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/fow830/ratchet/pkg/report"
	"github.com/fow830/ratchet/pkg/tokens"
)

const preCommitRel = tokens.PreCommitRel

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

// Install writes an executable pre-commit hook under root/.git/hooks.
func Install(root string) (string, error) {
	return NewInstaller().Install(root)
}

// Install writes an executable pre-commit hook under root/.git/hooks.
func (i *Installer) Install(root string) (string, error) {
	fsys := i.fs()
	gitDir := filepath.Join(root, tokens.GitDir)
	info, err := fsys.Stat(gitDir)
	if err != nil {
		if os.IsNotExist(err) {
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
