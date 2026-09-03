// Package graph exports import dependency graphs.
package graph

import (
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fow830/ratchet/pkg/fitness"
	"github.com/fow830/ratchet/pkg/tokens"
)

// Edge is a directed import edge between layers.
type Edge struct {
	FromLayer string
	ToLayer   string
	FromPkg   string
	ToPkg     string
}

// ExportMermaid renders layer import edges as a mermaid graph.
func ExportMermaid(ctx context.Context, root string, cfg tokens.Config) (string, error) {
	edges, err := CollectEdges(ctx, root, cfg)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("graph TD\n")
	for _, e := range edges {
		fmt.Fprintf(&b, "  %s[%s] --> %s[%s]\n", sanitize(e.FromLayer), e.FromPkg, sanitize(e.ToLayer), e.ToPkg)
	}
	return b.String(), nil
}

// CollectEdges gathers cross-layer imports.
func CollectEdges(ctx context.Context, root string, cfg tokens.Config) ([]Edge, error) {
	an := fitness.NewAnalyzer(cfg)
	fset := token.NewFileSet()
	type key struct{ from, to string }
	seen := map[key]Edge{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == tokens.DirVendor || name == tokens.GitDir || name == tokens.DirTestdata || strings.HasPrefix(name, ".") {
				if path != root {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(path, tokens.GoFileExt) || strings.HasSuffix(path, tokens.GoTestSuffix) {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, filepath.Dir(path))
		rel = filepath.ToSlash(rel)
		importer := cfg.Module
		if rel != "." {
			importer = cfg.Module + "/" + rel
		}
		fromLayer, ok := an.LayerOf(importer)
		if !ok {
			return nil
		}
		for _, imp := range file.Imports {
			toPath := strings.Trim(imp.Path.Value, `"`)
			toLayer, ok := an.LayerOf(toPath)
			if !ok || fromLayer == toLayer {
				continue
			}
			k := key{from: fromLayer, to: toLayer}
			if _, dup := seen[k]; dup {
				continue
			}
			seen[k] = Edge{FromLayer: fromLayer, ToLayer: toLayer, FromPkg: importer, ToPkg: toPath}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]Edge, 0, len(seen))
	for _, e := range seen {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].FromLayer != out[j].FromLayer {
			return out[i].FromLayer < out[j].FromLayer
		}
		return out[i].ToLayer < out[j].ToLayer
	})
	return out, nil
}

func sanitize(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "-", "_"), ".", "_")
}
