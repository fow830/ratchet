// Package tokens holds the pure-Go SSOT configuration for ratchet.
package tokens

// Config is the root SSOT for a ratchet-managed repository.
type Config struct {
	// Module is the Go module path (e.g. github.com/fow830/ratchet).
	Module string `json:"module"`

	// Layers maps package path suffixes to architectural layers.
	// Example: "internal/domain" -> "domain"
	Layers map[string]string `json:"layers"`

	// AllowedEdges lists permitted layer→layer imports.
	// Key is importer layer; value is the set of allowed dependency layers.
	AllowedEdges map[string][]string `json:"allowed_edges"`

	// ContractFiles are generated files whose content is locked by antidrift.
	ContractFiles []string `json:"contract_files"`
}

// DefaultConfig returns a sensible clean-architecture baseline.
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
			".cursorrules",
			"ratchet.go",
		},
	}
}
