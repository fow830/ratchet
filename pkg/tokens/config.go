// Package tokens holds the pure-Go SSOT configuration for ratchet.
package tokens

// On-disk artifact names and published identity (single place — no scattered literals).
const (
	ConfigFileName = "ratchet.json"
	LockFileName   = "ratchet.lock"
	CursorRules    = ".cursorrules"
	ClaudeSkillRel = ".claude/skills/ratchet.md"

	// ModulePath must match go.mod.
	ModulePath = "github.com/fow830/ratchet"
	// ToolName is the CLI / SARIF driver name.
	ToolName = "ratchet"
	// BinaryRel is the default local build output path used by CI and hooks.
	BinaryRel = "bin/" + ToolName
)

// ModuleHTTPSURL returns the canonical https:// URL for ModulePath.
func ModuleHTTPSURL() string {
	return "https://" + ModulePath
}

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
