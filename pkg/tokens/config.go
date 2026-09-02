// Package tokens holds the pure-Go SSOT configuration for ratchet.
package tokens

import "io/fs"

// Published identity — single source; all other artifact names derive from ToolName.
const (
	ToolName   = "ratchet"
	ModulePath = "github.com/fow830/" + ToolName
)

// On-disk artifact names and paths derived from ToolName.
const (
	ConfigFileName = ToolName + ".json"
	LockFileName   = ToolName + ".lock"
	CursorRules    = ".cursorrules"
	ClaudeSkillRel = ".claude/skills/" + ToolName + ".md"
	BinaryRel      = "bin/" + ToolName
	CmdRel         = "cmd/" + ToolName
	GoModFileName  = "go.mod"
	PreCommitRel   = ".git/hooks/pre-commit"
)

// Default clean-architecture layer names and import-path suffixes.
const (
	LayerDomain   = "domain"
	LayerUsecase  = "usecase"
	LayerDelivery = "delivery"

	SuffixDomain   = "/" + LayerDomain
	SuffixUsecase  = "/" + LayerUsecase
	SuffixDelivery = "/" + LayerDelivery
)

// Filesystem and lock ledger constants.
const (
	FileModeFile fs.FileMode = 0o644
	FileModeExec fs.FileMode = 0o755
	FileModeDir  fs.FileMode = 0o755

	LockVersion = 1
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
			SuffixDomain:   LayerDomain,
			SuffixUsecase:  LayerUsecase,
			SuffixDelivery: LayerDelivery,
		},
		AllowedEdges: map[string][]string{
			LayerDomain:   {},
			LayerUsecase:  {LayerDomain},
			LayerDelivery: {LayerUsecase, LayerDomain},
		},
		ContractFiles: []string{
			CursorRules,
			ConfigFileName,
		},
	}
}
