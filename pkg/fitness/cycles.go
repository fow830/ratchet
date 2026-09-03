package fitness

import (
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fow830/ratchet/pkg/tokens"
)

// DetectCycles finds import cycles among packages under modulePrefix.
func DetectCycles(ctx context.Context, root, modulePrefix string) ([]string, error) {
	graph, err := buildImportGraph(ctx, root, modulePrefix)
	if err != nil {
		return nil, err
	}
	return findCycles(graph), nil
}

func buildImportGraph(ctx context.Context, root, modulePrefix string) (map[string][]string, error) {
	fset := token.NewFileSet()
	graph := map[string][]string{}
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
		rel, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		pkg := modulePrefix
		if rel != "." {
			pkg = modulePrefix + "/" + rel
		}
		for _, imp := range file.Imports {
			to := strings.Trim(imp.Path.Value, `"`)
			if !strings.HasPrefix(to, modulePrefix) {
				continue
			}
			graph[pkg] = append(graph[pkg], to)
		}
		if _, ok := graph[pkg]; !ok {
			graph[pkg] = nil
		}
		return nil
	})
	return graph, err
}

func findCycles(graph map[string][]string) []string {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := map[string]int{}
	var stack []string
	var cycles []string
	seenCycle := map[string]struct{}{}

	var dfs func(string)
	dfs = func(n string) {
		color[n] = gray
		stack = append(stack, n)
		for _, to := range graph[n] {
			switch color[to] {
			case white:
				dfs(to)
			case gray:
				// cycle
				i := 0
				for i < len(stack) && stack[i] != to {
					i++
				}
				cyc := append([]string(nil), stack[i:]...)
				cyc = append(cyc, to)
				key := strings.Join(cyc, "->")
				if _, ok := seenCycle[key]; !ok {
					seenCycle[key] = struct{}{}
					cycles = append(cycles, key)
				}
			}
		}
		stack = stack[:len(stack)-1]
		color[n] = black
	}

	nodes := make([]string, 0, len(graph))
	for n := range graph {
		nodes = append(nodes, n)
	}
	sort.Strings(nodes)
	for _, n := range nodes {
		if color[n] == white {
			dfs(n)
		}
	}
	return cycles
}

// CheckExternal fails when non-stdlib imports are outside AllowedExternal/module.
func (a *Analyzer) CheckExternal(ctx context.Context, root string) ([]Violation, error) {
	if len(a.cfg.AllowedExternal) == 0 && len(a.cfg.AllowedModules) == 0 {
		return nil, nil
	}
	fset := token.NewFileSet()
	var out []Violation
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
		for _, imp := range file.Imports {
			ip := strings.Trim(imp.Path.Value, `"`)
			if isStdlib(ip) || strings.HasPrefix(ip, a.cfg.Module) {
				continue
			}
			if allowedPrefix(ip, a.cfg.AllowedExternal) || allowedPrefix(ip, a.cfg.AllowedModules) {
				continue
			}
			pos := fset.Position(imp.Pos())
			out = append(out, Violation{
				File:          pos.Filename,
				Line:          pos.Line,
				ImportPath:    ip,
				ImporterLayer: "external",
				ImportedLayer: "forbidden",
			})
		}
		return nil
	})
	return out, err
}

// CheckTestImports enforces TestForbiddenEdges on *_test.go files.
func (a *Analyzer) CheckTestImports(ctx context.Context, root string) ([]Violation, error) {
	if len(a.cfg.TestForbiddenEdges) == 0 {
		return nil, nil
	}
	deny := map[string]map[string]struct{}{}
	for _, fe := range a.cfg.TestForbiddenEdges {
		if deny[fe.From] == nil {
			deny[fe.From] = map[string]struct{}{}
		}
		deny[fe.From][fe.To] = struct{}{}
	}
	fset := token.NewFileSet()
	var out []Violation
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
		if !strings.HasSuffix(path, tokens.GoTestSuffix) {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, filepath.Dir(path))
		rel = filepath.ToSlash(rel)
		importer := a.cfg.Module
		if rel != "." {
			importer = a.cfg.Module + "/" + rel
		}
		fromLayer, ok := a.LayerOf(importer)
		if !ok {
			return nil
		}
		for _, imp := range file.Imports {
			ip := strings.Trim(imp.Path.Value, `"`)
			toLayer, ok := a.LayerOf(ip)
			if !ok {
				continue
			}
			if _, blocked := deny[fromLayer][toLayer]; blocked {
				pos := fset.Position(imp.Pos())
				out = append(out, Violation{
					File:          pos.Filename,
					Line:          pos.Line,
					ImportPath:    ip,
					ImporterPkg:   importer,
					ImporterLayer: fromLayer,
					ImportedLayer: toLayer,
				})
			}
		}
		return nil
	})
	return out, err
}

func allowedPrefix(ip string, prefixes []string) bool {
	for _, p := range prefixes {
		if p != "" && strings.HasPrefix(ip, p) {
			return true
		}
	}
	return false
}

func isStdlib(ip string) bool {
	if !strings.Contains(ip, ".") {
		return true
	}
	// rough: first path element has no domain
	first := strings.SplitN(ip, "/", 2)[0]
	return !strings.Contains(first, ".")
}

// FormatCycle returns a cycle violation message.
func FormatCycle(cycle string) string {
	return fmt.Sprintf("import cycle: %s", cycle)
}
