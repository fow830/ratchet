// Package antidrift provides byte-level integrity checks for generated contracts.
package antidrift

import (
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
	Path     string
	Expected string
	Actual   string
}

// Diff is the result of Verify.
type Diff struct {
	Changed []ChangedFile
	Missing []string
	Extra   []string
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
}

// New creates an Engine rooted at dir.
func New(root string) *Engine {
	return &Engine{Root: root}
}

// LockFilePath returns the absolute path of ratchet.lock.
func (e *Engine) LockFilePath() string {
	return filepath.Join(e.Root, tokens.LockFileName)
}

// Lock computes SHA-256 hashes for the given relative paths and writes ratchet.lock.
func (e *Engine) Lock(relPaths []string) error {
	files := make(map[string]string, len(relPaths))
	for _, rel := range relPaths {
		rel = filepath.ToSlash(rel)
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
	if err := os.WriteFile(e.LockFilePath(), data, 0o644); err != nil {
		return fmt.Errorf("write lock: %w", err)
	}
	return nil
}

// Verify compares on-disk contract files against ratchet.lock.
// Changed/Missing come from lock entries; Extra comes from ContractFiles in
// ratchet.json that exist on disk but are absent from the lock.
func (e *Engine) Verify() (Diff, error) {
	data, err := os.ReadFile(e.LockFilePath())
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
		expected := lf.Files[rel]
		actual, err := e.hashFile(rel)
		if err != nil {
			if os.IsNotExist(err) {
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

	extra, err := e.findExtra(lf.Files)
	if err != nil {
		return Diff{}, err
	}
	diff.Extra = extra
	return diff, nil
}

func (e *Engine) findExtra(locked map[string]string) ([]string, error) {
	expected, err := e.declaredContracts()
	if err != nil {
		return nil, err
	}
	var extra []string
	seen := make(map[string]struct{})
	for _, rel := range expected {
		rel = filepath.ToSlash(rel)
		if _, ok := locked[rel]; ok {
			continue
		}
		if _, dup := seen[rel]; dup {
			continue
		}
		path := filepath.Join(e.Root, filepath.FromSlash(rel))
		info, statErr := os.Stat(path)
		if statErr != nil {
			if os.IsNotExist(statErr) {
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

func (e *Engine) declaredContracts() ([]string, error) {
	cfg, err := tokens.Load(e.Root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return cfg.ContractFiles, nil
}

func (e *Engine) hashFile(rel string) (string, error) {
	path := filepath.Join(e.Root, filepath.FromSlash(rel))
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
