// Package fitness provides AST-level architecture fitness functions.
package fitness

import (
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/fow830/ratchet/pkg/tokens"
)

// Violation describes an illegal cross-layer import.
type Violation struct {
	File          string `json:"file"`
	Line          int    `json:"line,omitempty"`
	ImportPath    string `json:"import_path"`
	ImporterPkg   string `json:"importer_pkg"`
	ImporterLayer string `json:"importer_layer"`
	ImportedLayer string `json:"imported_layer"`
}

func (v Violation) String() string {
	loc := v.File
	if v.Line > 0 {
		loc = fmt.Sprintf("%s:%d", v.File, v.Line)
	}
	return fmt.Sprintf(
		"%s: package %q (layer %q) must not import %q (layer %q)",
		loc, v.ImporterPkg, v.ImporterLayer, v.ImportPath, v.ImportedLayer,
	)
}

// Analyzer enforces layer dependency direction via go/ast.
type Analyzer struct {
	cfg tokens.Config
}

// NewAnalyzer constructs an Analyzer from SSOT config.
func NewAnalyzer(cfg tokens.Config) *Analyzer {
	return &Analyzer{cfg: cfg}
}

// Analyze walks root for .go files and returns layer violations.
func (a *Analyzer) Analyze(ctx context.Context, root string) ([]Violation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	var violations []Violation

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == "vendor" || name == ".git" || name == "testdata" || strings.HasPrefix(name, ".") {
				if path != root {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		rel, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		importerPkg := a.cfg.Module
		if rel != "." {
			importerPkg = a.cfg.Module + "/" + rel
		}
		importerLayer, ok := a.LayerOf(importerPkg)
		if !ok {
			return nil
		}

		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			importedLayer, ok := a.LayerOf(importPath)
			if !ok {
				continue
			}
			if !a.Allowed(importerLayer, importedLayer) {
				pos := fset.Position(imp.Pos())
				violations = append(violations, Violation{
					File:          pos.Filename,
					Line:          pos.Line,
					ImportPath:    importPath,
					ImporterPkg:   importerPkg,
					ImporterLayer: importerLayer,
					ImportedLayer: importedLayer,
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return violations, nil
}

// LayerOf resolves an import path to a configured layer (longest suffix wins).
func (a *Analyzer) LayerOf(importPath string) (string, bool) {
	type hit struct {
		suffix string
		layer  string
	}
	var best hit
	for suffix, layer := range a.cfg.Layers {
		if strings.HasSuffix(importPath, suffix) || strings.Contains(importPath, suffix+"/") {
			if len(suffix) > len(best.suffix) {
				best = hit{suffix: suffix, layer: layer}
			}
		}
	}
	if best.layer == "" {
		return "", false
	}
	return best.layer, true
}

// Allowed reports whether from→to is a permitted dependency edge.
func (a *Analyzer) Allowed(from, to string) bool {
	if from == to {
		return true
	}
	for _, edge := range a.cfg.AllowedEdges[from] {
		if edge == to {
			return true
		}
	}
	return false
}
