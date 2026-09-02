// Package antidrift provides byte-level integrity checks for generated contracts.
package antidrift

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fow830/ratchet/pkg/tokens"
)

// LockFile is the on-disk hash ledger for contract files.
type LockFile struct {
	Version int               `json:"version"`
	Files   map[string]string `json:"files"` // relative path -> sha256 hex
}

// ChangedFile describes a drifted contract.
type ChangedFile struct {
	Path     string `json:"path"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
}

// Diff is the result of Verify.
type Diff struct {
	Changed []ChangedFile `json:"changed"`
	Missing []string      `json:"missing"`
	Extra   []string      `json:"extra"`
}

// OK reports whether verification passed with no drift.
func (d Diff) OK() bool {
	return len(d.Changed) == 0 && len(d.Missing) == 0 && len(d.Extra) == 0
}

func (d Diff) String() string {
	if d.OK() {
		return "antidrift: ok"
	}
	var b strings.Builder
	b.WriteString("antidrift: drift detected\n")
	for _, c := range d.Changed {
		fmt.Fprintf(&b, "  changed %s\n    expected %s\n    actual   %s\n", c.Path, c.Expected, c.Actual)
	}
	for _, m := range d.Missing {
		fmt.Fprintf(&b, "  missing %s\n", m)
	}
	for _, e := range d.Extra {
		fmt.Fprintf(&b, "  extra   %s\n", e)
	}
	return b.String()
}

// Engine locks and verifies contract file hashes under Root.
type Engine struct {
	Root string
	// FS optional filesystem mock; nil uses the real OS disk.
	FS FileSystem
	// ConfigPath optional absolute/relative path to ratchet.json; empty uses Root default.
	ConfigPath string
}

// New creates an Engine rooted at dir.
func New(root string) *Engine {
	return &Engine{Root: root}
}

// LockFilePath returns the absolute path of ratchet.lock.
func (e *Engine) LockFilePath() string {
	return filepath.Join(e.Root, tokens.LockFileName)
}

func (e *Engine) configPath() string {
	if e.ConfigPath != "" {
		return e.ConfigPath
	}
	return filepath.Join(e.Root, tokens.ConfigFileName)
}

// Lock computes SHA-256 hashes for the given relative paths and writes ratchet.lock.
func (e *Engine) Lock(ctx context.Context, relPaths []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	sorted := append([]string(nil), relPaths...)
	for i := range sorted {
		sorted[i] = filepath.ToSlash(sorted[i])
	}
	sort.Strings(sorted)

	files := make(map[string]string, len(sorted))
	for _, rel := range sorted {
		if err := ctx.Err(); err != nil {
			return err
		}
		if rel == "" {
			continue
		}
		sum, err := e.hashFile(rel)
		if err != nil {
			return err
		}
		files[rel] = sum
	}
	lf := LockFile{Version: 1, Files: files}
	data, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal lock: %w", err)
	}
	data = append(data, '\n')
	if err := e.fileSystem().WriteFile(e.LockFilePath(), data, 0o644); err != nil {
		return fmt.Errorf("write lock: %w", err)
	}
	return nil
}

// Verify compares on-disk contract files against ratchet.lock.
func (e *Engine) Verify(ctx context.Context) (Diff, error) {
	if err := ctx.Err(); err != nil {
		return Diff{}, err
	}
	data, err := e.fileSystem().ReadFile(e.LockFilePath())
	if err != nil {
		return Diff{}, fmt.Errorf("read lock: %w", err)
	}
	var lf LockFile
	if err := json.Unmarshal(data, &lf); err != nil {
		return Diff{}, fmt.Errorf("parse lock: %w", err)
	}
	if lf.Files == nil {
		lf.Files = map[string]string{}
	}

	var diff Diff
	paths := make([]string, 0, len(lf.Files))
	for p := range lf.Files {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, rel := range paths {
		if err := ctx.Err(); err != nil {
			return Diff{}, err
		}
		expected := lf.Files[rel]
		actual, err := e.hashFile(rel)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				diff.Missing = append(diff.Missing, rel)
				continue
			}
			return Diff{}, err
		}
		if actual != expected {
			diff.Changed = append(diff.Changed, ChangedFile{
				Path:     rel,
				Expected: expected,
				Actual:   actual,
			})
		}
	}

	extra, err := e.findExtra(ctx, lf.Files)
	if err != nil {
		return Diff{}, err
	}
	diff.Extra = extra
	return diff, nil
}

func (e *Engine) findExtra(ctx context.Context, locked map[string]string) ([]string, error) {
	expected, err := e.declaredContracts(ctx)
	if err != nil {
		return nil, err
	}
	var extra []string
	seen := make(map[string]struct{})
	for _, rel := range expected {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rel = filepath.ToSlash(rel)
		if _, ok := locked[rel]; ok {
			continue
		}
		if _, dup := seen[rel]; dup {
			continue
		}
		path := filepath.Join(e.Root, filepath.FromSlash(rel))
		info, statErr := e.fileSystem().Stat(path)
		if statErr != nil {
			if errors.Is(statErr, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("stat %s: %w", rel, statErr)
		}
		if info.IsDir() {
			continue
		}
		seen[rel] = struct{}{}
		extra = append(extra, rel)
	}
	sort.Strings(extra)
	return extra, nil
}

func (e *Engine) declaredContracts(ctx context.Context) ([]string, error) {
	cfg, err := tokens.LoadFile(ctx, e.configPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if cfg.ContractFiles == nil {
		return nil, nil
	}
	return cfg.ContractFiles, nil
}

func (e *Engine) hashFile(rel string) (string, error) {
	path := filepath.Join(e.Root, filepath.FromSlash(rel))
	data, err := e.fileSystem().ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
