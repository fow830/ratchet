package fitness

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AnalyzeAll runs layer fitness plus extended architecture rules.
func (a *Analyzer) AnalyzeAll(ctx context.Context, root string) ([]Violation, error) {
	base, err := a.Analyze(ctx, root)
	if err != nil {
		return nil, err
	}
	forbidden, err := a.checkForbidden(ctx, root)
	if err != nil {
		return nil, err
	}
	base = append(base, forbidden...)
	boundary, err := a.checkBoundaries(ctx, root)
	if err != nil {
		return nil, err
	}
	base = append(base, boundary...)
	return dedupeViolations(base), nil
}

// CheckLayerPaths verifies configured layer directories exist.
func (a *Analyzer) CheckLayerPaths(ctx context.Context, root string) ([]Violation, error) {
	var out []Violation
	for _, rel := range a.cfg.LayerPaths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		path := filepath.Join(root, filepath.FromSlash(rel))
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				out = append(out, Violation{
					File:          rel,
					ImporterLayer: "layer-path",
					ImportPath:    rel,
					ImportedLayer: "missing",
				})
				continue
			}
			return nil, err
		}
		if !info.IsDir() {
			out = append(out, Violation{
				File:          rel,
				ImporterLayer: "layer-path",
				ImportPath:    rel,
				ImportedLayer: "not-directory",
			})
		}
	}
	return out, nil
}

func (a *Analyzer) checkForbidden(ctx context.Context, root string) ([]Violation, error) {
	if len(a.cfg.ForbiddenEdges) == 0 {
		return nil, nil
	}
	deny := make(map[string]map[string]struct{})
	for _, fe := range a.cfg.ForbiddenEdges {
		if deny[fe.From] == nil {
			deny[fe.From] = map[string]struct{}{}
		}
		deny[fe.From][fe.To] = struct{}{}
	}
	all, err := a.Analyze(ctx, root)
	if err != nil {
		return nil, err
	}
	var extra []Violation
	for _, v := range all {
		if _, blocked := deny[v.ImporterLayer][v.ImportedLayer]; blocked {
			extra = append(extra, v)
		}
	}
	// Forbidden edges also apply when allowed_edges would permit the import.
	permitted, err := a.scanAllImports(ctx, root)
	if err != nil {
		return nil, err
	}
	for _, v := range permitted {
		if _, blocked := deny[v.ImporterLayer][v.ImportedLayer]; blocked {
			extra = append(extra, v)
		}
	}
	return dedupeViolations(extra), nil
}

func (a *Analyzer) checkBoundaries(ctx context.Context, root string) ([]Violation, error) {
	if len(a.cfg.BoundaryRules) == 0 {
		return nil, nil
	}
	all, err := a.scanAllImports(ctx, root)
	if err != nil {
		return nil, err
	}
	var out []Violation
	for _, v := range all {
		for _, br := range a.cfg.BoundaryRules {
			if globMatch(br.ImporterGlob, v.File) && globMatch(br.ImportGlob, v.ImportPath) {
				out = append(out, v)
				break
			}
		}
	}
	return out, nil
}

func (a *Analyzer) scanAllImports(ctx context.Context, root string) ([]Violation, error) {
	// Temporarily allow all edges to collect every cross-layer import.
	orig := a.cfg.AllowedEdges
	wide := map[string][]string{}
	uniq := map[string]struct{}{}
	for _, l := range a.cfg.Layers {
		uniq[l] = struct{}{}
	}
	for l := range uniq {
		var all []string
		for x := range uniq {
			all = append(all, x)
		}
		wide[l] = all
	}
	a.cfg.AllowedEdges = wide
	violations, err := a.Analyze(ctx, root)
	a.cfg.AllowedEdges = orig
	if err != nil {
		return nil, err
	}
	return violations, nil
}

func dedupeViolations(in []Violation) []Violation {
	seen := map[string]struct{}{}
	var out []Violation
	for _, v := range in {
		key := fmt.Sprintf("%s:%d:%s:%s", v.File, v.Line, v.ImportPath, v.ImporterLayer)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, v)
	}
	return out
}

func globMatch(pattern, value string) bool {
	pattern = filepath.ToSlash(pattern)
	value = filepath.ToSlash(value)
	if strings.Contains(pattern, "**") {
		parts := strings.Split(pattern, "**")
		if len(parts) == 2 {
			prefix := strings.TrimSuffix(parts[0], "/")
			suffix := strings.TrimPrefix(parts[1], "/")
			if prefix != "" && !strings.HasPrefix(value, prefix) {
				return false
			}
			if suffix != "" && !strings.Contains(value, suffix) {
				return false
			}
			return true
		}
	}
	trimmed := strings.Trim(pattern, "*")
	return trimmed != "" && strings.Contains(value, trimmed)
}
