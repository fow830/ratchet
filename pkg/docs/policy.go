// Package docs enforces prose documentation allowlists.
package docs

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/fow830/ratchet/pkg/tokens"
)

// Violation is an unauthorized prose file on disk.
type Violation struct {
	Path string `json:"path"`
}

func (v Violation) String() string {
	return fmt.Sprintf("docs policy: unauthorized prose file %q", v.Path)
}

var proseExt = map[string]struct{}{
	".md":  {},
	".txt": {},
}

var ignoredDirs = map[string]struct{}{
	tokens.DirVendor:    {},
	tokens.GitDir:       {},
	tokens.DirTestdata:  {},
	tokens.DirClaude:    {},
	tokens.DirGitHub:    {},
	tokens.DirDist:      {},
	tokens.DirBin:       {},
	tokens.DirGenerated: {},
	tokens.DirExamples:  {},
	tokens.DirCoverage:  {},
}

// Check walks root and fails on prose files not in cfg.AllowedProseDocs.
func Check(ctx context.Context, root string, cfg tokens.Config) ([]Violation, error) {
	if len(cfg.AllowedProseDocs) == 0 {
		return nil, nil
	}
	allowed := map[string]struct{}{}
	for _, p := range cfg.AllowedProseDocs {
		allowed[filepath.ToSlash(p)] = struct{}{}
	}
	var out []Violation
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() {
			if shouldSkipDir(path, root, d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if _, ok := proseExt[ext]; !ok {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if _, ok := allowed[rel]; ok {
			return nil
		}
		out = append(out, Violation{Path: rel})
		return nil
	})
	return out, err
}

func shouldSkipDir(path, root, name string) bool {
	if path == root {
		return false
	}
	if _, skip := ignoredDirs[name]; skip {
		return true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if _, skip := ignoredDirs[part]; skip {
			return true
		}
	}
	return false
}
