package tokens

import (
	"fmt"
	"strings"
)

// Validate checks config invariants before use.
func Validate(cfg Config) error {
	if strings.TrimSpace(cfg.Module) == "" {
		return fmt.Errorf("config: module is required")
	}
	if len(cfg.Layers) == 0 {
		return fmt.Errorf("config: layers must not be empty")
	}
	if cfg.AllowedEdges == nil {
		return fmt.Errorf("config: allowed_edges is required")
	}
	for _, fe := range cfg.ForbiddenEdges {
		if fe.From == "" || fe.To == "" {
			return fmt.Errorf("config: forbidden_edges entry must have from and to")
		}
	}
	for _, br := range cfg.BoundaryRules {
		if br.ImporterGlob == "" || br.ImportGlob == "" {
			return fmt.Errorf("config: boundary_rules entry must have importer_glob and import_glob")
		}
	}
	for _, cl := range cfg.ContractLocks {
		if cl.Path == "" {
			return fmt.Errorf("config: contract_locks path is required")
		}
		mode := cl.Mode
		if mode == "" {
			mode = LockModeSHA
		}
		if mode != LockModeSHA && mode != LockModeRender {
			return fmt.Errorf("config: contract_locks[%s] invalid mode %q", cl.Path, cl.Mode)
		}
		if mode == LockModeRender && (cl.RenderPackage == "" || cl.RenderFunc == "") {
			return fmt.Errorf("config: render lock for %s requires render_package and render_func", cl.Path)
		}
	}
	if cfg.Profile != "" {
		found := false
		for _, p := range AllProfiles() {
			if p == cfg.Profile {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("config: unknown profile %q", cfg.Profile)
		}
	}
	if err := ValidatePreset(cfg.Preset); err != nil {
		return err
	}
	if cfg.Quality.MutationMinPct < 0 || cfg.Quality.MutationMinPct > 100 {
		return fmt.Errorf("config: mutation_min_pct must be 0-100")
	}
	if cfg.Quality.CoverageMinPct < 0 || cfg.Quality.CoverageMinPct > 100 {
		return fmt.Errorf("config: coverage_min_pct must be 0-100")
	}
	return nil
}
