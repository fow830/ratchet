package antidrift

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fow830/ratchet/pkg/tokens"
)

// RenderFunc produces expected bytes for render-lock verification.
type RenderFunc func() ([]byte, error)

// LockAll locks all contract files using per-file SHA or render mode.
func (e *Engine) LockAll(ctx context.Context, cfg tokens.Config) error {
	locks := mergeLocks(cfg)
	paths := make([]string, 0, len(locks))
	for p := range locks {
		paths = append(paths, p)
	}
	if err := e.Lock(ctx, paths); err != nil {
		return err
	}
	for _, cl := range locks {
		if cl.Mode != tokens.LockModeRender {
			continue
		}
		sum, err := e.renderDigest(cl)
		if err != nil {
			return err
		}
		if err := e.patchLockEntry(ctx, cl.Path, sum); err != nil {
			return err
		}
	}
	return nil
}

// VerifyAll verifies SHA and render-mode contracts.
func (e *Engine) VerifyAll(ctx context.Context, cfg tokens.Config) (Diff, error) {
	diff, err := e.Verify(ctx)
	if err != nil {
		return Diff{}, err
	}
	locks := mergeLocks(cfg)
	for _, cl := range locks {
		if cl.Mode != tokens.LockModeRender {
			continue
		}
		expected, err := e.renderDigest(cl)
		if err != nil {
			return Diff{}, err
		}
		actual, err := e.hashFile(cl.Path)
		if err != nil {
			continue
		}
		if actual != expected {
			diff.Changed = append(diff.Changed, ChangedFile{
				Path:     cl.Path,
				Expected: expected,
				Actual:   actual,
			})
		}
	}
	return diff, nil
}

// AssertFileEqualsRender compares on-disk file with expected render bytes.
func AssertFileEqualsRender(root, rel string, render []byte) error {
	path := filepath.Join(root, filepath.FromSlash(rel))
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if string(data) != string(render) {
		return fmt.Errorf("render drift %s", rel)
	}
	return nil
}

func mergeLocks(cfg tokens.Config) map[string]tokens.ContractLock {
	out := map[string]tokens.ContractLock{}
	for _, p := range cfg.ContractFiles {
		out[p] = tokens.ContractLock{Path: p, Mode: tokens.LockModeSHA}
	}
	for _, cl := range cfg.ContractLocks {
		mode := cl.Mode
		if mode == "" {
			mode = tokens.LockModeSHA
		}
		out[cl.Path] = tokens.ContractLock{
			Path:          cl.Path,
			Mode:          mode,
			RenderPackage: cl.RenderPackage,
			RenderFunc:    cl.RenderFunc,
		}
	}
	return out
}

func (e *Engine) renderDigest(cl tokens.ContractLock) (string, error) {
	data, err := e.renderBytes(cl)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func (e *Engine) renderBytes(cl tokens.ContractLock) ([]byte, error) {
	key := cl.RenderPackage + "." + cl.RenderFunc
	if e.Renderers != nil {
		if fn, ok := e.Renderers[key]; ok {
			return fn()
		}
	}
	return nil, fmt.Errorf("render func not registered: %s", key)
}

func (e *Engine) patchLockEntry(ctx context.Context, rel, sum string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := e.fileSystem().ReadFile(e.LockFilePath())
	if err != nil {
		return err
	}
	var lf LockFile
	if err := json.Unmarshal(data, &lf); err != nil {
		return err
	}
	if lf.Files == nil {
		lf.Files = map[string]string{}
	}
	lf.Files[filepath.ToSlash(rel)] = sum
	out, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return e.fileSystem().WriteFile(e.LockFilePath(), out, tokens.FileModeFile)
}
