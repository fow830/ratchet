package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	gha "github.com/fow830/ratchet/pkg/github"
	"github.com/fow830/ratchet/pkg/tokens"
)

// Check is one diagnostic item.
type Check struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Details string `json:"details,omitempty"`
}

// Report aggregates doctor results.
type Report struct {
	Checks []Check `json:"checks"`
}

// Healthy reports whether all checks passed.
func (r Report) Healthy() bool {
	for _, c := range r.Checks {
		if !c.OK {
			return false
		}
	}
	return true
}

// Run executes setup diagnostics.
func Run(ctx context.Context, root string, cfg tokens.Config, configPath string) (Report, error) {
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	var checks []Check

	checks = append(checks, checkFile(filepath.Join(root, tokens.GoModFileName), tokens.GoModFileName))
	if configPath == "" {
		configPath = filepath.Join(root, tokens.ConfigFileName)
	}
	checks = append(checks, checkFile(configPath, tokens.ConfigFileName))
	checks = append(checks, checkFile(filepath.Join(root, tokens.LockFileName), tokens.LockFileName))

	wfPath := filepath.Join(root, filepath.FromSlash(gha.WorkflowRel))
	if _, err := os.Stat(wfPath); err != nil {
		checks = append(checks, Check{Name: CheckNameWorkflow, OK: true, Details: "optional: not present"})
	} else {
		checks = append(checks, Check{Name: CheckNameWorkflow, OK: true})
	}

	if err := tokens.Validate(cfg); err != nil {
		checks = append(checks, Check{Name: CheckNameConfigValidate, OK: false, Details: err.Error()})
	} else {
		checks = append(checks, Check{Name: CheckNameConfigValidate, OK: true})
	}

	checks = append(checks, checkSchema(root))

	gitDir := filepath.Join(root, tokens.GitDir)
	if info, err := os.Stat(gitDir); err != nil || !info.IsDir() {
		checks = append(checks, Check{Name: CheckNameGit, OK: false, Details: "not a git repository"})
	} else {
		checks = append(checks, Check{Name: CheckNameGit, OK: true})
	}

	return Report{Checks: checks}, nil
}

func checkSchema(root string) Check {
	schemaPath := filepath.Join(root, filepath.FromSlash(tokens.SchemaRel))
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return Check{Name: CheckNameJSONSchema, OK: true, Details: "optional: not present"}
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return Check{Name: CheckNameJSONSchema, OK: false, Details: "invalid json: " + err.Error()}
	}
	if raw["type"] == nil && raw["properties"] == nil && raw["$schema"] == nil {
		return Check{Name: CheckNameJSONSchema, OK: false, Details: "schema missing type/properties/$schema"}
	}
	return Check{Name: CheckNameJSONSchema, OK: true}
}

func checkFile(path, name string) Check {
	if _, err := os.Stat(path); err != nil {
		return Check{Name: name, OK: false, Details: fmt.Sprintf("missing: %s", path)}
	}
	return Check{Name: name, OK: true}
}
