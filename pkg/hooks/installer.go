// Package hooks installs local git soft-friction hooks.
package hooks

import (
	"fmt"
	"os"
	"path/filepath"
)

const preCommitRel = ".git/hooks/pre-commit"

const preCommitScript = `#!/bin/sh
# Installed by ratchet init-hooks (soft friction; CI is the hard constraint).
set -e

if command -v ratchet >/dev/null 2>&1; then
  ratchet check --format=llm
elif [ -x ./bin/ratchet ]; then
  ./bin/ratchet check --format=llm
elif [ -x ./ratchet ]; then
  ./ratchet check --format=llm
else
  echo "RULE_VIOLATION: HookSetup"
  echo "FILE: .git/hooks/pre-commit"
  echo "DETAILS: ratchet binary not found in PATH, ./bin/ratchet, or ./ratchet"
  echo "ACTION_REQUIRED: Install ratchet or run: go build -o bin/ratchet ./cmd/ratchet"
  exit 1
fi
`

// Install writes an executable pre-commit hook under root/.git/hooks.
func Install(root string) (string, error) {
	gitDir := filepath.Join(root, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("hooks: %s is not a git repository (.git missing)", root)
		}
		return "", fmt.Errorf("hooks: stat .git: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("hooks: %s/.git is not a directory", root)
	}

	path := filepath.Join(root, filepath.FromSlash(preCommitRel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("hooks: mkdir: %w", err)
	}
	if err := os.WriteFile(path, []byte(preCommitScript), 0o755); err != nil {
		return "", fmt.Errorf("hooks: write pre-commit: %w", err)
	}
	return path, nil
}
