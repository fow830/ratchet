package gates

import (
	"context"
	"fmt"

	"github.com/fow830/ratchet/pkg/workspace"
)

// WorkspaceOptions configures monorepo gate runs.
type WorkspaceOptions struct {
	Root    string
	Profile string
	Runner  CommandRunner
}

// RunWorkspace runs gates for each discovered module.
func RunWorkspace(ctx context.Context, opts WorkspaceOptions) ([]Result, error) {
	mods, err := workspace.Discover(ctx, opts.Root)
	if err != nil {
		return nil, err
	}
	var results []Result
	for _, m := range mods {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		res, err := Run(ctx, Options{
			Root:       m.Root,
			Config:     m.Config,
			Profile:    opts.Profile,
			ConfigPath: m.ConfigPath,
			Runner:     opts.Runner,
		})
		if err != nil {
			return nil, fmt.Errorf("module %s: %w", m.Module, err)
		}
		if !res.OK {
			res.Failures = append([]Failure{{
				Gate:    "workspace",
				Message: fmt.Sprintf("module %s failed", m.Module),
			}}, res.Failures...)
		}
		results = append(results, res)
	}
	return results, nil
}

// WorkspaceOK reports whether all module results passed.
func WorkspaceOK(results []Result) bool {
	for _, r := range results {
		if !r.OK {
			return false
		}
	}
	return true
}
