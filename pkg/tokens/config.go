// Package tokens holds the pure-Go SSOT configuration for ratchet.
package tokens

// On-disk artifact names (single place — no scattered string literals).
const (
	ConfigFileName = "ratchet.json"
	LockFileName   = "ratchet.lock"
	CursorRules    = ".cursorrules"
	ClaudeSkillRel = ".claude/skills/ratchet.md"
)

// Config is the root SSOT for a ratchet-managed repository.
type Config struct {
	// Module is the Go module path (from go.mod).
	Module string `json:"module"`

	// Layers maps import-path suffixes to architectural layers.
	// Example: "/domain" matches ".../internal/domain".
	Layers map[string]string `json:"layers"`

	// AllowedEdges lists permitted layer→layer imports.
	// Key is importer layer; value is allowed dependency layers.
	AllowedEdges map[string][]string `json:"allowed_edges"`

	// ContractFiles are generated files locked by antidrift.
	ContractFiles []string `json:"contract_files"`
}

// DefaultConfig returns a clean-architecture baseline for module.
func DefaultConfig(module string) Config {
	return Config{
		Module: module,
		Layers: map[string]string{
			"/domain":   "domain",
			"/usecase":  "usecase",
			"/delivery": "delivery",
		},
		AllowedEdges: map[string][]string{
			"domain":   {},
			"usecase":  {"domain"},
			"delivery": {"usecase", "domain"},
		},
		ContractFiles: []string{
			CursorRules,
			ConfigFileName,
		},
	}
}
